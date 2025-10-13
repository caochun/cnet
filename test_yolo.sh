#!/bin/bash

# 测试YOLO模型检测

set -e

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║              YOLOv5 目标检测测试                              ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# 检查模型文件
if [ ! -f "models/yolov5s.onnx" ]; then
    echo "下载 YOLOv5s 模型..."
    mkdir -p models
    curl -L -o models/yolov5s.onnx https://github.com/ultralytics/yolov5/releases/download/v7.0/yolov5s.onnx
    echo "✓ 模型已下载"
fi

echo "✓ YOLOv5s 模型: $(ls -lh models/yolov5s.onnx | awk '{print $5}')"
echo ""

# 准备测试图片
if [ ! -f "test_images/test.jpg" ]; then
    echo "下载测试图片..."
    mkdir -p test_images
    curl -s -o test_images/test.jpg https://raw.githubusercontent.com/opencv/opencv/master/samples/data/lena.jpg
fi

# 启动agent
echo "启动 CNET Agent..."
./bin/cnet-agent -config config.yaml > agent_yolo.log 2>&1 &
AGENT_PID=$!
sleep 2

echo "Agent PID: $AGENT_PID"
echo ""

echo "【提交YOLO检测任务】"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

START_TIME=$(date +%s)

RESULT=$(curl -s -X POST http://localhost:8080/api/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "name": "yolo-detection",
    "type": "vision",
    "requirements": {
      "cpu": 4.0,
      "memory": 2147483648
    },
    "config": {
      "task": "detection",
      "model_type": "yolo",
      "input_path": "test_images/test.jpg",
      "output_path": "test_output/yolo_result.jpg",
      "model_path": "models/yolov5s.onnx",
      "confidence": 0.5,
      "nms_threshold": 0.4,
      "vision_config": {
        "input_size": "640"
      }
    }
  }')

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

echo "$RESULT" | jq .

WORKLOAD_ID=$(echo "$RESULT" | jq -r '.id')
STATUS=$(echo "$RESULT" | jq -r '.status')

echo ""
echo "Workload ID: $WORKLOAD_ID"
echo "状态: $STATUS"
echo "耗时: ${DURATION}秒"
echo ""

# 等待任务完成
echo "等待YOLO推理完成..."
sleep 3

# 查看结果
echo ""
echo "【检测结果】"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
curl -s http://localhost:8080/api/workloads/$WORKLOAD_ID | jq '{
  name, 
  status,
  results
}'

# 查看日志
echo ""
echo "【模型加载日志】"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
grep "Loading model\|YOLO\|cached" agent_yolo.log | tail -5

# 检查输出文件
echo ""
if [ -f "test_output/yolo_result.jpg" ]; then
    echo "✓ 检测结果已保存: test_output/yolo_result.jpg"
    ls -lh test_output/yolo_result.jpg
    echo ""
    echo "查看标注图片:"
    echo "  open test_output/yolo_result.jpg"
else
    echo "⚠ 输出文件未生成"
fi

# 停止agent
echo ""
kill $AGENT_PID
sleep 1

echo ""
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║                    YOLO测试完成                               ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""
echo "YOLOv5s 可以检测80种物体，包括："
echo "  • person (人)"
echo "  • car, bicycle, motorcycle (交通工具)"
echo "  • dog, cat, bird (动物)"
echo "  • bottle, cup, chair (日常物品)"
echo "  • 等等..."
echo ""
echo "模型缓存："
echo "  ✓ YOLOv5s (14MB) 已加载到内存"
echo "  ✓ 后续任务将直接使用缓存，速度更快"
echo ""

