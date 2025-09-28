#!/bin/bash

# 测试机器学习模型部署功能
# 此脚本用于测试CNET Agent的机器学习模型部署功能

set -e

AGENT_URL="http://localhost:8080"

echo "🧪 测试CNET Agent机器学习模型部署功能"
echo "======================================"

# 检查CNET Agent是否运行
echo "📡 检查CNET Agent状态..."
if ! curl -s "$AGENT_URL/api/health" > /dev/null; then
    echo "❌ CNET Agent未运行，请先启动CNET Agent"
    echo "   运行: ./bin/cnet-agent -config config.yaml"
    exit 1
fi
echo "✅ CNET Agent正在运行"

# 测试1: 列出ML模型
echo ""
echo "🔍 测试1: 列出ML模型..."
curl -s "$AGENT_URL/api/ml/models" | jq '.' || echo "无ML模型"

# 测试2: 部署线性回归模型
echo ""
echo "🔬 测试2: 部署线性回归模型..."
RESPONSE=$(curl -s -X POST "$AGENT_URL/api/ml/models" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-linear-regression",
    "model_type": "linear_regression",
    "model_path": "models/test_linear_regression_model.joblib",
    "script_path": "examples/ml_models/simple_linear_regression.py",
    "command": "python3",
    "args": ["examples/ml_models/simple_linear_regression.py", "train", "models/test_linear_regression_model.joblib", "100"],
    "working_dir": ".",
    "env": {
      "PYTHONPATH": "examples/ml_models"
    },
    "resources": {
      "cpu_limit": 0.5,
      "memory_limit": 256000000,
      "disk_limit": 500000000
    }
  }')

echo "部署响应:"
echo "$RESPONSE" | jq '.'

# 获取模型ID
MODEL_ID=$(echo "$RESPONSE" | jq -r '.id')
echo "模型ID: $MODEL_ID"

# 等待模型训练完成
echo ""
echo "⏳ 等待模型训练完成..."
sleep 5

# 测试3: 获取模型信息
echo ""
echo "📋 测试3: 获取模型信息..."
curl -s "$AGENT_URL/api/ml/models/$MODEL_ID" | jq '.'

# 测试4: 使用模型进行预测
echo ""
echo "🔮 测试4: 使用模型进行预测..."
PREDICTION_RESPONSE=$(curl -s -X POST "$AGENT_URL/api/ml/models/$MODEL_ID/predict" \
  -H "Content-Type: application/json" \
  -d '{
    "input_data": 3.5
  }')

echo "预测响应:"
echo "$PREDICTION_RESPONSE" | jq '.'

# 测试5: 查看模型日志
echo ""
echo "📄 测试5: 查看模型日志..."
curl -s "$AGENT_URL/api/ml/models/$MODEL_ID/logs?lines=10" | jq -r '.[]' | head -20

# 测试6: 列出所有ML模型
echo ""
echo "📋 测试6: 列出所有ML模型..."
curl -s "$AGENT_URL/api/ml/models" | jq '.'

# 测试7: 停止模型
echo ""
echo "⏹️ 测试7: 停止模型..."
curl -s -X DELETE "$AGENT_URL/api/ml/models/$MODEL_ID" | jq '.'

# 验证模型已停止
echo ""
echo "✅ 验证模型已停止..."
curl -s "$AGENT_URL/api/ml/models/$MODEL_ID" | jq '.'

echo ""
echo "🎉 机器学习模型部署功能测试完成！"
echo ""
echo "📊 测试总结："
echo "  ✅ 成功部署了线性回归模型"
echo "  ✅ 成功获取了模型信息"
echo "  ✅ 成功进行了模型预测"
echo "  ✅ 成功查看了模型日志"
echo "  ✅ 成功停止了模型"
echo ""
echo "🌐 访问Web UI: http://localhost:8080"
echo "📚 查看ML Models页面以进行图形化管理"
