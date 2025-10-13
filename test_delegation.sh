#!/bin/bash

# 测试Workload委托功能

set -e

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║           Vision Workload 委托功能测试                        ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# 停止旧进程
pkill -f cnet-agent 2>/dev/null || true
sleep 1

# 确保测试图片存在
if [ ! -f "test_images/test.jpg" ]; then
    echo "下载测试图片..."
    mkdir -p test_images
    curl -s -o test_images/test.jpg https://raw.githubusercontent.com/opencv/opencv/master/samples/data/lena.jpg
    echo "✓ 测试图片已下载"
fi

# 创建日志目录
mkdir -p logs test_output

echo "【场景说明】"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "江苏省节点: CPU 0.5核, 内存 512MB (资源不足)"
echo "南京市节点: CPU 8核, 内存 16GB (资源充足)"
echo "Vision任务: 需要 CPU 2核, 内存 1GB"
echo ""
echo "预期流程:"
echo "  1. 用户向江苏省提交Vision任务"
echo "  2. 江苏省发现本地资源不足"
echo "  3. 江苏省委托给南京市"
echo "  4. 南京市执行人脸检测"
echo "  5. 返回检测结果"
echo ""

# 启动江苏省节点
echo "1. 启动江苏省节点 (port 8080, 资源不足)..."
./bin/cnet-agent -config configs/config_jiangsu.yaml > logs/jiangsu.log 2>&1 &
JIANGSU_PID=$!
echo "   PID: $JIANGSU_PID"
sleep 2

# 启动南京市节点
echo ""
echo "2. 启动南京市节点 (port 8081, 资源充足)..."
./bin/cnet-agent -config configs/config_nanjing.yaml > logs/nanjing.log 2>&1 &
NANJING_PID=$!
echo "   PID: $NANJING_PID"
sleep 3

# 验证节点连接
echo ""
echo "3. 验证节点关系..."
CHILD_COUNT=$(curl -s http://localhost:8080/api/resources/stats | jq -r '.child_nodes_count')
echo "   江苏省管理的子节点数: $CHILD_COUNT"
if [ "$CHILD_COUNT" = "1" ]; then
    echo "   ✓ 南京市已注册到江苏省"
else
    echo "   ❌ 节点注册失败"
    pkill -f cnet-agent
    exit 1
fi

# 显示资源情况
echo ""
echo "4. 查看资源情况..."
echo "   江苏省资源:"
curl -s http://localhost:8080/api/resources | jq '{cpu: .total.cpu, memory_gb: (.total.memory / 1073741824 | floor)}'
echo "   南京市资源:"
curl -s http://localhost:8081/api/resources | jq '{cpu: .total.cpu, memory_gb: (.total.memory / 1073741824 | floor)}'

# 提交Vision任务到江苏省
echo ""
echo "5. 向江苏省提交Vision任务 (需要 2核CPU, 1GB内存)..."
RESPONSE=$(curl -s -X POST http://localhost:8080/api/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "name": "face-detection-delegated",
    "type": "vision",
    "requirements": {
      "cpu": 2.0,
      "memory": 1073741824
    },
    "config": {
      "task": "face_detection",
      "input_path": "test_images/test.jpg",
      "output_path": "test_output/delegated_result.jpg"
    }
  }')

echo "$RESPONSE" | jq .

WORKLOAD_ID=$(echo "$RESPONSE" | jq -r '.id // empty')
STATUS=$(echo "$RESPONSE" | jq -r '.status // empty')

if [ -z "$WORKLOAD_ID" ]; then
    echo ""
    echo "❌ 创建Workload失败"
    echo "错误信息:"
    echo "$RESPONSE" | jq .
    echo ""
    echo "查看江苏省日志:"
    tail -20 logs/jiangsu.log
    pkill -f cnet-agent
    exit 1
fi

echo "   Workload ID: $WORKLOAD_ID"
echo "   初始状态: $STATUS"

# 等待任务完成
echo ""
echo "6. 等待任务执行..."
sleep 3

# 查看最终状态
echo ""
echo "7. 查看任务执行结果..."
FINAL_STATUS=$(curl -s http://localhost:8080/api/workloads/$WORKLOAD_ID | jq -r '.status')
echo "   最终状态: $FINAL_STATUS"

# 检查是否被委托
DELEGATED=$(curl -s http://localhost:8080/api/workloads/$WORKLOAD_ID | jq -r '.metadata.delegated // false')
if [ "$DELEGATED" = "true" ]; then
    DELEGATED_TO=$(curl -s http://localhost:8080/api/workloads/$WORKLOAD_ID | jq -r '.metadata.delegated_to')
    REMOTE_ID=$(curl -s http://localhost:8080/api/workloads/$WORKLOAD_ID | jq -r '.metadata.remote_workload_id')
    echo "   ✓ 任务已委托"
    echo "   委托给: $DELEGATED_TO"
    echo "   远程ID: $REMOTE_ID"
    
    # 查看南京市的workload
    echo ""
    echo "8. 查看南京市执行情况..."
    curl -s http://localhost:8081/api/workloads | jq '.workloads[] | {id, name, status, results}'
else
    echo "   任务在江苏省本地执行"
fi

# 查看检测结果
echo ""
echo "9. 人脸检测结果..."
curl -s http://localhost:8080/api/workloads/$WORKLOAD_ID | jq '.results'

# 检查输出文件
echo ""
if [ -f "test_output/delegated_result.jpg" ]; then
    echo "✓ 标注图片已生成: test_output/delegated_result.jpg"
    ls -lh test_output/delegated_result.jpg
else
    echo "⚠ 标注图片未生成"
fi

# 查看日志
echo ""
echo "10. 查看执行日志..."
echo "江苏省日志 (最后10行):"
tail -10 logs/jiangsu.log | grep -E "Delegat|Schedul|Vision" || echo "  (无相关日志)"
echo ""
echo "南京市日志 (最后10行):"
tail -10 logs/nanjing.log | grep -E "Workload|Vision|face" || echo "  (无相关日志)"

# 停止集群
echo ""
echo "11. 停止集群..."
kill $JIANGSU_PID $NANJING_PID 2>/dev/null || true
sleep 1

echo ""
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║                    委托功能测试完成                           ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""
if [ "$DELEGATED" = "true" ]; then
    echo "✅ 委托功能正常："
    echo "   江苏省 (资源不足) → 南京市 (资源充足)"
    echo "   Vision任务成功在南京市执行"
else
    echo "⚠ 委托未发生，可能的原因："
    echo "   1. 江苏省资源仍然充足"
    echo "   2. 委托逻辑需要调试"
    echo "   3. 查看日志文件了解详情"
fi
echo ""

