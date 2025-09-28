#!/bin/bash

# CNET Agent 机器学习模型部署演示脚本
# 此脚本演示如何在CNET Agent节点上部署和运行机器学习模型

set -e

# 配置
AGENT_URL="http://localhost:8080"
MODEL_DIR="examples/ml_models"
MODELS_DIR="models"

echo "🚀 CNET Agent 机器学习模型部署演示"
echo "=================================="

# 检查CNET Agent是否运行
echo "📡 检查CNET Agent状态..."
if ! curl -s "$AGENT_URL/api/health" > /dev/null; then
    echo "❌ CNET Agent未运行，请先启动CNET Agent"
    echo "   运行: ./bin/cnet-agent -config config.yaml"
    exit 1
fi
echo "✅ CNET Agent正在运行"

# 创建模型目录
echo "📁 创建模型目录..."
mkdir -p "$MODELS_DIR"
echo "✅ 模型目录已创建: $MODELS_DIR"

# 1. 部署线性回归模型
echo ""
echo "🔬 部署线性回归模型..."
curl -X POST "$AGENT_URL/api/ml/models" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "linear-regression-model",
    "model_type": "linear_regression",
    "model_path": "models/linear_regression_model.joblib",
    "script_path": "examples/ml_models/simple_linear_regression.py",
    "command": "python3",
    "args": ["examples/ml_models/simple_linear_regression.py", "train", "models/linear_regression_model.joblib", "1000"],
    "working_dir": ".",
    "env": {
      "PYTHONPATH": "examples/ml_models"
    },
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

# 2. 部署神经网络模型
echo ""
echo "🧠 部署神经网络模型..."
curl -X POST "$AGENT_URL/api/ml/models" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "neural-network-model",
    "model_type": "neural_network",
    "model_path": "models/neural_network_model.h5",
    "script_path": "examples/ml_models/neural_network.py",
    "command": "python3",
    "args": ["examples/ml_models/neural_network.py", "train", "models/neural_network_model.h5", "1000", "50"],
    "working_dir": ".",
    "env": {
      "PYTHONPATH": "examples/ml_models"
    },
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

# 3. 列出所有ML模型
echo ""
echo "📋 列出所有ML模型..."
curl -s "$AGENT_URL/api/ml/models" | jq '.'

# 4. 使用线性回归模型进行预测
echo ""
echo "🔮 使用线性回归模型进行预测..."
# 首先获取模型ID
MODEL_ID=$(curl -s "$AGENT_URL/api/ml/models" | jq -r '.[0].id')
echo "模型ID: $MODEL_ID"

# 进行预测
curl -X POST "$AGENT_URL/api/ml/models/$MODEL_ID/predict" \
  -H "Content-Type: application/json" \
  -d '{
    "input_data": 5.5
  }' | jq '.'

echo "✅ 线性回归模型预测完成"

# 5. 查看模型日志
echo ""
echo "📄 查看线性回归模型日志..."
curl -s "$AGENT_URL/api/ml/models/$MODEL_ID/logs?lines=20" | jq -r '.[]'

# 6. 获取资源使用情况
echo ""
echo "📊 获取节点资源使用情况..."
curl -s "$AGENT_URL/api/resources/usage" | jq '.'

# 7. 获取任务列表
echo ""
echo "📋 获取所有任务列表..."
curl -s "$AGENT_URL/api/tasks" | jq '.'

echo ""
echo "🎉 机器学习模型部署演示完成！"
echo ""
echo "📝 演示总结："
echo "  ✅ 成功部署了线性回归模型"
echo "  ✅ 成功部署了神经网络模型"
echo "  ✅ 成功进行了模型预测"
echo "  ✅ 查看了模型日志和资源使用情况"
echo ""
echo "🌐 访问Web UI: http://localhost:8080"
echo "📚 API文档: 查看 /api 端点"
echo ""
echo "🔧 手动测试命令："
echo "  # 列出所有ML模型"
echo "  curl -s $AGENT_URL/api/ml/models | jq '.'"
echo ""
echo "  # 获取特定模型信息"
echo "  curl -s $AGENT_URL/api/ml/models/\$MODEL_ID | jq '.'"
echo ""
echo "  # 停止模型"
echo "  curl -X DELETE $AGENT_URL/api/ml/models/\$MODEL_ID"
echo ""
echo "  # 查看模型日志"
echo "  curl -s $AGENT_URL/api/ml/models/\$MODEL_ID/logs | jq -r '.[]'"
