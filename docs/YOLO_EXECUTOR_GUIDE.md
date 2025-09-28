# YOLO推理引擎执行器指南

本文档介绍如何在CNET系统中使用YOLO推理引擎执行器进行目标检测任务。

## 概述

YOLO（You Only Look Once）推理引擎执行器是CNET系统的一个专门组件，用于执行基于YOLO模型的目标检测推理任务。它基于[Ultralytics YOLO](https://docs.ultralytics.com/zh/modes/predict/)官方库实现，支持多种YOLO版本（YOLOv5、YOLOv8、YOLOv9、YOLOv10、YOLOv11等），并提供了完整的API接口用于模型管理和推理。

## 功能特性

- **多版本YOLO支持**：支持YOLOv5、YOLOv8、YOLOv9、YOLOv10、YOLOv11等版本
- **灵活的模型配置**：支持自定义置信度阈值、IoU阈值、图像尺寸等参数
- **多种输入格式**：支持图像文件路径、Base64编码图像、图像URL等输入方式
- **实时推理服务**：提供HTTP API接口进行实时目标检测
- **资源管理**：支持CPU、内存、GPU资源限制
- **日志监控**：完整的任务日志和性能监控

## 架构组件

### 1. YOLO执行器 (YOLOExecutor)
- 位置：`internal/agent/tasks/yolo_executor.go`
- 功能：执行YOLO推理任务，管理推理服务器生命周期

### 2. YOLO推理引擎 (YOLOInferenceEngine)
- 位置：`internal/agent/ml/service.go`
- 功能：提供YOLO模型加载、卸载、推理等核心功能

### 3. YOLO任务处理器 (YOLOTaskHandler)
- 位置：`internal/agent/api/yolo_task_handler.go`
- 功能：处理YOLO相关的HTTP API请求

### 4. API路由
- 基础路径：`/api/yolo`
- 支持的操作：创建任务、获取任务信息、执行推理、获取日志等

## 安装和设置

### 安装依赖

首先需要安装Ultralytics YOLO和相关依赖：

```bash
# 安装YOLO依赖
./examples/install_yolo_deps.sh

# 或者手动安装
pip install ultralytics>=8.0.0
pip install torch torchvision torchaudio
pip install opencv-python>=4.5.0
pip install Pillow>=8.0.0
pip install numpy>=1.19.0
pip install requests>=2.25.0
```

### 验证安装

```bash
python3 -c "
from ultralytics import YOLO
model = YOLO('yolo11n.pt')
print('YOLO installation successful!')
"
```

## 配置说明

### 基本配置

```yaml
# YOLO执行器配置
tasks:
  yolo:
    enabled: true
    max_concurrent_tasks: 5
    default_timeout: "300s"
    
    # 模型配置
    model:
      default_type: "yolov8"
      default_confidence: 0.5
      default_iou_threshold: 0.45
      default_image_size: 640
      max_detections: 100
      
    # 资源限制
    resources:
      cpu_limit: 2.0
      memory_limit: 4096  # MB
      disk_limit: 1024    # MB
      gpu_limit: 1
```

### 环境变量

```bash
# CUDA设置
CUDA_VISIBLE_DEVICES=0

# Python路径
PYTHONPATH=/opt/yolo

# 线程数设置
OMP_NUM_THREADS=4

# YOLO服务器端口
YOLO_SERVER_PORT=9000
```

## API使用指南

### 1. 创建YOLO任务

```bash
curl -X POST http://localhost:8080/api/yolo/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "yolo-detection-task",
    "model_path": "/path/to/yolo/model.pt",
    "script_path": "/path/to/yolo/inference.py",
    "config": {
      "model_type": "yolov8",
      "confidence": 0.5,
      "iou_threshold": 0.45,
      "image_size": 640,
      "max_detections": 100,
      "classes": ["person", "car", "bicycle", "dog", "cat"]
    },
    "resources": {
      "cpu_limit": 2.0,
      "memory_limit": 4096,
      "disk_limit": 1024
    }
  }'
```

### 2. 列出YOLO任务

```bash
curl -X GET http://localhost:8080/api/yolo/tasks
```

### 3. 获取任务信息

```bash
curl -X GET http://localhost:8080/api/yolo/tasks/{task_id}
```

### 4. 执行YOLO推理

```bash
curl -X POST http://localhost:8080/api/yolo/tasks/{task_id}/predict \
  -H "Content-Type: application/json" \
  -d '{
    "image_path": "/path/to/image.jpg",
    "config": {
      "confidence": 0.6,
      "iou_threshold": 0.4,
      "image_size": 640
    },
    "options": {
      "save_results": true,
      "output_format": "json"
    }
  }'
```

### 5. 获取模型信息

```bash
curl -X GET http://localhost:8080/api/yolo/tasks/{task_id}/model
```

### 6. 检查任务健康状态

```bash
curl -X GET http://localhost:8080/api/yolo/tasks/{task_id}/health
```

### 7. 获取任务日志

```bash
curl -X GET http://localhost:8080/api/yolo/tasks/{task_id}/logs
```

### 8. 停止任务

```bash
curl -X DELETE http://localhost:8080/api/yolo/tasks/{task_id}
```

## 推理请求格式

### 输入格式

支持三种输入方式：

1. **图像文件路径**
```json
{
  "image_path": "/path/to/image.jpg"
}
```

2. **Base64编码图像**
```json
{
  "image_data": "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQ..."
}
```

3. **图像URL**
```json
{
  "image_url": "https://example.com/image.jpg"
}
```

### 配置参数

基于[Ultralytics YOLO官方文档](https://docs.ultralytics.com/zh/modes/predict/)，支持以下推理参数：

```json
{
  "config": {
    "model_type": "yolov8",           // YOLO模型类型
    "confidence": 0.5,                // 置信度阈值 (conf)
    "iou_threshold": 0.45,           // IoU阈值 (iou)
    "image_size": 640,                // 输入图像尺寸 (imgsz)
    "max_detections": 100,            // 最大检测数量 (max_det)
    "device": "cpu",                 // 推理设备 (cpu/cuda)
    "half": false,                   // 半精度推理
    "dnn": false,                    // 使用OpenCV DNN
    "vid_stride": 1,                 // 视频帧步长
    "stream": false,                 // 流式推理
    "classes": ["person", "car"]      // 目标类别
  }
}
```

#### 支持的推理参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `conf` | float | 0.25 | 置信度阈值 |
| `iou` | float | 0.7 | IoU阈值 |
| `imgsz` | int | 640 | 输入图像尺寸 |
| `max_det` | int | 300 | 最大检测数量 |
| `device` | str | 'cpu' | 推理设备 |
| `half` | bool | False | 半精度推理 |
| `dnn` | bool | False | 使用OpenCV DNN |
| `vid_stride` | int | 1 | 视频帧步长 |
| `stream` | bool | False | 流式推理模式 |

### 输出格式

```json
{
  "detections": [
    {
      "class_id": 0,
      "class_name": "person",
      "confidence": 0.95,
      "bounding_box": {
        "x": 100.0,
        "y": 150.0,
        "width": 80.0,
        "height": 200.0
      }
    }
  ],
  "image_info": {
    "width": 640,
    "height": 480,
    "channels": 3
  },
  "processing_time": 0.1,
  "model_info": {
    "model_type": "yolov8",
    "confidence_threshold": 0.5,
    "iou_threshold": 0.45
  },
  "timestamp": 1640995200.0
}
```

## 使用示例

### 1. 基本使用流程

```bash
# 1. 启动CNET代理
./bin/cnet-agent -config configs/config_yolo.yaml

# 2. 创建YOLO任务
TASK_ID=$(curl -s -X POST http://localhost:8080/api/yolo/tasks \
  -H "Content-Type: application/json" \
  -d '{"name": "my-yolo-task", "model_path": "/path/to/model.pt", "script_path": "/path/to/script.py"}' \
  | jq -r '.id')

# 3. 执行推理
curl -X POST http://localhost:8080/api/yolo/tasks/$TASK_ID/predict \
  -H "Content-Type: application/json" \
  -d '{"image_path": "/path/to/test.jpg"}'

# 4. 停止任务
curl -X DELETE http://localhost:8080/api/yolo/tasks/$TASK_ID
```

### 2. 使用演示脚本

```bash
# 运行YOLO演示
./examples/yolo_demo.sh

# 运行YOLO测试
./examples/test_yolo.sh
```

## 开发指南

### 1. 添加新的YOLO版本支持

1. 在`YOLOExecutor`中添加新版本的处理逻辑
2. 更新`YOLOConfig`结构体以支持新版本参数
3. 在推理脚本中添加新版本的模型加载代码

### 2. 自定义推理逻辑

修改`examples/yolo_inference.py`中的`run_yolo_inference`方法：

```python
def run_yolo_inference(self, image, request_data):
    # 1. 预处理图像
    processed_image = self.preprocess_image(image)
    
    # 2. 运行模型推理
    results = self.model(processed_image)
    
    # 3. 后处理结果
    detections = self.postprocess_results(results)
    
    return detections
```

### 3. 添加新的检测类别

在配置文件中更新`default_classes`列表：

```yaml
ml:
  yolo:
    default_classes:
      - "person"
      - "bicycle"
      - "car"
      # 添加新类别
      - "motorcycle"
      - "airplane"
```

## 故障排除

### 常见问题

1. **模型加载失败**
   - 检查模型文件路径是否正确
   - 确认模型文件格式是否支持
   - 检查Python依赖是否安装完整

2. **推理速度慢**
   - 调整`image_size`参数
   - 启用GPU加速（设置`CUDA_VISIBLE_DEVICES`）
   - 增加CPU核心数

3. **内存不足**
   - 调整`memory_limit`参数
   - 减少`max_detections`数量
   - 使用更小的模型

4. **检测精度低**
   - 调整`confidence`阈值
   - 检查输入图像质量
   - 确认模型训练数据匹配

### 日志查看

```bash
# 查看任务日志
curl -X GET http://localhost:8080/api/yolo/tasks/{task_id}/logs

# 查看系统日志
tail -f /tmp/cnet/yolo-agent.log
```

## 性能优化

### 1. 硬件优化
- 使用GPU加速推理
- 增加CPU核心数
- 使用SSD存储模型文件

### 2. 软件优化
- 调整批处理大小
- 使用模型量化
- 启用多线程推理

### 3. 网络优化
- 使用本地模型文件
- 压缩图像传输
- 启用HTTP/2

## 扩展功能

### 1. 批量推理
支持多张图像同时推理：

```json
{
  "images": [
    {"path": "/path/to/image1.jpg"},
    {"path": "/path/to/image2.jpg"}
  ]
}
```

### 2. 实时视频流
支持视频流推理：

```json
{
  "video_url": "rtsp://camera/stream",
  "frame_interval": 30
}
```

### 3. 自定义后处理
支持自定义检测结果后处理：

```json
{
  "postprocessing": {
    "filter_classes": ["person", "car"],
    "min_area": 1000,
    "max_area": 50000
  }
}
```

## 总结

YOLO推理引擎执行器为CNET系统提供了强大的目标检测能力，支持多种YOLO版本和灵活的配置选项。通过RESTful API接口，可以轻松集成到各种应用中，实现高效的目标检测服务。
