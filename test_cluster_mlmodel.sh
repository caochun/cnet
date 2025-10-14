#!/bin/bash

# CNET Agent - 集群ML模型部署测试
# 测试触发式心跳和资源实时同步

set -e

echo "=== CNET 集群ML模型部署测试 ==="
echo ""

# 1. 启动集群
echo "1. 启动江苏集群..."
./start_cluster.sh > /dev/null 2>&1
sleep 5
echo "✓ 集群启动成功"
echo "  - 江苏省 (jiangsu): http://localhost:8080"
echo "  - 南京市 (nanjing): http://localhost:8081"
echo "  - 宿迁市 (suqian): http://localhost:8082"
echo "  - 常州市 (changzhou): http://localhost:8083"

# 2. 检查初始资源
echo ""
echo "2. 检查南京节点初始资源..."
NANJING_INITIAL=$(curl -s http://localhost:8081/api/resources | jq '.resources')
CPU_TOTAL=$(echo $NANJING_INITIAL | jq -r '.total.cpu')
MEM_TOTAL=$(echo $NANJING_INITIAL | jq -r '.total.memory')
echo "  总资源: CPU ${CPU_TOTAL}核心, 内存 $((MEM_TOTAL / 1024 / 1024 / 1024))GB"

# 3. 江苏节点视角（部署前）
echo ""
echo "3. 江苏节点看到的南京资源（部署前）..."
JIANGSU_VIEW_BEFORE=$(curl -s http://localhost:8080/api/nodes | jq '.peers[] | select(.node_id=="nanjing")')
AVAILABLE_CPU_BEFORE=$(echo $JIANGSU_VIEW_BEFORE | jq -r '.available.cpu')
AVAILABLE_MEM_BEFORE=$(echo $JIANGSU_VIEW_BEFORE | jq -r '.available.memory')
echo "  可用资源: CPU ${AVAILABLE_CPU_BEFORE}核心, 内存 $((AVAILABLE_MEM_BEFORE / 1024 / 1024 / 1024))GB"

# 4. 在南京部署YOLO模型
echo ""
echo "4. 在南京节点部署YOLO模型..."
DEPLOY_RESULT=$(cat << 'EOF' | curl -s -X POST http://localhost:8081/api/workloads -H "Content-Type: application/json" -d @-
{
  "name": "nanjing-yolo-service",
  "type": "mlmodel",
  "config": {
    "model_type": "yolo",
    "model_path": "models/yolo11n.onnx",
    "service_port": 9201
  },
  "requirements": {
    "cpu": 2.0,
    "memory": 2147483648
  }
}
EOF
)

WORKLOAD_ID=$(echo $DEPLOY_RESULT | jq -r '.id')
ENDPOINT=$(echo $DEPLOY_RESULT | jq -r '.endpoint')
PID=$(echo $DEPLOY_RESULT | jq -r '.process_pid')

echo "✓ YOLO模型部署成功"
echo "  Workload ID: $WORKLOAD_ID"
echo "  Endpoint: $ENDPOINT"
echo "  Process PID: $PID"

# 5. 立即检查江苏节点视角（测试触发式心跳）
echo ""
echo "5. 🔥 触发式心跳测试：部署后立即检查江苏节点视角（2秒后）..."
sleep 2
JIANGSU_VIEW_AFTER=$(curl -s http://localhost:8080/api/nodes | jq '.peers[] | select(.node_id=="nanjing")')
AVAILABLE_CPU_AFTER=$(echo $JIANGSU_VIEW_AFTER | jq -r '.available.cpu')
AVAILABLE_MEM_AFTER=$(echo $JIANGSU_VIEW_AFTER | jq -r '.available.memory')
USED_CPU=$(echo $JIANGSU_VIEW_AFTER | jq -r '.used.cpu')
USED_MEM=$(echo $JIANGSU_VIEW_AFTER | jq -r '.used.memory')

echo "  可用资源: CPU ${AVAILABLE_CPU_AFTER}核心, 内存 $((AVAILABLE_MEM_AFTER / 1024 / 1024 / 1024))GB"
echo "  已用资源: CPU ${USED_CPU}核心, 内存 $((USED_MEM / 1024 / 1024 / 1024))GB"

# 验证资源变化
if [ "$AVAILABLE_CPU_AFTER" == "6" ] && [ "$USED_CPU" == "2" ]; then
    echo "  ✅ 资源变化立即同步成功！（2秒内）"
else
    echo "  ❌ 资源同步失败"
    exit 1
fi

# 6. 验证推理服务
echo ""
echo "6. 验证YOLO推理服务..."
sleep 3
HEALTH=$(curl -s http://localhost:9201/health | jq -r '.status')
if [ "$HEALTH" == "healthy" ]; then
    echo "✓ 推理服务健康检查通过"
else
    echo "❌ 推理服务健康检查失败"
    exit 1
fi

# 7. 在Web UI中查看
echo ""
echo "7. Web UI展示验证..."
WORKLOAD_IN_UI=$(curl -s http://localhost:8081/api/workloads | jq '.workloads[] | select(.name=="nanjing-yolo-service")')
if [ -n "$WORKLOAD_IN_UI" ]; then
    echo "✓ Workload在Web UI中可见"
    echo $WORKLOAD_IN_UI | jq '{name, type, status, endpoint}'
else
    echo "❌ Workload未在Web UI中显示"
fi

# 8. 停止服务
echo ""
echo "8. 停止YOLO服务..."
curl -s -X POST http://localhost:8081/api/workloads/$WORKLOAD_ID/stop | jq -r '.message'
sleep 2

# 9. 验证资源释放的触发式心跳
echo ""
echo "9. 🔥 触发式心跳测试：停止后立即检查资源释放（2秒后）..."
sleep 2
JIANGSU_VIEW_RELEASED=$(curl -s http://localhost:8080/api/nodes | jq '.peers[] | select(.node_id=="nanjing")')
AVAILABLE_CPU_RELEASED=$(echo $JIANGSU_VIEW_RELEASED | jq -r '.available.cpu')
USED_CPU_RELEASED=$(echo $JIANGSU_VIEW_RELEASED | jq -r '.used.cpu')

echo "  可用资源: CPU ${AVAILABLE_CPU_RELEASED}核心"
echo "  已用资源: CPU ${USED_CPU_RELEASED}核心"

if [ "$AVAILABLE_CPU_RELEASED" == "8" ] && [ "$USED_CPU_RELEASED" == "0" ]; then
    echo "  ✅ 资源释放立即同步成功！（2秒内）"
else
    echo "  ❌ 资源释放同步失败"
    exit 1
fi

echo ""
echo "=== 测试完成 ==="
echo ""
echo "📊 触发式心跳测试总结："
echo "✅ 资源分配时立即触发心跳（2秒同步）"
echo "✅ 资源释放时立即触发心跳（2秒同步）"
echo "✅ 父节点实时了解子节点资源状态"
echo "✅ YOLO模型部署和运行正常"
echo "✅ Web UI正确显示workload"
echo "✅ 推理服务健康检查通过"
echo ""
echo "🎉 集群ML模型部署测试全部通过！"
echo ""
echo "💡 提示："
echo "  - 南京节点Web UI: http://localhost:8081/"
echo "  - 江苏节点Web UI: http://localhost:8080/"
echo "  - 你可以在Web UI中实时看到资源变化和workload状态"

