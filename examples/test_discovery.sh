#!/bin/bash

# CNET Agent Discovery 测试脚本
# 测试agent之间的注册和发现功能

set -e

echo "🌐 CNET Agent Discovery 测试"
echo "============================"
echo ""

# 停止任何现有的agent
echo "🛑 停止现有agent..."
pkill -f cnet-agent 2>/dev/null || true
sleep 2

# 创建两个不同的配置文件
echo "📝 创建测试配置文件..."

# Agent 1 配置 (作为发现服务器)
cat > config_agent1.yaml << EOF
agent:
  address: "0.0.0.0"
  port: 8080
  node_id: "agent-1"
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

# Agent 2 配置 (向Agent 1注册)
cat > config_agent2.yaml << EOF
agent:
  address: "0.0.0.0"
  port: 8081
  node_id: "agent-2"
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

# 构建agent
echo "📦 构建CNET Agent..."
make build

echo ""
echo "🚀 启动Agent 1 (发现服务器)..."
./bin/cnet-agent -config config_agent1.yaml &
AGENT1_PID=$!

# 等待Agent 1启动
echo "⏳ 等待Agent 1启动..."
for i in {1..10}; do
    if curl -s http://localhost:8080/api/health > /dev/null 2>&1; then
        echo "✅ Agent 1 启动成功"
        break
    fi
    if [ $i -eq 10 ]; then
        echo "❌ Agent 1 启动失败"
        kill $AGENT1_PID 2>/dev/null || true
        exit 1
    fi
    sleep 1
done

echo ""
echo "🚀 启动Agent 2 (工作节点)..."
./bin/cnet-agent -config config_agent2.yaml &
AGENT2_PID=$!

# 等待Agent 2启动
echo "⏳ 等待Agent 2启动..."
for i in {1..10}; do
    if curl -s http://localhost:8081/api/health > /dev/null 2>&1; then
        echo "✅ Agent 2 启动成功"
        break
    fi
    if [ $i -eq 10 ]; then
        echo "❌ Agent 2 启动失败"
        kill $AGENT1_PID $AGENT2_PID 2>/dev/null || true
        exit 1
    fi
    sleep 1
done

echo ""
echo "🔍 测试发现功能..."

# 等待注册完成
echo "⏳ 等待节点注册..."
sleep 5

# 测试Agent 1的节点列表
echo ""
echo "📋 Agent 1 发现的节点:"
curl -s http://localhost:8080/api/discovery/nodes | jq '.' || echo "无节点发现"

# 测试Agent 2的节点列表
echo ""
echo "📋 Agent 2 发现的节点:"
curl -s http://localhost:8081/api/discovery/nodes | jq '.' || echo "无节点发现"

# 测试手动注册
echo ""
echo "🔧 测试手动注册..."
REGISTER_RESPONSE=$(curl -s -X POST http://localhost:8080/api/discovery/register \
  -H "Content-Type: application/json" \
  -d '{
    "node": {
      "id": "test-node",
      "name": "Test Node",
      "address": "192.168.1.100",
      "port": 8082,
      "region": "us-east",
      "datacenter": "dc2",
      "status": "active",
      "last_seen": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
      "metadata": {
        "version": "1.0.0",
        "test": "true"
      }
    }
  }')

echo "注册响应: $REGISTER_RESPONSE"

# 再次查看节点列表
echo ""
echo "📋 注册后的节点列表:"
curl -s http://localhost:8080/api/discovery/nodes | jq '.'

echo ""
echo "🎉 发现功能测试完成!"
echo ""
echo "🌐 访问地址:"
echo "   • Agent 1 (发现服务器): http://localhost:8080"
echo "   • Agent 2 (工作节点): http://localhost:8081"
echo ""
echo "🛑 停止测试..."
kill $AGENT1_PID $AGENT2_PID 2>/dev/null || true
sleep 2

# 清理配置文件
rm -f config_agent1.yaml config_agent2.yaml

echo "✅ 测试完成，配置文件已清理"
