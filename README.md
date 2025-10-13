# CNET Agent

CNET Agent 是一个分布式资源管理和任务调度系统，支持进程、容器、机器学习模型和计算机视觉任务的智能调度与执行。

## 核心架构

### 三大核心组件

1. **Register（资源注册器）**
   - 管理本地资源（CPU、GPU、Memory、Storage）
   - 维护下级节点的资源信息（树状结构）
   - 维护同级节点的资源信息
   - 提供资源分配和释放功能

2. **Scheduler（调度器）**
   - 根据Register的资源信息做出调度决策
   - 本地资源充足时在本地执行
   - 本地资源不足时委托给下级或同级节点
   - 支持多种调度策略（默认策略、最佳适配等）

3. **Manager（管理器）**
   - 接收用户的workload请求
   - 验证和管理workload生命周期
   - 协调Scheduler进行调度
   - 提供HTTP API接口

### 节点关系

支持两种节点组织方式：

1. **层次化架构（Hierarchical）**
```
上级节点
└── 下级节点1
    └── 下级节点2
```

2. **对等架构（P2P）**
```
节点A ←→ 节点B ←→ 节点C
```

可以混合使用两种架构。

## Workload类型

支持四种workload类型：

1. **Process**: 本地进程
2. **Container**: Docker容器（简化实现）
3. **MLModel**: ML模型推理服务（简化实现）
4. **Vision**: 计算机视觉任务（基于GoCV/OpenCV）
   - 人脸检测（Haar Cascade）
   - 目标检测（YOLO、DNN）
   - 图像分类
   - 视频处理
   - 模型自动缓存机制

## 快速开始

### 1. 编译

```bash
# 编译
make build

# 或者
go build -o bin/cnet-agent main.go
```

### 2. 单节点运行

```bash
# 使用默认配置
./bin/cnet-agent -config config.yaml
```

### 3. 启动完整集群

```bash
# 使用脚本一键启动所有节点
./start_cluster.sh

# 集群拓扑：
#                  江苏省 (jiangsu) :8080
#                        |
#                        ↓
#               南京市 (nanjing) :8081
#                  /           \
#                 /             \
#    宿迁市 (suqian) :8082  ←→  常州市 (changzhou) :8083

# 停止集群
./stop_cluster.sh
```

### 4. 手动启动节点

```bash
# 终端1: 启动江苏省节点（父节点）
./bin/cnet-agent -config configs/config_jiangsu.yaml

# 终端2: 启动南京市节点（子节点）
./bin/cnet-agent -config configs/config_nanjing.yaml

# 终端3: 启动宿迁市节点（对等节点）
./bin/cnet-agent -config configs/config_suqian.yaml

# 终端4: 启动常州市节点（对等节点）
./bin/cnet-agent -config configs/config_changzhou.yaml
```

## API使用

### 提交Process Workload

```bash
curl -X POST http://localhost:8080/api/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-process",
    "type": "process",
    "requirements": {
      "cpu": 1.0,
      "memory": 536870912
    },
    "config": {
      "command": "sleep",
      "args": ["60"],
      "env": {"TEST": "value"}
    }
  }'
```

### 提交Container Workload

```bash
curl -X POST http://localhost:8080/api/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-container",
    "type": "container",
    "requirements": {
      "cpu": 1.0,
      "memory": 536870912
    },
    "config": {
      "image": "nginx:alpine",
      "command": ["nginx"],
      "args": ["-g", "daemon off;"]
    }
  }'
```

### 提交MLModel Workload

```bash
curl -X POST http://localhost:8080/api/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-ml-model",
    "type": "mlmodel",
    "requirements": {
      "cpu": 2.0,
      "memory": 2147483648
    },
    "config": {
      "model_path": "models/model.joblib",
      "model_type": "sklearn",
      "framework": "scikit-learn",
      "port": 9000
    }
  }'
```

### 提交Vision Workload

#### 人脸检测（Haar Cascade）

```bash
curl -X POST http://localhost:8080/api/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "name": "face-detection",
    "type": "vision",
    "requirements": {
      "cpu": 1.0,
      "memory": 536870912
    },
    "config": {
      "task": "face_detection",
      "input_path": "test_images/test.jpg",
      "output_path": "test_output/result.jpg"
    }
  }'
```

#### 目标检测（YOLO）

```bash
curl -X POST http://localhost:8080/api/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "name": "yolo-detection",
    "type": "vision",
    "requirements": {
      "cpu": 2.0,
      "memory": 1073741824
    },
    "config": {
      "task": "detection",
      "model_type": "yolo",
      "model_path": "models/yolov5s.onnx",
      "input_path": "test_images/test.jpg",
      "output_path": "test_output/yolo_result.jpg",
      "confidence": 0.5,
      "nms_threshold": 0.4
    }
  }'
```

#### 图像分类

```bash
curl -X POST http://localhost:8080/api/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "name": "classification",
    "type": "vision",
    "requirements": {
      "cpu": 2.0,
      "memory": 1073741824
    },
    "config": {
      "task": "classification",
      "model_path": "models/resnet50.onnx",
      "input_path": "test_images/test.jpg"
    }
  }'
```

### 查看所有Workload

```bash
curl http://localhost:8080/api/workloads
```

### 查看单个Workload

```bash
curl http://localhost:8080/api/workloads/{workload_id}
```

### 获取Workload日志

```bash
curl http://localhost:8080/api/workloads/{workload_id}/logs?lines=100
```

### 停止Workload

```bash
curl -X POST http://localhost:8080/api/workloads/{workload_id}/stop
```

### 删除Workload

```bash
curl -X DELETE http://localhost:8080/api/workloads/{workload_id}
```

### 查看资源信息

```bash
# 本地资源
curl http://localhost:8080/api/resources

# 资源统计
curl http://localhost:8080/api/resources/stats

# 所有节点（包括下级和同级）
curl http://localhost:8080/api/nodes
```

### 健康检查

```bash
curl http://localhost:8080/api/health
```

## 调度策略

### 默认策略（DefaultScheduleStrategy）

优先级顺序：
1. 本地资源
2. 下级节点
3. 同级节点

### 最佳适配策略（BestFitStrategy）

选择资源最接近需求的节点，避免资源浪费。

切换策略：
```go
// 在代码中设置
scheduler.SetStrategy(&scheduler.BestFitStrategy{})
```

## 配置说明

### 基础配置

```yaml
agent:
  node_id: "agent-1"      # 节点ID（唯一）
  address: "0.0.0.0"      # 监听地址
  port: 8080              # 监听端口

resources:
  cpu: 4.0                # CPU核心数
  gpu: 0                  # GPU数量
  memory: 8589934592      # 内存（字节）
  storage: 107374182400   # 存储（字节）
```

### 父节点配置

```yaml
parent:
  enabled: true
  address: "parent-host:8080"
```

### Peer节点配置

```yaml
peers:
  enabled: true
  addresses:
    - "peer1:8080"
    - "peer2:8080"
```

## 架构特点

1. **模块化设计**: Register、Scheduler、Manager职责清晰
2. **可扩展性**: 支持自定义调度策略和workload类型
3. **灵活部署**: 支持层次化和P2P两种架构
4. **资源感知**: 智能的资源调度和分配
5. **故障处理**: 节点健康监控和自动标记
6. **简洁API**: RESTful API设计
7. **智能委托**: 资源不足时自动委托给子节点或对等节点
8. **模型缓存**: Vision模型首次使用时加载并缓存到内存，后续任务直接复用（性能提升3倍以上）

## 目录结构

```
cnet/
├── bin/                    # 编译后的可执行文件
├── configs/                # 配置文件
│   ├── config_jiangsu.yaml
│   ├── config_nanjing.yaml
│   ├── config_suqian.yaml
│   └── config_changzhou.yaml
├── internal/               # 源代码
│   ├── agent/              # Agent主类
│   │   └── agent.go
│   ├── register/           # 资源注册器
│   │   ├── register.go
│   │   └── resources.go
│   ├── manager/            # 管理器
│   │   ├── manager.go
│   │   └── api.go
│   ├── scheduler/          # 调度器
│   │   ├── scheduler.go
│   │   ├── strategy.go
│   │   └── delegate.go
│   ├── workload/           # Workload定义
│   │   ├── workload.go
│   │   ├── container.go
│   │   ├── process.go
│   │   ├── mlmodel.go
│   │   └── vision.go
│   ├── executor/           # 执行器
│   │   ├── executor.go
│   │   ├── process_executor.go
│   │   ├── container_executor.go
│   │   ├── mlmodel_executor.go
│   │   └── vision_executor.go
│   └── discovery/          # 节点发现
│       ├── parent.go
│       └── peer.go
├── models/                 # 模型文件
│   └── yolov5s.onnx       # YOLOv5s模型 (14MB)
├── test_images/            # 测试输入图片
├── test_output/            # 测试输出结果
├── logs/                   # 日志文件
├── *.sh                    # 测试和管理脚本
├── config.yaml             # 默认配置
├── main.go                 # 入口文件
├── README.md               # 项目文档
└── VISION_GUIDE.md         # Vision功能详细指南
```

## 测试脚本

```bash
# 基础功能测试
./test_agent.sh

# 集群功能测试
./test_cluster.sh

# Vision功能测试
./test_vision.sh

# YOLO模型测试
./test_yolo.sh

# 模型缓存性能测试
./test_model_cache.sh

# 任务委托测试
./test_delegation.sh
```

## Vision功能特色

### 支持的任务类型

1. **人脸检测** (face_detection)
   - 使用 Haar Cascade 算法
   - 高性能，低资源消耗
   - 自动标注检测框

2. **目标检测** (detection)
   - 支持 YOLO (YOLOv5s ONNX)
   - 支持 DNN (MobileNet-SSD, ResNet等)
   - 可检测80种常见物体
   - 可调置信度和NMS阈值

3. **图像分类** (classification)
   - 支持主流分类模型
   - ONNX格式模型
   - 返回Top-K预测结果

4. **视频处理** (tracking)
   - 视频信息提取
   - 帧提取和分析

### 模型缓存机制

- **自动缓存**: 模型首次使用时自动加载到内存
- **高性能**: 后续任务直接使用缓存，性能提升3倍以上
- **智能管理**: 线程安全，支持并发访问
- **零配置**: 无需手动预加载，按需加载

示例性能对比：
```
首次任务: 122ms (包含模型加载)
后续任务: 36-43ms (直接使用缓存)
性能提升: 约3倍
```

### YOLOv5s模型

已集成YOLOv5s (ONNX格式)，可检测80种物体：
- 人物: person
- 交通工具: car, bus, truck, bicycle, motorcycle
- 动物: dog, cat, bird, horse, sheep, cow
- 物品: bottle, cup, fork, knife, laptop, mouse, keyboard
- 家具: chair, couch, bed, dining table
- 等等...

## 下一步计划

- [x] Vision workload支持（已完成）
- [x] YOLO模型集成（已完成）
- [x] 模型缓存机制（已完成）
- [x] 任务委托功能（已完成）
- [ ] 完善Container和MLModel执行器的实际实现
- [ ] 添加更多调度策略
- [ ] 实现资源预留机制
- [ ] 添加监控和指标导出
- [ ] 实现Web UI
- [ ] GPU加速支持（CUDA）
- [ ] 更多Vision模型支持

