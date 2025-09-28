#!/bin/bash

# CNET Agent 机器学习模型作为进程任务部署演示
# 此脚本演示如何将机器学习模型作为进程类型任务来部署

set -e

AGENT_URL="http://localhost:8080"

echo "🤖 CNET Agent 机器学习模型作为进程任务部署演示"
echo "=============================================="

# 检查CNET Agent是否运行
echo "📡 检查CNET Agent状态..."
if ! curl -s "$AGENT_URL/api/health" > /dev/null; then
    echo "❌ CNET Agent未运行，请先启动CNET Agent"
    echo "   运行: ./bin/cnet-agent -config config.yaml"
    exit 1
fi
echo "✅ CNET Agent正在运行"

# 1. 部署线性回归模型作为进程任务
echo ""
echo "🔬 部署线性回归模型作为进程任务..."
curl -X POST "$AGENT_URL/api/tasks" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "linear-regression-model",
    "type": "process",
    "command": "python3",
    "args": ["examples/ml_models/simple_linear_regression.py", "train", "models/linear_regression_model.joblib", "1000"],
    "env": {
      "PYTHONPATH": "examples/ml_models",
      "MODEL_PATH": "models/linear_regression_model.joblib",
      "MODEL_TYPE": "linear_regression"
    },
    "working_dir": ".",
    "resources": {
      "cpu_limit": 1.0,
      "memory_limit": 512000000,
      "disk_limit": 1000000000
    }
  }' | jq '.'

echo "✅ 线性回归模型部署请求已发送"

# 等待模型训练完成
echo "⏳ 等待模型训练完成..."
sleep 10

# 2. 部署神经网络模型作为进程任务
echo ""
echo "🧠 部署神经网络模型作为进程任务..."
curl -X POST "$AGENT_URL/api/tasks" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "neural-network-model",
    "type": "process",
    "command": "python3",
    "args": ["examples/ml_models/neural_network.py", "train", "models/neural_network_model.h5", "1000", "50"],
    "env": {
      "PYTHONPATH": "examples/ml_models",
      "MODEL_PATH": "models/neural_network_model.h5",
      "MODEL_TYPE": "neural_network"
    },
    "working_dir": ".",
    "resources": {
      "cpu_limit": 2.0,
      "memory_limit": 1024000000,
      "disk_limit": 2000000000
    }
  }' | jq '.'

echo "✅ 神经网络模型部署请求已发送"

# 等待模型训练完成
echo "⏳ 等待神经网络模型训练完成..."
sleep 15

# 3. 列出所有任务
echo ""
echo "📋 列出所有任务..."
curl -s "$AGENT_URL/api/tasks" | jq '.'

# 4. 使用线性回归模型进行预测
echo ""
echo "🔮 使用线性回归模型进行预测..."
# 首先获取模型任务ID
MODEL_TASK_ID=$(curl -s "$AGENT_URL/api/tasks" | jq -r '.[] | select(.name == "linear-regression-model") | .id')
echo "模型任务ID: $MODEL_TASK_ID"

# 进行预测
curl -X POST "$AGENT_URL/api/tasks" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "linear-regression-prediction",
    "type": "process",
    "command": "python3",
    "args": ["examples/ml_models/simple_linear_regression.py", "predict", "models/linear_regression_model.joblib", "5.5"],
    "env": {
      "PYTHONPATH": "examples/ml_models",
      "MODEL_PATH": "models/linear_regression_model.joblib",
      "MODEL_TYPE": "linear_regression"
    },
    "working_dir": ".",
    "resources": {
      "cpu_limit": 0.5,
      "memory_limit": 256000000,
      "disk_limit": 500000000
    }
  }' | jq '.'

echo "✅ 线性回归模型预测任务已创建"

# 等待预测完成
echo "⏳ 等待预测完成..."
sleep 5

# 5. 查看预测任务日志
echo ""
echo "📄 查看预测任务日志..."
PREDICTION_TASK_ID=$(curl -s "$AGENT_URL/api/tasks" | jq -r '.[] | select(.name == "linear-regression-prediction") | .id')
curl -s "$AGENT_URL/api/tasks/$PREDICTION_TASK_ID/logs?lines=10" | jq -r '.[]'

# 6. 获取资源使用情况
echo ""
echo "📊 获取节点资源使用情况..."
curl -s "$AGENT_URL/api/resources/usage" | jq '.'

echo ""
echo "🎉 机器学习模型作为进程任务部署演示完成！"
echo ""
echo "📝 演示总结："
echo "  ✅ 成功将线性回归模型部署为进程任务"
echo "  ✅ 成功将神经网络模型部署为进程任务"
echo "  ✅ 成功进行了模型预测"
echo "  ✅ 查看了任务日志和资源使用情况"
echo ""
echo "🌐 访问Web UI: http://localhost:8080"
echo "📚 在Tasks页面查看和管理所有任务"
echo ""
echo "🔧 手动测试命令："
echo "  # 列出所有任务"
echo "  curl -s $AGENT_URL/api/tasks | jq '.'"
echo ""
echo "  # 获取特定任务信息"
echo "  curl -s $AGENT_URL/api/tasks/\$TASK_ID | jq '.'"
echo ""
echo "  # 停止任务"
echo "  curl -X DELETE $AGENT_URL/api/tasks/\$TASK_ID"
echo ""
echo "  # 查看任务日志"
echo "  curl -s $AGENT_URL/api/tasks/\$TASK_ID/logs | jq -r '.[]'"
