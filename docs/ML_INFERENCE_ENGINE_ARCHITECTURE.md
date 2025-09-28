# CNET Agent 模型推理引擎架构设计

## 🎯 设计理念

CNET Agent 现在将模型推理引擎作为节点的基础服务集成进来，就像容器运行时一样，这样可以支持多种形态的工作负载：

- **进程（process）** - 本地进程执行
- **容器（container）** - Docker容器执行  
- **虚拟机（vm）** - 虚拟机执行
- **模型推理（ml）** - 机器学习模型推理服务

## 🏗️ 架构概览

```
┌─────────────────────────────────────────────────────────────┐
│                    CNET Agent Node                         │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │   Process   │  │  Container   │  │     VM      │        │
│  │  Executor  │  │  Executor   │  │  Executor   │        │
│  └─────────────┘  └─────────────┘  └─────────────┘        │
│                                                             │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │              ML Inference Engine                        │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐   │ │
│  │  │   Python    │  │ TensorFlow   │  │  PyTorch    │   │ │
│  │  │   Engine    │  │   Engine     │  │   Engine    │   │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘   │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                             │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                Task Management                          │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐   │ │
│  │  │   Process   │  │  Container  │  │     VM      │   │ │
│  │  │   Tasks     │  │   Tasks     │  │   Tasks     │   │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘   │ │
│  │  ┌─────────────────────────────────────────────────────┐ │ │
│  │  │              ML Inference Tasks                     │ │ │
│  │  └─────────────────────────────────────────────────────┘ │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## 🔧 核心组件

### 1. ML推理引擎服务 (MLService)

**位置**: `internal/agent/ml/service.go`

**功能**:
- 管理模型生命周期（加载、卸载、运行）
- 支持多种推理引擎（Python、TensorFlow、PyTorch）
- 提供模型推理API
- 资源管理和监控

**主要接口**:
```go
type Service struct {
    config    *config.Config
    logger    *logger.Logger
    models    map[string]*Model
    engines   map[string]InferenceEngine
}

type InferenceEngine interface {
    LoadModel(model *Model) error
    UnloadModel(model *Model) error
    Infer(model *Model, request *InferenceRequest) (*InferenceResponse, error)
    GetModelInfo(model *Model) (map[string]interface{}, error)
    HealthCheck(model *Model) error
}
```

### 2. ML执行器 (MLExecutor)

**位置**: `internal/agent/tasks/ml_executor.go`

**功能**:
- 作为任务执行器，支持ML推理任务
- 自动生成Python推理服务器脚本
- 管理推理服务的生命周期
- 提供日志和监控功能

**特点**:
- 自动端口分配
- 动态脚本生成
- 资源限制支持
- 健康检查

### 3. 统一任务管理

**集成方式**:
- ML推理作为新的任务类型 `ml`
- 使用相同的任务管理API
- 统一的资源限制和监控
- 一致的Web UI界面

## 📋 支持的推理引擎

### 1. Python推理引擎
- **用途**: 通用Python模型推理
- **支持框架**: scikit-learn, pandas, numpy等
- **特点**: 轻量级，快速启动

### 2. TensorFlow推理引擎
- **用途**: TensorFlow模型推理
- **支持格式**: SavedModel, H5, TFLite
- **特点**: 高性能，GPU支持

### 3. PyTorch推理引擎
- **用途**: PyTorch模型推理
- **支持格式**: TorchScript, ONNX
- **特点**: 动态图支持，灵活配置

## 🚀 使用方式

### 1. 作为ML推理服务

```bash
# 创建ML模型
curl -X POST "http://localhost:8080/api/ml/models" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "linear-regression-model",
    "type": "linear_regression",
    "engine": "python",
    "model_path": "models/model.joblib",
    "script_path": "examples/ml_models/simple_linear_regression.py",
    "resources": {
      "cpu_limit": 1.0,
      "memory_limit": 536870912,
      "disk_limit": 1073741824
    }
  }'

# 进行推理
curl -X POST "http://localhost:8080/api/ml/models/{model_id}/predict" \
  -H "Content-Type: application/json" \
  -d '{"input_data": 5.5}'
```

### 2. 作为ML任务

```bash
# 创建ML推理任务
curl -X POST "http://localhost:8080/api/tasks" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ml-inference-task",
    "type": "ml",
    "command": "python3",
    "args": ["examples/ml_models/simple_linear_regression.py", "predict", "models/model.joblib", "5.5"],
    "env": {
      "PYTHONPATH": "examples/ml_models",
      "MODEL_PATH": "models/model.joblib"
    },
    "resources": {
      "cpu_limit": 1.0,
      "memory_limit": 512000000
    }
  }'
```

## 🔧 配置选项

### ML服务配置

```yaml
ml:
  enabled: true
  engines: ["python", "tensorflow", "pytorch"]
  default_engine: "python"
  model_path: "./models"
  script_path: "./examples/ml_models"
  port_range:
    start: 9000
    end: 9100
  resource_limits:
    cpu_limit: 1.0
    memory_limit: 536870912  # 512MB
    disk_limit: 1073741824   # 1GB
    gpu_limit: 0
  timeout: "30s"
```

### 模型配置

```json
{
  "name": "my-model",
  "type": "linear_regression",
  "engine": "python",
  "model_path": "models/model.joblib",
  "script_path": "examples/ml_models/script.py",
  "config": {
    "framework": "sklearn",
    "version": "1.0.0",
    "input_shape": [1],
    "output_shape": [1],
    "preprocessing": {},
    "postprocessing": {}
  },
  "resources": {
    "cpu_limit": 1.0,
    "memory_limit": 536870912,
    "disk_limit": 1073741824,
    "gpu_limit": 0
  }
}
```

## 🌐 API接口

### ML模型管理

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/ml/models` | 列出所有模型 |
| POST | `/api/ml/models` | 创建新模型 |
| GET | `/api/ml/models/{id}` | 获取模型信息 |
| DELETE | `/api/ml/models/{id}` | 停止模型 |
| POST | `/api/ml/models/{id}/predict` | 模型推理 |
| GET | `/api/ml/models/{id}/logs` | 获取模型日志 |
| GET | `/api/ml/models/{id}/info` | 获取模型详细信息 |
| GET | `/api/ml/models/{id}/health` | 模型健康检查 |

### 任务管理（扩展）

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/tasks` | 列出所有任务（包括ML任务） |
| POST | `/api/tasks` | 创建新任务（支持type: "ml"） |
| GET | `/api/tasks/{id}` | 获取任务信息 |
| DELETE | `/api/tasks/{id}` | 停止任务 |

## 🎨 Web UI功能

### ML模型管理页面

- **模型列表**: 显示所有部署的模型
- **模型创建**: 图形化创建新模型
- **模型监控**: 实时监控模型状态
- **模型测试**: 在线测试模型推理
- **日志查看**: 查看模型运行日志
- **资源监控**: 监控模型资源使用

### 功能特性

- **统一界面**: ML模型与普通任务使用相同的Web UI
- **实时更新**: 自动刷新模型状态
- **交互式测试**: 直接在界面中测试模型
- **日志管理**: 实时查看模型日志
- **资源监控**: 监控CPU、内存使用情况

## 🔄 工作流程

### 1. 模型部署流程

```
用户请求 → API服务器 → ML服务 → 推理引擎 → 模型加载 → 服务启动
```

### 2. 推理请求流程

```
推理请求 → API服务器 → ML服务 → 推理引擎 → 模型推理 → 返回结果
```

### 3. 任务执行流程

```
任务创建 → 任务服务 → ML执行器 → 推理服务器 → 任务完成
```

## 📊 监控和管理

### 模型状态

- **pending** - 等待加载
- **loading** - 正在加载
- **running** - 运行中
- **stopped** - 已停止
- **failed** - 加载失败

### 资源监控

- CPU使用率
- 内存使用量
- 磁盘使用量
- 网络I/O
- 推理延迟

### 日志管理

- 模型加载日志
- 推理请求日志
- 错误日志
- 性能日志

## 🚀 优势特性

### 1. 统一架构
- 所有工作负载使用相同的管理接口
- 一致的资源管理和监控
- 统一的Web UI界面

### 2. 灵活部署
- 支持多种推理引擎
- 可配置的资源限制
- 动态端口分配

### 3. 高性能
- 模型预加载
- 并发推理支持
- 资源隔离

### 4. 易于扩展
- 插件化推理引擎
- 标准化接口
- 模块化设计

## 🔮 未来规划

- [ ] GPU加速支持
- [ ] 模型版本管理
- [ ] 自动扩缩容
- [ ] 分布式推理
- [ ] 模型市场
- [ ] 性能优化
- [ ] 安全加固

## 📝 总结

CNET Agent的模型推理引擎架构实现了：

✅ **统一管理**: ML模型与普通任务使用相同的管理接口  
✅ **灵活部署**: 支持多种推理引擎和执行方式  
✅ **高性能**: 优化的推理性能和资源管理  
✅ **易于使用**: 图形化界面和RESTful API  
✅ **可扩展**: 插件化架构，易于添加新功能  

这种设计确保了机器学习模型与CNET Agent的整体架构完美集成，提供了统一、灵活、高性能的ML模型部署和推理解决方案。

---

**CNET Agent 让机器学习模型部署变得简单而统一！** 🚀
