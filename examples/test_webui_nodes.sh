#!/bin/bash

# 测试Web UI中的节点显示功能

echo "🌐 测试Web UI中的节点显示功能"
echo "================================"
echo ""

# 检查两个agent是否运行
echo "🔍 检查Agent状态..."

AGENT1_RUNNING=false
AGENT2_RUNNING=false

if curl -s http://localhost:8080/api/health > /dev/null 2>&1; then
    echo "✅ Agent 1 (发现服务器) 运行中"
    AGENT1_RUNNING=true
else
    echo "❌ Agent 1 未运行"
fi

if curl -s http://localhost:8081/api/health > /dev/null 2>&1; then
    echo "✅ Agent 2 (工作节点) 运行中"
    AGENT2_RUNNING=true
else
    echo "❌ Agent 2 未运行"
fi

echo ""

if [ "$AGENT1_RUNNING" = true ] && [ "$AGENT2_RUNNING" = true ]; then
    echo "🎉 两个Agent都在运行，可以测试Web UI功能"
    echo ""
    
    # 测试Agent 1的节点发现
    echo "📋 Agent 1 发现的节点:"
    curl -s http://localhost:8080/api/discovery/nodes | jq '.'
    
    echo ""
    echo "📊 集群统计信息:"
    NODES=$(curl -s http://localhost:8080/api/discovery/nodes)
    TOTAL_NODES=$(echo "$NODES" | jq '. | length')
    ACTIVE_NODES=$(echo "$NODES" | jq '[.[] | select(.status == "active")] | length')
    REGIONS=$(echo "$NODES" | jq '[.[].region] | unique | length')
    
    echo "   总节点数: $TOTAL_NODES"
    echo "   活跃节点: $ACTIVE_NODES"
    echo "   区域数量: $REGIONS"
    
    echo ""
    echo "🌐 Web UI访问地址:"
    echo "   • Agent 1 (发现服务器): http://localhost:8080"
    echo "   • Agent 2 (工作节点): http://localhost:8081"
    echo ""
    echo "💡 在Web UI中你可以看到:"
    echo "   • 仪表板页面显示注册的节点信息"
    echo "   • 集群概览统计"
    echo "   • 节点状态和最后活跃时间"
    echo "   • 节点详细信息（地址、端口、区域等）"
    
else
    echo "❌ 需要先启动两个Agent"
    echo "   运行: ./examples/start_two_agents.sh"
fi

echo ""
echo "🔧 手动注册测试节点..."
REGISTER_RESPONSE=$(curl -s -X POST http://localhost:8080/api/discovery/register \
  -H "Content-Type: application/json" \
  -d '{
    "node": {
      "id": "test-node-001",
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

echo ""
echo "📋 注册后的节点列表:"
curl -s http://localhost:8080/api/discovery/nodes | jq '.'

echo ""
echo "🎉 测试完成！现在可以在Web UI中查看节点信息了"
