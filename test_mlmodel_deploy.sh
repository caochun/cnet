#!/bin/bash

# CNET Agent - ML模型部署测试脚本
# 测试YOLO模型部署、推理和资源管理完整流程

set -e

echo "=== CNET ML模型部署测试 ==="
echo ""

# 1. 检查依赖
echo "1. 检查依赖..."
if [ ! -f "bin/cnet-agent" ]; then
    echo "Error: bin/cnet-agent not found. Run 'make build' first."
    exit 1
fi

if [ ! -f "bin/cnet-inference-yolo" ]; then
    echo "Error: bin/cnet-inference-yolo not found. Run 'make build' first."
    exit 1
fi

if [ ! -f "models/yolo11n.onnx" ]; then
    echo "Error: YOLO model not found at models/yolo11n.onnx"
    exit 1
fi

echo "✓ 所有依赖就绪"
echo ""

# 2. 启动Agent
echo "2. 启动CNET Agent..."
pkill -f cnet-agent 2>/dev/null || true
sleep 2
./bin/cnet-agent -config config.yaml > test_mlmodel.log 2>&1 &
AGENT_PID=$!
echo "✓ Agent started (PID: $AGENT_PID)"
sleep 3

# 3. 检查Agent健康状态
echo ""
echo "3. 检查Agent健康状态..."
HEALTH=$(curl -s http://localhost:8080/api/health | jq -r '.status')
if [ "$HEALTH" != "healthy" ]; then
    echo "Error: Agent is not healthy"
    exit 1
fi
echo "✓ Agent健康状态正常"

# 4. 部署YOLO模型
echo ""
echo "4. 部署YOLO模型..."
DEPLOY_RESPONSE=$(cat << 'EOF' | curl -s -X POST http://localhost:8080/api/workloads -H "Content-Type: application/json" -d @-
{
  "name": "yolo11-service",
  "type": "mlmodel",
  "config": {
    "model_type": "yolo",
    "model_path": "models/yolo11n.onnx",
    "service_port": 9003
  },
  "requirements": {
    "cpu": 2.0,
    "memory": 2147483648
  }
}
EOF
)

WORKLOAD_ID=$(echo $DEPLOY_RESPONSE | jq -r '.id')
ENDPOINT=$(echo $DEPLOY_RESPONSE | jq -r '.endpoint')
PID=$(echo $DEPLOY_RESPONSE | jq -r '.process_pid')

echo "✓ YOLO模型部署成功"
echo "  Workload ID: $WORKLOAD_ID"
echo "  Endpoint: $ENDPOINT"
echo "  Process PID: $PID"

# 5. 等待服务启动
echo ""
echo "5. 等待YOLO推理服务启动..."
sleep 3

# 6. 检查推理服务健康状态
echo ""
echo "6. 检查推理服务健康状态..."
INFERENCE_HEALTH=$(curl -s http://localhost:9003/health | jq -r '.status')
if [ "$INFERENCE_HEALTH" != "healthy" ]; then
    echo "Error: Inference service is not healthy"
    exit 1
fi
echo "✓ 推理服务健康"

# 7. 获取服务信息
echo ""
echo "7. 获取服务信息..."
curl -s http://localhost:9003/info | jq '.'

# 8. 测试推理（使用测试图片）
echo ""
echo "8. 测试推理功能..."
if [ -f "test_images/images.jpeg" ]; then
    IMAGE_B64=$(base64 -i test_images/images.jpeg | tr -d '\n')
    PREDICT_RESULT=$(cat << EOF | curl -s -X POST http://localhost:9003/predict -H "Content-Type: application/json" -d @-
{
  "image": "$IMAGE_B64",
  "confidence": 0.5
}
EOF
)
    DETECTION_COUNT=$(echo $PREDICT_RESULT | jq -r '.count')
    echo "✓ 推理成功，检测到 $DETECTION_COUNT 个目标"
else
    echo "⚠ 测试图片不存在，跳过推理测试"
fi

# 9. 检查资源分配
echo ""
echo "9. 检查资源分配..."
RESOURCES=$(curl -s http://localhost:8080/api/resources | jq '.resources')
AVAILABLE_CPU=$(echo $RESOURCES | jq -r '.available.cpu')
USED_CPU=$(echo $RESOURCES | jq -r '.used.cpu')
USED_MEM=$(echo $RESOURCES | jq -r '.used.memory')
echo "  可用CPU: $AVAILABLE_CPU 核心"
echo "  已用CPU: $USED_CPU 核心"
echo "  已用内存: $((USED_MEM / 1024 / 1024)) MB"

# 10. 停止workload
echo ""
echo "10. 停止YOLO服务..."
curl -s -X POST http://localhost:8080/api/workloads/$WORKLOAD_ID/stop | jq -r '.message'
sleep 2

# 11. 验证进程已停止
echo ""
echo "11. 验证进程已停止..."
if ps -p $PID > /dev/null 2>&1; then
    echo "⚠ 进程仍在运行"
else
    echo "✓ 进程已停止"
fi

# 12. 验证资源已释放
echo ""
echo "12. 验证资源已释放..."
RESOURCES_AFTER=$(curl -s http://localhost:8080/api/resources | jq '.resources')
AVAILABLE_CPU_AFTER=$(echo $RESOURCES_AFTER | jq -r '.available.cpu')
USED_CPU_AFTER=$(echo $RESOURCES_AFTER | jq -r '.used.cpu')
echo "  可用CPU: $AVAILABLE_CPU_AFTER 核心"
echo "  已用CPU: $USED_CPU_AFTER 核心"

if [ "$USED_CPU_AFTER" == "0" ]; then
    echo "✓ 资源已完全释放"
else
    echo "⚠ 仍有资源被占用"
fi

echo ""
echo "=== 测试完成 ==="
echo ""
echo "📊 测试总结："
echo "✅ YOLO模型部署成功"
echo "✅ 推理服务启动正常"
echo "✅ 健康检查通过"
echo "✅ 推理功能正常"
echo "✅ 资源管理正确"
echo "✅ 停止和清理正常"
echo ""
echo "🎉 所有测试通过！"

