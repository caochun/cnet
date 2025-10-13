# CNET Agent

CNET Agent 是一个分布式资源管理和任务调度系统。

## 核心架构

### 三大核心组件

1. **Register（资源注册器）**
   - 管理本地资源（CPU、GPU、Memory、Storage）
   - 维护下级节点的资源信息（树状结构）
   - 维护同级peer节点的资源信息
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
Parent Agent
└── Child Agent 1
    └── Child Agent 2
```

2. **对等架构（P2P）**
```
Peer Agent 1 ←→ Peer Agent 2 ←→ Peer Agent 3
```

可以混合使用两种架构。

## Workload类型

支持三种workload类型：

1. **Process**: 本地进程
2. **Container**: Docker容器（简化实现）
3. **MLModel**: ML模型推理服务（简化实现）

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

### 3. 层次化集群

```bash
# 终端1: 启动父节点
./bin/cnet-agent -config configs/config_parent.yaml

# 终端2: 启动子节点
./bin/cnet-agent -config configs/config_child.yaml
```

### 4. P2P集群

```bash
# 终端1: 启动peer1
./bin/cnet-agent -config configs/config_peer1.yaml

# 终端2: 启动peer2
./bin/cnet-agent -config configs/config_peer2.yaml
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

# 所有节点（包括child和peer）
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
2. 下级（child）节点
3. 同级（peer）节点

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

## 目录结构

```
internal/
├── agent/              # Agent主类
│   └── agent_new.go
├── register/           # 资源注册器
│   ├── register.go
│   └── resources.go
├── manager/            # 管理器
│   ├── manager.go
│   └── api.go
├── scheduler/          # 调度器
│   ├── scheduler.go
│   └── strategy.go
├── workload/           # Workload定义
│   ├── workload.go
│   ├── container.go
│   ├── process.go
│   └── mlmodel.go
├── executor/           # 执行器
│   ├── executor.go
│   ├── process_executor.go
│   ├── container_executor.go
│   └── mlmodel_executor.go
└── discovery/          # 节点发现
    ├── parent.go
    └── peer.go
```

## 下一步

- [ ] 完善Container和MLModel执行器的实际实现
- [ ] 实现workload的远程委托（HTTP/gRPC）
- [ ] 添加更多调度策略
- [ ] 实现资源预留机制
- [ ] 添加监控和指标导出
- [ ] 实现Web UI

