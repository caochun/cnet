#!/bin/bash

# 启动两个CNET Agent，其中一个向另一个注册

set -e

echo "🚀 启动两个CNET Agent演示"
echo "========================="
echo ""

# 停止任何现有的agent
echo "🛑 停止现有agent..."
pkill -f cnet-agent 2>/dev/null || true
sleep 2

# 构建agent
echo "📦 构建CNET Agent..."
make build

# 创建Agent 1配置 (发现服务器)
echo "📝 创建Agent 1配置 (发现服务器)..."
cat > config_agent1.yaml << EOF
agent:
  address: "0.0.0.0"
  port: 8080
  node_id: "discovery-server"
  node_name: "Discovery Server"
  region: "us-west"
  datacenter: "dc1"
  heartbeat: "30s"

logging:
  level: "info"
  format: "json"

discovery:
  enabled: true
  servers: []  # 作为发现服务器，不向其他服务器注册
  timeout: "5s"
  retry: 3

resources:
  cpu: true
  memory: true
  disk: true
  network: true
  interval: "10s"

tasks:
  max_concurrent: 10
  timeout: "5m"
  cleanup: true
EOF

# 创建Agent 2配置 (工作节点，向Agent 1注册)
echo "📝 创建Agent 2配置 (工作节点)..."
cat > config_agent2.yaml << EOF
agent:
  address: "0.0.0.0"
  port: 8081
  node_id: "worker-node"
  node_name: "Worker Node"
  region: "us-west"
  datacenter: "dc1"
  heartbeat: "30s"

logging:
  level: "info"
  format: "json"

discovery:
  enabled: true
  servers:
    - "localhost:8080"  # 向Agent 1注册
  timeout: "5s"
  retry: 3

resources:
  cpu: true
  memory: true
  disk: true
  network: true
  interval: "10s"

tasks:
  max_concurrent: 10
  timeout: "5m"
  cleanup: true
EOF

echo "✅ 配置文件创建完成"
echo ""

# 启动Agent 1 (发现服务器)
echo "🚀 启动Agent 1 (发现服务器) on port 8080..."
./bin/cnet-agent -config config_agent1.yaml > agent1.log 2>&1 &
AGENT1_PID=$!

# 等待Agent 1启动
echo "⏳ 等待Agent 1启动..."
for i in {1..15}; do
    if curl -s http://localhost:8080/api/health > /dev/null 2>&1; then
        echo "✅ Agent 1 启动成功"
        break
    fi
    if [ $i -eq 15 ]; then
        echo "❌ Agent 1 启动失败"
        echo "查看日志:"
        cat agent1.log
        kill $AGENT1_PID 2>/dev/null || true
        exit 1
    fi
    echo "   尝试 $i/15..."
    sleep 1
done

# 启动Agent 2 (工作节点)
echo ""
echo "🚀 启动Agent 2 (工作节点) on port 8081..."
./bin/cnet-agent -config config_agent2.yaml > agent2.log 2>&1 &
AGENT2_PID=$!

# 等待Agent 2启动
echo "⏳ 等待Agent 2启动..."
for i in {1..15}; do
    if curl -s http://localhost:8081/api/health > /dev/null 2>&1; then
        echo "✅ Agent 2 启动成功"
        break
    fi
    if [ $i -eq 15 ]; then
        echo "❌ Agent 2 启动失败"
        echo "查看日志:"
        cat agent2.log
        kill $AGENT1_PID $AGENT2_PID 2>/dev/null || true
        exit 1
    fi
    echo "   尝试 $i/15..."
    sleep 1
done

echo ""
echo "🎉 两个Agent都启动成功!"
echo ""

# 等待注册完成
echo "⏳ 等待节点注册完成..."
sleep 5

# 测试发现功能
echo "🔍 测试发现功能..."
echo ""

# 检查Agent 1发现的节点
echo "📋 Agent 1 (发现服务器) 发现的节点:"
curl -s http://localhost:8080/api/discovery/nodes | jq '.' || echo "无节点发现"

echo ""

# 检查Agent 2发现的节点
echo "📋 Agent 2 (工作节点) 发现的节点:"
curl -s http://localhost:8081/api/discovery/nodes | jq '.' || echo "无节点发现"

echo ""
echo "🌐 访问地址:"
echo "   • Agent 1 (发现服务器): http://localhost:8080"
echo "   • Agent 2 (工作节点): http://localhost:8081"
echo ""

# 显示日志
echo "📝 查看Agent日志:"
echo "   • Agent 1 日志: tail -f agent1.log"
echo "   • Agent 2 日志: tail -f agent2.log"
echo ""

echo "🔄 运行中... 按 Ctrl+C 停止所有Agent"

# 设置清理函数
cleanup() {
    echo ""
    echo "🛑 停止所有Agent..."
    kill $AGENT1_PID $AGENT2_PID 2>/dev/null || true
    sleep 2
    
    echo "🧹 清理配置文件..."
    rm -f config_agent1.yaml config_agent2.yaml agent1.log agent2.log
    
    echo "✅ 清理完成"
    exit 0
}

# 捕获中断信号
trap cleanup INT

# 保持运行
while true; do
    sleep 1
done
