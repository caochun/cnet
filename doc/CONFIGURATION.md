# CNET Agent 配置指南

## 📋 配置文件概述

CNET Agent 使用 YAML 格式的配置文件来管理所有设置。主配置文件为 `config.yaml`，支持多种配置选项来满足不同的部署需求。

## ⚙️ 基础配置

### 主配置文件 `config.yaml`

```yaml
agent:
  address: "0.0.0.0"
  port: 8080
  node_id: ""  # 自动生成
  node_name: ""  # 使用主机名
  region: "default"
  datacenter: "dc1"
  heartbeat: "30s"

logging:
  level: "info"
  format: "json"

discovery:
  enabled: true
  servers:
    - "localhost:8080"
  timeout: "5s"
  retry: 3

resources:
  cpu: true
  memory: true
  disk: true
  network: true
  interval: "10s"

tasks:
  max_concurrent: 10
  timeout: "5m"
  cleanup: true

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

## 🔧 配置选项详解

### Agent 配置

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `address` | string | "0.0.0.0" | Agent监听地址 |
| `port` | int | 8080 | Agent监听端口 |
| `node_id` | string | "" | 节点ID（空值自动生成） |
| `node_name` | string | "" | 节点名称（空值使用主机名） |
| `region` | string | "default" | 节点所属区域 |
| `datacenter` | string | "dc1" | 数据中心标识 |
| `heartbeat` | string | "30s" | 心跳间隔 |

### 日志配置

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `level` | string | "info" | 日志级别 (debug, info, warn, error) |
| `format` | string | "json" | 日志格式 (json, text) |

### 发现服务配置

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `enabled` | bool | true | 是否启用发现服务 |
| `servers` | array | [] | 发现服务器列表 |
| `timeout` | string | "5s" | 连接超时时间 |
| `retry` | int | 3 | 重试次数 |

### 资源监控配置

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `cpu` | bool | true | 是否监控CPU |
| `memory` | bool | true | 是否监控内存 |
| `disk` | bool | true | 是否监控磁盘 |
| `network` | bool | true | 是否监控网络 |
| `interval` | string | "10s" | 监控间隔 |

### 任务配置

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `max_concurrent` | int | 10 | 最大并发任务数 |
| `timeout` | string | "5m" | 任务超时时间 |
| `cleanup` | bool | true | 是否自动清理任务 |

### 机器学习配置

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `enabled` | bool | true | 是否启用ML功能 |
| `engines` | array | ["python", "tensorflow", "pytorch"] | 支持的推理引擎 |
| `default_engine` | string | "python" | 默认推理引擎 |
| `model_path` | string | "./models" | 模型文件路径 |
| `script_path` | string | "./examples/ml_models" | 脚本文件路径 |
| `port_range.start` | int | 9000 | ML服务端口范围起始 |
| `port_range.end` | int | 9100 | ML服务端口范围结束 |
| `resource_limits.cpu_limit` | float | 1.0 | CPU限制 |
| `resource_limits.memory_limit` | int | 536870912 | 内存限制（字节） |
| `resource_limits.disk_limit` | int | 1073741824 | 磁盘限制（字节） |
| `resource_limits.gpu_limit` | int | 0 | GPU限制 |
| `timeout` | string | "30s" | ML任务超时时间 |

## 🌐 集群配置

### 多层级集群配置

CNET Agent支持多级层次化集群部署，可以实现复杂的节点层次结构。项目提供了完整的层次化集群配置文件：

#### 根节点 (discovery-server) - `config.yaml`
```yaml
agent:
  address: "0.0.0.0"
  port: 8080
  node_id: "discovery-server"
  node_name: "Discovery Server"
  region: "default"
  datacenter: "dc1"
  heartbeat: "30s"

discovery:
  enabled: true
  servers: []  # 根节点，不向其他服务器注册
  timeout: "5s"
  retry: 3
```

#### Level 2节点 - `config_level2.yaml`
```yaml
agent:
  address: "0.0.0.0"
  port: 8082
  node_id: "level2-node"
  node_name: "Level 2 Node"
  region: "us-west"
  datacenter: "dc1"
  heartbeat: "30s"

discovery:
  enabled: true
  servers:
    - "localhost:8080"  # 向根节点注册
  timeout: "5s"
  retry: 3
```

#### Level 3节点 - `config_level3.yaml`
```yaml
agent:
  address: "0.0.0.0"
  port: 8083
  node_id: "level3-node"
  node_name: "Level 3 Node"
  region: "us-west"
  datacenter: "dc1"
  heartbeat: "30s"

discovery:
  enabled: true
  servers:
    - "localhost:8082"  # 向Level 2节点注册
  timeout: "5s"
  retry: 3
```

#### Level 4节点 - `config_level4_node1.yaml`
```yaml
agent:
  address: "0.0.0.0"
  port: 8084
  node_id: "level4-node1"
  node_name: "Level 4 Node 1"
  region: "us-west"
  datacenter: "dc1"
  heartbeat: "30s"

discovery:
  enabled: true
  servers:
    - "localhost:8083"  # 向Level 3节点注册
  timeout: "5s"
  retry: 3
```

### 层次化集群架构

```
discovery-server (34.23.1) - 根节点
└── level2-node (34.23.1.1) - Level 2节点
    └── level3-node (34.23.1.1.1) - Level 3节点
        ├── level4-node1 (34.23.1.1.1.1) - Level 4节点1
        └── level4-node2 (34.23.1.1.1.2) - Level 4节点2
```

**层次化特性**:
- 🌳 **多级层次**: 支持无限层级的节点层次结构
- 🏷️ **自动标识**: 自动分配唯一的层次化标识
- 🔄 **动态注册**: 节点可动态注册到上级节点
- 🔒 **线程安全**: 完全线程安全的并发访问
- ⚡ **高性能**: 优化的锁机制和算法

### 集群功能

- **层次化发现**: 支持多级节点层次结构
- **自动标识分配**: 自动为节点分配层次化标识
- **负载分布**: 任务可以在不同层级的节点上执行
- **资源监控**: 每个节点独立监控资源
- **Web UI**: 每个节点都有独立的管理界面
- **API接口**: 统一的RESTful API接口
- **标识解析**: 支持层次化标识的快速解析和查找

## 🚀 启动层次化集群

```bash
# 1. 启动根节点 (discovery-server)
./bin/cnet-agent -config config.yaml > discovery-server.log 2>&1 &
sleep 3

# 2. 启动Level 2节点
./bin/cnet-agent -config configs/config_level2.yaml > level2.log 2>&1 &
sleep 3

# 3. 启动Level 3节点
./bin/cnet-agent -config configs/config_level3.yaml > level3.log 2>&1 &
sleep 3

# 4. 启动Level 4节点
./bin/cnet-agent -config configs/config_level4_node1.yaml > level4_node1.log 2>&1 &
./bin/cnet-agent -config configs/config_level4_node2.yaml > level4_node2.log 2>&1 &

# 查看所有节点状态
curl http://localhost:8080/api/health  # 根节点
curl http://localhost:8082/api/health  # Level 2
curl http://localhost:8083/api/health  # Level 3
curl http://localhost:8084/api/health  # Level 4节点1
curl http://localhost:8085/api/health  # Level 4节点2
```

## 🔍 层次化集群验证

```bash
# 查看根节点的层次化结构
curl http://localhost:8080/api/discovery/hierarchy/nodes | jq .

# 查看Level 2节点的子节点
curl http://localhost:8082/api/discovery/hierarchy/nodes | jq .

# 查看Level 3节点的子节点
curl http://localhost:8083/api/discovery/hierarchy/nodes | jq .

# 查看完整的层次化结构
echo "=== 完整的层次化结构 ==="
echo "discovery-server (34.23.1)"
echo "└── level2-node (34.23.1.1)"
echo "    └── level3-node (34.23.1.1.1)"
echo "        ├── level4-node1 (34.23.1.1.1.1)"
echo "        └── level4-node2 (34.23.1.1.1.2)"
```

## 📝 配置最佳实践

### 1. 生产环境配置
- 设置合适的日志级别（info或warn）
- 配置适当的资源限制
- 启用任务清理功能
- 设置合理的超时时间

### 2. 开发环境配置
- 使用debug日志级别
- 启用所有资源监控
- 设置较短的超时时间便于调试

### 3. 集群配置
- 确保端口不冲突
- 配置正确的发现服务器地址
- 设置合适的区域和数据中心标识
- 启用层次化集群功能

### 4. 安全配置
- 限制监听地址（生产环境避免0.0.0.0）
- 配置适当的资源限制
- 启用任务清理防止资源泄漏

## 🔧 环境变量支持

CNET Agent 支持通过环境变量覆盖配置：

```bash
# 设置节点ID
export CNET_NODE_ID="my-node-1"

# 设置端口
export CNET_PORT="8080"

# 设置区域
export CNET_REGION="us-west"

# 设置数据中心
export CNET_DATACENTER="dc1"
```

## 📚 配置文件位置

- **主配置文件**: `config.yaml`
- **集群配置文件**: `configs/` 目录
- **示例配置文件**: `examples/` 目录
- **文档**: `doc/` 目录

## 🆘 故障排除

### 常见问题

1. **端口冲突**: 检查端口是否被占用
2. **配置文件格式错误**: 验证YAML语法
3. **发现服务连接失败**: 检查网络连接和防火墙设置
4. **资源监控异常**: 检查系统权限和资源可用性

### 调试技巧

1. 使用debug日志级别获取详细信息
2. 检查系统日志和Agent日志
3. 验证网络连接和端口可用性
4. 使用健康检查API监控状态

## 📞 支持

如有配置问题，请参考：
- [项目文档](../README.md)
- [层次化集群功能](./HIERARCHY_FEATURE.md)
- [ML模型部署](./ML_MODEL_DEPLOYMENT.md)
- 创建Issue获取帮助
