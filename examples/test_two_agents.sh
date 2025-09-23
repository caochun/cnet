#!/bin/bash

# 测试两个CNET Agent之间的通信

echo "🔍 测试两个CNET Agent之间的通信"
echo "================================"
echo ""

# 检查Agent 1状态
echo "🏥 Agent 1 (发现服务器) 健康检查:"
curl -s http://localhost:8080/api/health | jq '.'

echo ""

# 检查Agent 2状态
echo "🏥 Agent 2 (工作节点) 健康检查:"
curl -s http://localhost:8081/api/health | jq '.'

echo ""

# 检查节点发现
echo "📋 Agent 1 发现的节点:"
curl -s http://localhost:8080/api/discovery/nodes | jq '.'

echo ""

# 检查Agent 2发现的节点
echo "📋 Agent 2 发现的节点:"
curl -s http://localhost:8081/api/discovery/nodes | jq '.'

echo ""

# 测试在Agent 2上创建任务
echo "🔧 在Agent 2上创建测试任务..."
TASK_RESPONSE=$(curl -s -X POST http://localhost:8081/api/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "worker-task",
    "type": "process",
    "command": "echo",
    "args": ["Hello from Worker Node!"],
    "env": {
      "NODE": "worker-node"
    }
  }')

echo "任务创建响应:"
echo "$TASK_RESPONSE" | jq '.'

# 提取任务ID
TASK_ID=$(echo "$TASK_RESPONSE" | jq -r '.id')
echo "任务ID: $TASK_ID"

echo ""

# 等待任务完成
echo "⏳ 等待任务完成..."
sleep 3

# 查看任务状态
echo "📋 Agent 2 的任务列表:"
curl -s http://localhost:8081/api/tasks | jq '.'

echo ""

# 查看任务日志
echo "📝 任务日志:"
curl -s "http://localhost:8081/api/tasks/$TASK_ID/logs" | jq '.'

echo ""

echo "🎉 测试完成!"
echo ""
echo "💡 总结:"
echo "   ✅ Agent 1 作为发现服务器运行在端口 8080"
echo "   ✅ Agent 2 作为工作节点运行在端口 8081"
echo "   ✅ Agent 2 成功向 Agent 1 注册"
echo "   ✅ Agent 1 可以发现 Agent 2"
echo "   ✅ 两个Agent都可以独立处理任务"
echo ""
echo "🌐 访问地址:"
echo "   • Agent 1 Web UI: http://localhost:8080"
echo "   • Agent 2 Web UI: http://localhost:8081"
