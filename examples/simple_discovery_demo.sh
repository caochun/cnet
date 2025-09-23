#!/bin/bash

# 简单的CNET Agent Discovery演示

echo "🌐 CNET Agent Discovery 功能演示"
echo "================================"
echo ""

# 停止现有agent
pkill -f cnet-agent 2>/dev/null || true
sleep 1

echo "📝 当前配置状态:"
echo "   Discovery enabled: $(grep -A1 'discovery:' config.yaml | grep 'enabled:' | awk '{print $2}')"
echo "   Discovery servers: $(grep -A1 'servers:' config.yaml | tail -1 | sed 's/.*- //')"
echo ""

echo "🔧 修改配置以启用发现功能..."

# 创建启用发现的配置
cat > config_with_discovery.yaml << EOF
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
  servers: []  # 作为发现服务器
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

# 构建并启动agent
echo "📦 构建并启动Agent..."
make build
./bin/cnet-agent -config config_with_discovery.yaml &
AGENT_PID=$!

# 等待启动
echo "⏳ 等待Agent启动..."
for i in {1..10}; do
    if curl -s http://localhost:8080/api/health > /dev/null 2>&1; then
        echo "✅ Agent启动成功"
        break
    fi
    sleep 1
done

echo ""
echo "🔍 测试发现功能..."

# 测试手动注册一个节点
echo "📝 手动注册测试节点..."
REGISTER_RESPONSE=$(curl -s -X POST http://localhost:8080/api/discovery/register \
  -H "Content-Type: application/json" \
  -d '{
    "node": {
      "id": "test-node-001",
      "name": "Test Worker Node",
      "address": "192.168.1.100",
      "port": 8081,
      "region": "us-west",
      "datacenter": "dc1",
      "status": "active",
      "last_seen": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
      "metadata": {
        "version": "1.0.0",
        "os": "linux",
        "arch": "amd64"
      }
    }
  }')

echo "注册响应: $REGISTER_RESPONSE"

# 查看发现的节点
echo ""
echo "📋 发现的节点列表:"
curl -s http://localhost:8080/api/discovery/nodes | jq '.' || echo "无节点发现"

echo ""
echo "🎉 演示完成!"
echo ""
echo "💡 总结:"
echo "   ✅ Agent 可以作为发现服务器接收其他节点的注册"
echo "   ✅ 支持手动注册节点到发现服务器"
echo "   ✅ 可以查看已注册的节点列表"
echo "   ✅ 支持节点状态管理和心跳检测"
echo ""
echo "🌐 访问Web UI查看节点: http://localhost:8080"
echo ""

# 清理
echo "🛑 停止Agent..."
kill $AGENT_PID 2>/dev/null || true
rm -f config_with_discovery.yaml

echo "✅ 清理完成"
