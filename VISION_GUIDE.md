# Vision Workload 使用指南

CNET Agent 支持基于 GoCV 的计算机视觉任务。

## 前置要求

```bash
# macOS
brew install opencv pkg-config

# 验证安装
pkg-config --modversion opencv4
```

## 支持的任务类型

### 1. 人脸检测 (face_detection)

最简单的用法，使用默认的 Haar Cascade 模型：

```bash
curl -X POST http://localhost:8080/api/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "name": "face-detection-task",
    "type": "vision",
    "requirements": {
      "cpu": 2.0,
      "memory": 1073741824
    },
    "config": {
      "task": "face_detection",
      "input_path": "/path/to/image.jpg",
      "output_path": "/path/to/output.jpg"
    }
  }'
```

**指定自定义模型：**
```json
{
  "config": {
    "task": "face_detection",
    "input_path": "photo.jpg",
    "output_path": "faces_detected.jpg",
    "model_path": "/path/to/haarcascade_frontalface_default.xml"
  }
}
```

### 2. 目标检测 (detection)

#### 使用 DNN 模型 (MobileNet-SSD 等)

```bash
curl -X POST http://localhost:8080/api/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "name": "object-detection",
    "type": "vision",
    "requirements": {
      "cpu": 4.0,
      "memory": 2147483648
    },
    "config": {
      "task": "detection",
      "model_type": "dnn",
      "input_path": "image.jpg",
      "output_path": "detected.jpg",
      "model_path": "mobilenet_ssd.caffemodel",
      "confidence": 0.5,
      "vision_config": {
        "config_path": "mobilenet_ssd.prototxt",
        "input_size": "300x300"
      }
    }
  }'
```

#### 使用 YOLO 模型

**ONNX 格式：**
```json
{
  "config": {
    "task": "detection",
    "model_type": "yolo",
    "input_path": "image.jpg",
    "output_path": "yolo_detected.jpg",
    "model_path": "yolov5s.onnx",
    "confidence": 0.5,
    "nms_threshold": 0.4,
    "vision_config": {
      "input_size": "640"
    }
  }
}
```

**Darknet 格式：**
```json
{
  "config": {
    "task": "detection",
    "model_type": "yolo",
    "model_path": "yolov4.weights",
    "confidence": 0.5,
    "nms_threshold": 0.4,
    "vision_config": {
      "config_path": "yolov4.cfg",
      "input_size": "416"
    }
  }
}
```

### 3. 图像分类 (classification)

```bash
curl -X POST http://localhost:8080/api/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "name": "image-classification",
    "type": "vision",
    "requirements": {
      "cpu": 2.0,
      "memory": 1073741824
    },
    "config": {
      "task": "classification",
      "model_type": "dnn",
      "input_path": "image.jpg",
      "model_path": "resnet50.onnx",
      "confidence": 0.5
    }
  }'
```

### 4. 视频跟踪 (tracking)

```bash
curl -X POST http://localhost:8080/api/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "name": "video-tracking",
    "type": "vision",
    "requirements": {
      "cpu": 4.0,
      "memory": 2147483648
    },
    "config": {
      "task": "tracking",
      "input_path": "video.mp4"
    }
  }'
```

## 查看结果

```bash
# 查看 workload 状态
curl http://localhost:8080/api/workloads/{workload-id}

# 查看详细日志
curl http://localhost:8080/api/workloads/{workload-id}/logs

# 检测结果会在 Results 字段中
```

## 配置参数说明

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| task | string | 是 | 任务类型：detection, face_detection, classification, tracking |
| input_path | string | 是 | 输入图片/视频路径 |
| output_path | string | 否 | 输出路径（标注后的图像） |
| model_path | string | 视情况 | 模型文件路径 |
| model_type | string | 视情况 | 模型类型：dnn, yolo, cascade |
| confidence | float | 否 | 置信度阈值 (0.0-1.0)，默认 0.5 |
| nms_threshold | float | 否 | NMS阈值 (0.0-1.0)，默认 0.4 |

## 模型文件

### Haar Cascade (人脸检测)

GoCV会自动查找以下路径：
- `/usr/local/share/opencv4/haarcascades/`
- `/opt/homebrew/share/opencv4/haarcascades/`

常用模型：
- `haarcascade_frontalface_default.xml` - 正面人脸
- `haarcascade_eye.xml` - 眼睛检测
- `haarcascade_smile.xml` - 笑容检测

### DNN 模型

支持格式：
- Caffe (.caffemodel + .prototxt)
- TensorFlow (.pb)
- ONNX (.onnx)
- Darknet (.weights + .cfg)

### YOLO 模型

推荐使用 ONNX 格式的 YOLOv5/YOLOv8：
```bash
# 下载预训练模型
wget https://github.com/ultralytics/yolov5/releases/download/v7.0/yolov5s.onnx
```

## 示例场景

### 场景1：批量人脸检测

```bash
#!/bin/bash
for img in images/*.jpg; do
  basename=$(basename "$img" .jpg)
  curl -X POST http://localhost:8080/api/workloads \
    -H "Content-Type: application/json" \
    -d "{
      \"name\": \"face-$basename\",
      \"type\": \"vision\",
      \"requirements\": {\"cpu\": 1.0, \"memory\": 536870912},
      \"config\": {
        \"task\": \"face_detection\",
        \"input_path\": \"$img\",
        \"output_path\": \"output/${basename}_faces.jpg\"
      }
    }"
done
```

### 场景2：视频监控分析

```bash
curl -X POST http://localhost:8080/api/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "name": "surveillance-analysis",
    "type": "vision",
    "requirements": {
      "cpu": 4.0,
      "memory": 2147483648
    },
    "config": {
      "task": "detection",
      "model_type": "yolo",
      "input_path": "surveillance.mp4",
      "model_path": "yolov5s.onnx",
      "confidence": 0.6
    }
  }'
```

## 性能建议

- **CPU**: 建议至少 2 核用于图像处理，4 核用于视频处理
- **内存**: 
  - 图像处理: 512MB - 1GB
  - 视频处理: 1GB - 2GB
  - YOLO模型: 2GB+
- **GPU**: 如果有 CUDA 支持，性能可提升 10-50 倍

## 故障排查

### 编译错误

```bash
# 检查 OpenCV 是否安装
pkg-config --modversion opencv4

# 检查 pkg-config
which pkg-config

# 重新安装
brew reinstall opencv pkg-config
```

### 模型加载失败

- 检查模型文件路径是否正确
- 检查模型格式是否匹配
- 查看 workload 日志获取详细错误

### 内存不足

- 增加 workload 的 memory 资源需求
- 使用更小的模型
- 降低输入图像分辨率

