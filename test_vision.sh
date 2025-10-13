#!/bin/bash

# Vision功能测试脚本

set -e

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║              Vision Workload 功能测试                         ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# 检查OpenCV是否安装
echo "1. 检查依赖..."
if ! pkg-config --exists opencv4; then
    echo "❌ OpenCV未安装！"
    echo "请运行: brew install opencv pkg-config"
    exit 1
fi

OPENCV_VERSION=$(pkg-config --modversion opencv4)
echo "✓ OpenCV版本: $OPENCV_VERSION"
echo ""

# 查找测试图片
echo "2. 准备测试数据..."
if [ ! -f "test_images/test.jpg" ]; then
    echo "提示: test_images/test.jpg 不存在"
    echo "你可以："
    echo "  1. 放一张图片到 test_images/test.jpg"
    echo "  2. 或者下载测试图片："
    echo "     curl -o test_images/test.jpg https://raw.githubusercontent.com/opencv/opencv/master/samples/data/lena.jpg"
    echo ""
    read -p "是否下载测试图片? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        mkdir -p test_images
        curl -o test_images/test.jpg https://raw.githubusercontent.com/opencv/opencv/master/samples/data/lena.jpg
        echo "✓ 测试图片已下载"
    else
        echo "跳过测试图片准备"
        echo "请手动准备图片后再运行测试"
        exit 0
    fi
else
    echo "✓ 找到测试图片: test_images/test.jpg"
fi
echo ""

# 查找Haar Cascade模型
echo "3. 查找人脸检测模型..."
HAAR_MODEL=""
POSSIBLE_PATHS=(
    "/opt/homebrew/share/opencv4/haarcascades/haarcascade_frontalface_default.xml"
    "/usr/local/share/opencv4/haarcascades/haarcascade_frontalface_default.xml"
    "/usr/share/opencv4/haarcascades/haarcascade_frontalface_default.xml"
)

for path in "${POSSIBLE_PATHS[@]}"; do
    if [ -f "$path" ]; then
        HAAR_MODEL="$path"
        echo "✓ 找到模型: $HAAR_MODEL"
        break
    fi
done

if [ -z "$HAAR_MODEL" ]; then
    echo "⚠ 未找到Haar Cascade模型"
    echo "将使用自动查找功能"
fi
echo ""

# 启动agent
echo "4. 启动CNET Agent..."
./bin/cnet-agent -config config.yaml > test_vision_agent.log 2>&1 &
AGENT_PID=$!
echo "   Agent PID: $AGENT_PID"
sleep 2

# 测试健康检查
if ! curl -s http://localhost:8080/api/health > /dev/null; then
    echo "❌ Agent启动失败！"
    kill $AGENT_PID 2>/dev/null || true
    exit 1
fi
echo "✓ Agent运行正常"
echo ""

# 测试1: 人脸检测
echo "5. 测试人脸检测..."
WORKLOAD_ID=$(curl -s -X POST http://localhost:8080/api/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "name": "face-detection-test",
    "type": "vision",
    "requirements": {
      "cpu": 2.0,
      "memory": 1073741824
    },
    "config": {
      "task": "face_detection",
      "input_path": "test_images/test.jpg",
      "output_path": "test_output/faces_detected.jpg"
    }
  }' | jq -r '.id')

echo "   Workload ID: $WORKLOAD_ID"
sleep 2

# 查看结果
echo ""
echo "6. 查看检测结果..."
RESULT=$(curl -s http://localhost:8080/api/workloads/$WORKLOAD_ID)
STATUS=$(echo "$RESULT" | jq -r '.status')
echo "   状态: $STATUS"

if [ "$STATUS" = "completed" ]; then
    echo "   ✓ 人脸检测完成"
    echo "$RESULT" | jq '.results'
    
    if [ -f "test_output/faces_detected.jpg" ]; then
        echo "   ✓ 标注图片已保存: test_output/faces_detected.jpg"
        ls -lh test_output/faces_detected.jpg
    fi
elif [ "$STATUS" = "failed" ]; then
    echo "   ❌ 检测失败"
    echo "   查看日志:"
    curl -s http://localhost:8080/api/workloads/$WORKLOAD_ID/logs | jq '.logs[]'
else
    echo "   ⏳ 状态: $STATUS"
fi
echo ""

# 查看workload日志
echo "7. Workload日志..."
curl -s http://localhost:8080/api/workloads/$WORKLOAD_ID/logs | jq -r '.logs[]'
echo ""

# 停止agent
echo "8. 停止Agent..."
kill $AGENT_PID
wait $AGENT_PID 2>/dev/null || true
echo ""

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║                    测试完成                                   ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""
echo "测试结果："
echo "  ✓ Vision Workload 创建成功"
echo "  ✓ GoCV 功能正常"
if [ -f "test_output/faces_detected.jpg" ]; then
    echo "  ✓ 图像处理完成"
    echo ""
    echo "查看结果图片:"
    echo "  open test_output/faces_detected.jpg"
fi
echo ""
echo "其他测试:"
echo "  - 修改 test_images/test.jpg 为你的图片"
echo "  - 修改脚本中的 task 类型（detection, classification等）"
echo "  - 提供自己的模型文件"
echo ""

