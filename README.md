# CNET Agent

CNET Agent 是一个分布式资源管理和任务调度系统，支持进程、容器、机器学习模型推理服务的智能调度与执行。

## 核心架构

### 三大核心组件

1. **Register（资源注册器）**
   - 管理本地资源（CPU、GPU、Memory、Storage）
   - 维护子节点和peer节点的资源信息（树状结构）
   - 提供资源分配和释放功能
   - **触发式心跳**：资源变化时立即通知父节点（2秒内同步）

2. **Scheduler（调度器）**
   - 根据Register的资源信息做出调度决策
   - 本地资源充足时在本地执行
   - 本地资源不足时委托给子节点或peer节点
   - 支持多种调度策略

3. **Manager（管理器）**
   - 接收用户的workload请求
   - 验证和管理workload生命周期
   - 协调Scheduler进行调度
   - 提供RESTful API和Web UI

### 节点关系

支持两种节点组织方式：

1. **层次化架构**
```
江苏省 (parent)
└── 南京市 (child)
```

2. **对等架构（P2P）**
```
南京市 ←→ 宿迁市 ←→ 常州市
```

可以混合使用。示例集群拓扑：
```
                  江苏省 (jiangsu) :8080
                        |
                        ↓
               南京市 (nanjing) :8081
                  /           \
                 /             \
    宿迁市 (suqian) :8082  ←→  常州市 (changzhou) :8083
```

## Workload类型

### 1. Process（进程服务）
运行本地进程，如脚本、命令行工具等。

### 2. Container（容器服务）
运行Docker容器（框架已实现，待完善）。

### 3. MLModel（机器学习模型推理服务）⭐ 核心特性

**设计理念：**
- **模型即服务**：部署ML模型=启动持续运行的HTTP推理服务
- **服务化架构**：每个模型作为独立进程提供推理API
- **资源声明式管理**：用户声明资源需求，Register精确追踪

**架构层次：**
```
MLModelWorkload (用户提交)
    ↓
MLModelExecutorDispatcher (根据model_type分发)
    ↓
具体Executor (如YOLOInferenceExecutor)
    ↓
启动独立的HTTP推理服务进程
    ↓
提供推理API endpoint
```

**支持的模型类型：**
- **YOLO** (YOLOv5/v8/v11) - 目标检测
- TensorFlow (待实现)
- PyTorch (待实现)

**工作流程：**
```
1. 用户提交MLModelWorkload（包含模型路径、资源需求）
2. Register分配资源（CPU、Memory、GPU）
3. YOLOInferenceExecutor 启动推理服务子进程
4. 推理服务加载模型，启动HTTP server
5. 返回推理endpoint（如 http://localhost:9001）
6. 用户通过endpoint调用推理API
7. 服务持续运行，自动健康检查
8. 停止workload时，进程终止，资源释放
```

**推理服务API：**
```bash
# 健康检查
GET http://localhost:9001/health
→ {"status": "healthy", "model": "models/yolo11n.onnx"}

# 服务信息
GET http://localhost:9001/info
→ {"model_type": "yolo", "loaded": true}

# 推理接口
POST http://localhost:9001/predict
Body: {"image": "base64_encoded_image", "confidence": 0.5}
→ {"detections": [...], "count": 16}
```

**资源管理：**
- 用户在workload的`requirements`中声明资源需求
- Register根据声明分配资源配额
- 模型运行期间，资源保持allocated状态
- 触发式心跳：资源变化2秒内同步到父节点
- 停止服务时，资源立即释放

**健康保障：**
- 30秒自动健康检查
- 服务崩溃自动重启（最多3次）
- 详细的日志输出到`yolo_service_PORT.log`

### 4. OpenCV（OpenCV推理服务）

基于 Haar Cascade 的 OpenCV 推理服务，专注于经典 CV 算法。

**支持的功能：**
- 人脸检测（face）
- 眼睛检测（eye）
- 笑脸检测（smile）

## 快速开始

### 1. 编译

```bash
make build
# 产出：
# - bin/cnet-agent               (主程序)
# - bin/cnet-inference-yolo      (YOLO 推理服务)
# - bin/cnet-inference-opencv    (OpenCV 推理服务)
# - bin/cnet-gateway-data        (数据网关服务)
```

### 2. 单节点运行

```bash
./bin/cnet-agent -config config.yaml
```

访问Web UI: http://localhost:8080/

### 3. 启动完整集群

```bash
# 一键启动四节点集群
./start_cluster.sh

# 停止集群
./stop_cluster.sh
```

## API使用示例

### 部署 YOLO 模型推理服务

```bash
curl -X POST http://localhost:8080/api/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "name": "yolo-detection-service",
    "type": "mlmodel",
    "config": {
      "model_type": "yolo",
      "model_path": "models/yolo11n.onnx",
      "service_port": 9001
    },
    "requirements": {
      "cpu": 2.0,
      "memory": 2147483648,
      "gpu": 0
    }
  }'

# 返回：
# {
#   "id": "xxx",
#   "status": "running",
#   "endpoint": "http://localhost:9001",
#   "process_pid": 12345
# }
```

### 调用 YOLO 推理

```bash
# 方式1: 使用base64编码的图片
IMAGE_B64=$(base64 -i test.jpg | tr -d '\n')
curl -X POST http://localhost:9001/predict \
  -H "Content-Type: application/json" \
  -d "{\"image\": \"$IMAGE_B64\", \"confidence\": 0.5}"

# 返回：
# {
#   "detections": [
#     {"class": "class_0", "confidence": 0.85, "bbox": [10, 20, 100, 150]},
#     ...
#   ],
#   "count": 5
# }
```

### 部署 OpenCV 推理服务

```bash
curl -X POST http://localhost:8080/api/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "name": "opencv-face-detection",
    "type": "opencv",
    "config": {
      "cascade_type": "face",
      "service_port": 9000
    },
    "requirements": {
      "cpu": 1.0,
      "memory": 536870912
    }
  }'
```

### 提交 Process Workload

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
      "args": ["60"]
    }
  }'
```

### 查看和管理 Workload

```bash
# 查看所有workload
curl http://localhost:8080/api/workloads

# 查看单个workload
curl http://localhost:8080/api/workloads/{workload_id}

# 停止workload
curl -X POST http://localhost:8080/api/workloads/{workload_id}/stop

# 删除workload
curl -X DELETE http://localhost:8080/api/workloads/{workload_id}
```

### 查看资源信息

```bash
# 本地资源
curl http://localhost:8080/api/resources

# 所有节点（包括子节点和peer节点）
curl http://localhost:8080/api/nodes

# 健康检查
curl http://localhost:8080/api/health
```

## 集群管理

```bash
# 启动集群
./start_cluster.sh

# 停止集群
./stop_cluster.sh
```

## ML 模型部署架构详解

### 核心概念

#### 1. MLModelExecutor 接口

所有 ML 模型执行器遵循统一的服务型契约：

```go
// MLModelExecutor 继承 ServiceExecutor（统一服务型工作负载契约）
type MLModelExecutor interface {
    ServiceExecutor
}
```

#### 2. YOLOInferenceExecutor 实现

**职责：**
- 管理 YOLO 推理服务进程的生命周期
- 启动 `cnet-inference-yolo` 独立进程
- 监控服务健康状态
- 自动重启崩溃的服务

**流程：**
```
Execute() 被调用
  ↓
启动 ./bin/cnet-inference-yolo 子进程
  ↓
等待服务ready（健康检查）
  ↓
启动后台健康监控（30秒间隔）
  ↓
返回推理endpoint
```

#### 3. YOLO 推理服务器

独立的 Go 程序（`cmd/inference/yolo/main.go`）：

**功能：**
- 使用 GoCV 加载 YOLO ONNX 模型
- 提供 HTTP 推理 API
- 处理图片预处理和后处理
- NMS 过滤重复检测

**API端点：**
- `POST /predict` - 推理接口
- `GET /health` - 健康检查
- `GET /info` - 服务信息

**启动示例：**
```bash
./bin/cnet-inference-yolo \
  --model models/yolo11n.onnx \
  --port 9001
```

#### 4. MLModelExecutorDispatcher

根据 `model_type` 智能分发到对应的执行器：

```go
switch mlWorkload.ModelType {
case "yolo":
    return YOLOInferenceExecutor.Execute(...)
case "tensorflow":
    return TensorFlowExecutor.Execute(...)
case "pytorch":
    return PyTorchExecutor.Execute(...)
}
```

### 资源管理机制

#### 用户提交 MLModelWorkload

```json
{
  "name": "yolo-service",
  "type": "mlmodel",
  "config": {
    "model_type": "yolo",
    "model_path": "models/yolo11n.onnx",
    "service_port": 9001
  },
  "requirements": {
    "cpu": 2.0,           // 声明需要2核CPU
    "memory": 2147483648, // 声明需要2GB内存
    "gpu": 0
  }
}
```

#### Register 资源追踪

**部署前：**
```
Available: CPU 8核心, Memory 16GB
Used: CPU 0, Memory 0
```

**部署YOLO后：**
```
Available: CPU 6核心, Memory 14GB  ← 减少了用户声明的资源
Used: CPU 2核心, Memory 2GB        ← 记录已分配的配额
```

**停止服务后：**
```
Available: CPU 8核心, Memory 16GB  ← 恢复到初始值
Used: CPU 0, Memory 0
```

#### 触发式心跳同步

子节点资源变化时：
```
AllocateResources() 被调用
  ↓
资源状态更新
  ↓
触发 resourceChangeCallback
  ↓
ParentConnector.TriggerHeartbeat()
  ↓
立即发送心跳到父节点
  ↓
父节点更新子节点资源信息（2秒内完成）
```

### 推理服务生命周期

```
1. 部署阶段
   用户提交 → Scheduler调度 → Register分配资源 → Executor启动服务
   
2. 运行阶段
   推理服务持续运行 → 30秒健康检查 → 崩溃自动重启（最多3次）
   
3. 使用阶段
   用户调用 endpoint/predict → 推理服务处理 → 返回结果
   
4. 停止阶段
   Stop请求 → 停止健康检查 → Kill服务进程 → 释放资源
```

## 使用示例

### 场景 1: 单节点部署 YOLO 模型

```bash
# 1. 启动agent
./bin/cnet-agent -config config.yaml

# 2. 部署YOLO模型
curl -X POST http://localhost:8080/api/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "name": "yolo-service",
    "type": "mlmodel",
    "config": {
      "model_type": "yolo",
      "model_path": "models/yolo11n.onnx",
      "service_port": 9001
    },
    "requirements": {
      "cpu": 2.0,
      "memory": 2147483648
    }
  }'

# 3. 调用推理
IMAGE_B64=$(base64 -i image.jpg | tr -d '\n')
curl -X POST http://localhost:9001/predict \
  -H "Content-Type: application/json" \
  -d "{\"image\": \"$IMAGE_B64\"}"
```

### 场景 2: 集群部署和资源委托

```bash
# 1. 启动集群
./start_cluster.sh

# 2. 在南京节点部署YOLO（江苏资源不足，会委托给南京）
curl -X POST http://localhost:8080/api/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "name": "yolo",
    "type": "mlmodel",
    "config": {
      "model_type": "yolo",
      "model_path": "models/yolo11n.onnx",
      "service_port": 9001
    },
    "requirements": {
      "cpu": 2.0,
      "memory": 2147483648
    }
  }'

# 3. 查看资源分配情况
curl http://localhost:8081/api/resources  # 南京节点资源
curl http://localhost:8080/api/nodes      # 江苏看到的子节点资源
```

## Web UI

访问任意节点的Web UI查看实时状态：

```
http://localhost:8080/  # 江苏节点
http://localhost:8081/  # 南京节点
http://localhost:8082/  # 宿迁节点
http://localhost:8083/  # 常州节点
```

**Web UI 功能：**
- 📊 本节点信息（节点 ID、地址、状态）
- 🔗 上级节点和 Peer 节点信息（已剔除本节点）
- 💻 资源使用情况（CPU、内存、GPU、存储）
- 📋 工作负载管理（查看、提交、停止、动态表单提交）
- 🧩 动态表单支持：mlmodel / opencv / process / container / data / datagateway
- 🎨 Tailwind CSS 现代化界面 + 简洁交互
- ⚡ 30 秒自动刷新 + 触发式心跳带来的近实时同步

### 数据工作负载（Data）与数据网关（DataGateway）

数据作为独立资源管理：使用 SQLite + 文件系统（/tmp/cnet_storage.db + /tmp/cnet_data）存储元数据与对象。

1) 提交单文件 Data Workload（multipart/form-data）：
```bash
curl -X POST http://localhost:8080/api/workloads \
  -H "Content-Type: multipart/form-data" \
  -F "type=data" \
  -F "name=test-file" \
  -F "file=@./gw_test.txt"
# 返回包含 data_key、最终持久化路径等信息
```

2) 提交目录 Data Workload（目录上传，浏览器提交，后端聚合保存）：
- Web UI 选择目录后自动打包为多文件上传，由后端归档到 /tmp/cnet_data，并写入 SQLite 元数据。

3) 启动 DataGateway Workload（只读 S3 子集接口）：
```bash
curl -X POST http://localhost:8080/api/workloads \
  -H "Content-Type: application/json" \
  -d '{
        "name": "data-gateway",
        "type": "datagateway",
        "config": {
          "service_port": 9091,
          "service_host": "127.0.0.1",
          "base_path": "/tmp/cnet_data",
          "bucket": "cnet"
        }
      }'
# 返回 endpoint，如 http://127.0.0.1:9091
```

4) 通过网关访问对象（只读）：
```bash
# 健康检查
curl http://127.0.0.1:9091/health

# 列举对象（ListObjectsV2 子集）
curl "http://127.0.0.1:9091/s3/cnet?list-type=2&prefix=<data_key>"

# 下载对象
curl http://127.0.0.1:9091/s3/cnet/<data_key>/gw_test.txt
```

## 目录结构

```
cnet/
├── bin/                           # 编译产物
│   ├── cnet-agent                 # 主程序
│   ├── cnet-inference-yolo        # YOLO 推理服务
│   ├── cnet-inference-opencv      # OpenCV 推理服务
│   └── cnet-gateway-data          # 数据网关服务
├── cmd/                           # 命令行程序
│   ├── gateway/                   # 数据网关服务
│   │   └── main.go
│   └── inference/
│       ├── yolo/                  # YOLO推理服务器
│       │   └── main.go
│       └── opencv/                # OpenCV推理服务器
│           └── main.go
├── configs/                       # 配置文件
│   ├── config_jiangsu.yaml        # 江苏省节点
│   ├── config_nanjing.yaml        # 南京市节点
│   ├── config_suqian.yaml         # 宿迁市节点
│   └── config_changzhou.yaml      # 常州市节点
├── internal/                      # 源代码
│   ├── agent/                     # Agent主类
│   ├── register/                  # 资源注册器（含触发式心跳）
│   ├── manager/                   # 管理器（含Web UI）
│   ├── scheduler/                 # 调度器（含委托逻辑）
│   ├── workload/                  # Workload 定义
│   │   ├── workload.go
│   │   ├── process.go
│   │   ├── container.go
│   │   ├── mlmodel.go             # ML 模型 workload
│   │   ├── opencv.go              # OpenCV workload
│   │   └── data.go                # Data / DataGateway workload
│   ├── executor/                  # 执行器
│   │   ├── executor.go            # 基础接口
│   │   ├── service_executor.go    # 服务型接口（统一契约）
│   │   ├── process_executor.go
│   │   ├── container_executor.go
│   │   ├── mlmodel_executor.go    # ML 模型接口（继承 ServiceExecutor）
│   │   ├── mlmodel_executor_dispatcher.go  # 分发器
│   │   ├── yolo_inference_executor.go      # YOLO 推理服务管理
│   │   ├── opencv_inference_executor.go    # OpenCV 推理服务管理
│   │   ├── data_executor.go               # Data 工作负载处理
│   │   └── data_gateway_executor.go       # DataGateway 子进程管理
│   ├── storage/                   # 存储抽象与实现
│   │   ├── storage.go             # StorageBackend 接口与管理器
│   │   ├── sqlite_backend.go      # SQLite + 文件系统实现
│   │   └── errors.go              # 自定义错误
│   ├── discovery/                 # 节点发现
│   │   ├── parent.go              # 父节点连接（含触发式心跳）
│   │   └── peer.go                # Peer发现
│   ├── config/                    # 配置解析
│   ├── http/                      # HTTP客户端
│   └── logger/                    # 日志
├── web/                           # Web UI
│   ├── templates/
│   │   └── index.html
│   └── static/
│       ├── css/
│       └── js/
│           └── app.js
├── models/                        # 模型文件（示例，可选）
├── config.yaml                    # 默认配置
├── main.go                        # 入口文件
├── Makefile                       # 构建脚本
├── start_cluster.sh               # 启动集群
└── stop_cluster.sh                # 停止集群
```

## 架构特点

1. **模块化设计**: Register、Scheduler、Manager职责清晰
2. **ML 模型即服务**: 部署=启动 HTTP 推理服务，持续运行
3. **服务型架构统一**: ML 推理与数据网关均为 ServiceExecutor
4. **实时资源同步**: 触发式心跳，2 秒内同步资源变化
5. **智能调度**: 资源不足时自动委托给子节点或peer
6. **健康保障**: 自动健康检查和服务重启
7. **Web UI**: 实时展示节点状态、资源、workload
8. **生产就绪**: 完整的日志、错误处理、资源管理

## 配置说明

### 基础配置

```yaml
agent:
  node_id: "nanjing"      # 节点ID（唯一）
  address: "0.0.0.0"      # 监听地址
  port: 8081              # 监听端口

resources:
  cpu: 8.0                # CPU核心数
  gpu: 1                  # GPU数量
  memory: 17179869184     # 内存（16GB）
  storage: 214748364800   # 存储（200GB）

parent:
  enabled: true           # 启用父节点连接
  address: "localhost:8080"  # 父节点地址

peers:
  enabled: true           # 启用peer发现
  addresses:
    - "localhost:8082"    # peer节点1
    - "localhost:8083"    # peer节点2

logging:
  level: "info"           # 日志级别
  format: "json"          # 日志格式

storage:
  sqlite:
    db_path: "/tmp/cnet_storage.db"   # 元数据数据库
    data_path: "/tmp/cnet_data"       # 对象数据根目录
```

## 性能特点

### 触发式心跳性能

- **原方案**: 30秒定时心跳，资源同步延迟30秒
- **新方案**: 资源变化立即触发，2秒内同步到父节点
- **提升**: 15倍实时性提升

### YOLO推理性能

- **模型加载**: ~1秒（首次启动）
- **推理延迟**: 取决于图片大小和模型复杂度
- **并发支持**: 单个服务支持多个并发请求

## 下一步计划

- [x] Vision workload支持
- [x] YOLO模型集成
- [x] 模型缓存机制
- [x] 任务委托功能
- [x] Web UI实现
- [x] ML模型即服务架构
- [x] 触发式心跳机制
- [ ] TensorFlow/PyTorch executor实现
- [ ] GPU资源调度和追踪
- [ ] 更多YOLO模型支持（YOLOv7/v9等）
- [ ] 推理服务水平扩展（同一模型多实例）
- [ ] 监控和指标导出（Prometheus）
- [ ] 容器化部署（Docker/K8s）

## 许可证

MIT License
