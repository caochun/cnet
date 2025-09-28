# CNET Agent

CNET Agent 是一个简化的分布式计算节点代理，灵感来自 HashiCorp Nomad。它提供了在计算节点上运行工作负载、监控资源状态和节点发现的核心功能。

## ✨ 功能特性

- **🚀 工作负载管理**: 支持本地进程执行，容器和虚拟机支持（开发中）
- **📊 资源监控**: 实时监控 CPU、内存、磁盘和网络使用情况
- **🌐 节点发现**: 支持节点注册和发现，构建分布式集群
- **🏷️ 层次化标识**: 支持层次化节点标识分配和解析，如 34.23.1.8
- **🌳 层次化集群**: 支持多级层次化集群结构，节点可注册到上级节点
- **🔒 线程安全**: 完全线程安全的discovery服务，支持并发访问
- **⚡ 高性能**: 优化的锁机制和算法，确保高性能运行
- **🔌 RESTful API**: 提供完整的 HTTP API 接口
- **💻 Web UI界面**: 现代化的Web管理界面，支持实时监控和任务管理
- **⚙️ 配置管理**: 灵活的 YAML 配置文件
- **📝 日志管理**: 结构化日志输出和任务日志收集
- **🐳 Docker支持**: 完整的容器化部署方案
- **🔍 集群监控**: 实时显示注册节点和集群状态
- **✅ 输入验证**: 完整的输入验证和错误处理
- **🔄 自动重注册**: 支持节点重复注册和状态更新

## 架构设计

### 基础架构
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   CNET Agent    │    │   CNET Agent    │    │   CNET Agent    │
│   (Node 1)      │    │   (Node 2)      │    │   (Node 3)      │
├─────────────────┤    ├─────────────────┤    ├─────────────────┤
│ - Task Manager  │    │ - Task Manager  │    │ - Task Manager  │
│ - Resource Mon │    │ - Resource Mon  │    │ - Resource Mon  │
│ - Discovery     │    │ - Discovery     │    │ - Discovery     │
│ - HTTP API      │    │ - HTTP API      │    │ - HTTP API      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │ Discovery Server│
                    │ (Optional)      │
                    └─────────────────┘
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

## 🚀 快速开始

### 1. 一键启动

```bash
# 克隆项目
git clone <repository-url>
cd cnet

# 一键启动（构建+运行）
./start.sh
```

### 2. 手动构建和运行

```bash
# 构建
make build

# 运行
make run

# 或直接运行
./bin/cnet-agent -config config.yaml
```

### 3. 使用 Docker

```bash
# 构建镜像
docker build -t cnet-agent .

# 运行容器
docker run -p 8080:8080 -v $(pwd)/config.yaml:/app/config.yaml cnet-agent
```

### 4. 使用 Docker Compose

```bash
# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f
```

### 5. 访问 Web UI

启动 Agent 后，打开浏览器访问：
- **🌐 Web UI**: http://localhost:8080
- **🔌 API 健康检查**: http://localhost:8080/api/health
- **📊 仪表板**: 实时资源监控和任务管理
- **🌐 节点发现**: 查看注册的节点和集群状态

## 配置

配置文件 `config.yaml` 包含以下主要部分：

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
```

## 🌐 集群配置

### 双Agent配置

项目提供了两个预配置的Agent配置文件，用于演示多节点集群功能：

#### Agent 1 (发现服务器) - `config_agent1.yaml`
```yaml
agent:
  address: "0.0.0.0"
  port: 8080
  node_id: "discovery-server"
  node_name: "Discovery Server"
  region: "us-west"
  datacenter: "dc1"
  heartbeat: "30s"

discovery:
  enabled: true
  servers: []  # 作为发现服务器，不向其他服务器注册
  timeout: "5s"
  retry: 3
```

#### Agent 2 (工作节点) - `config_agent2.yaml`
```yaml
agent:
  address: "0.0.0.0"
  port: 8081
  node_id: "worker-node"
  node_name: "Worker Node"
  region: "us-west"
  datacenter: "dc1"
  heartbeat: "30s"

discovery:
  enabled: true
  servers:
    - "localhost:8080"  # 向Agent 1注册
  timeout: "5s"
  retry: 3
```

### 集群架构

```
┌─────────────────┐    ┌─────────────────┐
│   Agent 1       │    │   Agent 2       │
│   (发现服务器)    │    │   (工作节点)      │
│   Port: 8080    │    │   Port: 8081    │
├─────────────────┤    ├─────────────────┤
│ - 接受节点注册   │    │ - 向Agent 1注册  │
│ - 维护节点列表   │    │ - 执行任务      │
│ - 提供发现服务   │    │ - 资源监控      │
│ - 执行任务      │    │ - 任务管理      │
└─────────────────┘    └─────────────────┘
         │                       │
         └───────────────────────┘
                │
        ┌─────────────────┐
        │   发现协议      │
        │   HTTP API     │
        └─────────────────┘
```

### 集群功能

- **节点发现**: Agent 2自动向Agent 1注册
- **负载分布**: 任务可以在不同节点上执行
- **资源监控**: 每个节点独立监控资源
- **Web UI**: 每个节点都有独立的管理界面
- **API接口**: 统一的RESTful API接口

## 💻 Web UI 界面

CNET Agent 提供了现代化的 Web 管理界面，包含以下功能：

### 🏠 仪表板 (Dashboard)
- **📊 实时资源监控**: CPU、内存、磁盘使用情况
- **📈 任务统计**: 运行中、已完成、失败的任务数量
- **🌐 集群概览**: 总节点数、活跃节点数、区域分布
- **📋 最近任务**: 最近创建的任务列表
- **🔄 自动刷新**: 实时更新数据

### 📋 任务管理 (Tasks)
- **➕ 创建任务**: 支持进程、容器、虚拟机（开发中）
- **📝 任务列表**: 查看所有任务及其状态
- **🔍 任务详情**: 查看任务配置和资源使用
- **📄 任务日志**: 实时查看任务输出和错误日志
- **⏹️ 任务控制**: 停止、重启、删除任务

### 📊 资源监控 (Resources)
- **💻 CPU监控**: 使用率、核心数、频率信息
- **🧠 内存监控**: 总内存、可用内存、使用率
- **💾 磁盘监控**: 磁盘使用情况、I/O统计
- **🌐 网络监控**: 网络接口、流量统计
- **📈 历史数据**: 资源使用趋势图

### 🌐 节点发现 (Nodes)
- **🔍 节点列表**: 显示所有注册的节点
- **📊 节点状态**: 活跃、离线、未知状态
- **🌍 区域信息**: 节点所属区域和数据中心
- **⏰ 最后活跃**: 节点最后活跃时间
- **🔄 自动发现**: 自动发现和注册新节点
- **🏷️ 层次化标识**: 支持层次化节点标识管理
- **🌳 层次结构**: 按层次组织节点，支持父子关系
- **🔍 标识解析**: 通过层次化标识快速定位节点

### 📝 日志查看 (Logs)
- **📄 Agent日志**: 系统运行日志
- **📋 任务日志**: 任务执行日志
- **🔄 实时更新**: 自动刷新日志内容
- **🔍 日志过滤**: 按级别和关键词过滤

## 🔌 API 接口

### 🏥 健康检查
```bash
curl http://localhost:8080/api/health
```

### 🏠 节点信息
```bash
curl http://localhost:8080/api/node
```

### 📊 资源状态
```bash
# 获取资源信息
curl http://localhost:8080/api/resources

# 获取资源使用情况
curl http://localhost:8080/api/resources/usage
```

### 📋 任务管理
```bash
# 列出所有任务
curl http://localhost:8080/api/tasks

# 创建任务
curl -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-task",
    "type": "process",
    "command": "echo",
    "args": ["Hello, CNET!"],
    "env": {"ENV_VAR": "value"}
  }'

# 获取任务详情
curl http://localhost:8080/api/tasks/{task-id}

# 停止任务
curl -X DELETE http://localhost:8080/api/tasks/{task-id}

# 获取任务日志
curl http://localhost:8080/api/tasks/{task-id}/logs?lines=100
```

### 🌐 节点发现
```bash
# 列出发现的节点
curl http://localhost:8080/api/discovery/nodes

# 注册节点
curl -X POST http://localhost:8080/api/discovery/register \
  -H "Content-Type: application/json" \
  -d '{
    "node": {
      "id": "node-1",
      "name": "worker-1",
      "address": "192.168.1.100",
      "port": 8080,
      "region": "us-west",
      "datacenter": "dc1"
    }
  }'
```

### 🏷️ 层次化标识管理
```bash
# 为节点分配层次化标识
curl -X POST http://localhost:8080/api/discovery/hierarchy/assign \
  -H "Content-Type: application/json" \
  -d '{
    "node_id": "node-1"
  }'

# 解析层次化标识
curl -X POST http://localhost:8080/api/discovery/hierarchy/resolve \
  -H "Content-Type: application/json" \
  -d '{
    "hierarchy_id": "34.23.1.8"
  }'

# 按层次结构列出节点
curl http://localhost:8080/api/discovery/hierarchy/nodes
```

## 🧪 演示和测试

### 快速演示
```bash
# 运行完整演示
./examples/demo.sh

# 测试Web UI
./examples/test_webui.sh

# 测试节点发现
./examples/test_discovery.sh

# 测试层次化标识功能
./examples/test_hierarchy.sh
```

### 多节点演示
```bash
# 启动两个Agent进行发现演示
./examples/start_two_agents.sh

# 测试两个Agent的通信
./examples/test_two_agents.sh
```

### 🌐 层次化集群部署

CNET Agent支持多级层次化集群部署，可以实现复杂的节点层次结构。

#### 层次化集群配置

项目提供了完整的层次化集群配置文件：

**根节点 (discovery-server) - `config.yaml`**:
```yaml
agent:
  address: "0.0.0.0"
  port: 8080
  node_id: "discovery-server"
  node_name: "Discovery Server"
  region: "default"
  datacenter: "dc1"

discovery:
  enabled: true
  servers: []  # 根节点，不向其他服务器注册
```

**Level 2节点 - `config_level2.yaml`**:
```yaml
agent:
  address: "0.0.0.0"
  port: 8082
  node_id: "level2-node"
  node_name: "Level 2 Node"
  region: "us-west"
  datacenter: "dc1"

discovery:
  enabled: true
  servers:
    - "localhost:8080"  # 向根节点注册
```

**Level 3节点 - `config_level3.yaml`**:
```yaml
agent:
  address: "0.0.0.0"
  port: 8083
  node_id: "level3-node"
  node_name: "Level 3 Node"
  region: "us-west"
  datacenter: "dc1"

discovery:
  enabled: true
  servers:
    - "localhost:8082"  # 向Level 2节点注册
```

**Level 4节点 - `config_level4_node1.yaml`**:
```yaml
agent:
  address: "0.0.0.0"
  port: 8084
  node_id: "level4-node1"
  node_name: "Level 4 Node 1"
  region: "us-west"
  datacenter: "dc1"

discovery:
  enabled: true
  servers:
    - "localhost:8083"  # 向Level 3节点注册
```

#### 启动层次化集群

```bash
# 1. 启动根节点 (discovery-server)
./bin/cnet-agent -config config.yaml > discovery-server.log 2>&1 &
sleep 3

# 2. 启动Level 2节点
./bin/cnet-agent -config config_level2.yaml > level2.log 2>&1 &
sleep 3

# 3. 启动Level 3节点
./bin/cnet-agent -config config_level3.yaml > level3.log 2>&1 &
sleep 3

# 4. 启动Level 4节点
./bin/cnet-agent -config config_level4_node1.yaml > level4_node1.log 2>&1 &
./bin/cnet-agent -config config_level4_node2.yaml > level4_node2.log 2>&1 &

# 查看所有节点状态
curl http://localhost:8080/api/health  # 根节点
curl http://localhost:8082/api/health  # Level 2
curl http://localhost:8083/api/health  # Level 3
curl http://localhost:8084/api/health  # Level 4节点1
curl http://localhost:8085/api/health  # Level 4节点2
```

#### 层次化集群验证

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

#### 访问地址

启动后可以访问以下地址：

- **根节点 (discovery-server)**: http://localhost:8080
- **Level 2节点**: http://localhost:8082
- **Level 3节点**: http://localhost:8083
- **Level 4节点1**: http://localhost:8084
- **Level 4节点2**: http://localhost:8085

#### 层次化集群功能验证

```bash
# 检查根节点发现的节点
curl http://localhost:8080/api/discovery/nodes | jq .

# 检查Level 2节点发现的节点
curl http://localhost:8082/api/discovery/nodes | jq .

# 检查Level 3节点发现的节点
curl http://localhost:8083/api/discovery/nodes | jq .

# 测试层次化标识解析
curl -X POST http://localhost:8080/api/discovery/hierarchy/resolve \
  -H "Content-Type: application/json" \
  -d '{"hierarchy_id": "34.23.1.1.1.1"}'

# 在不同层级的节点上创建任务
curl -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "root-task",
    "type": "process",
    "command": "echo",
    "args": ["Hello from Root Node"]
  }'

curl -X POST http://localhost:8083/api/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "level3-task",
    "type": "process",
    "command": "echo",
    "args": ["Hello from Level 3 Node"]
  }'
```

#### 停止层次化集群

```bash
# 停止所有Agent
pkill -f cnet-agent

# 或使用脚本停止
./examples/stop_agents.sh
```

## 🔧 技术特性

### 线程安全设计
- **🔒 读写锁分离**: 使用 `sync.RWMutex` 实现高效的并发访问
- **⚡ 无死锁设计**: 避免在已持有锁的情况下再次获取锁
- **🔄 原子操作**: 确保数据一致性和线程安全
- **📊 性能优化**: 最小化锁的持有时间，提高并发性能

### 层次化标识系统
- **🏷️ 自动分配**: 自动为节点分配唯一的层次化标识
- **🌳 无限层级**: 支持任意深度的层次结构
- **🔍 快速解析**: 高效的标识解析和查找算法
- **📈 动态扩展**: 支持动态添加和删除节点

### 输入验证和错误处理
- **✅ 完整验证**: 对所有输入进行严格验证
- **🛡️ 错误处理**: 详细的错误信息和日志记录
- **🔄 重试机制**: 自动重试失败的注册请求
- **📝 状态管理**: 完善的节点状态管理

### 性能优化
- **⚡ 高效算法**: 优化的字符串操作和数据结构
- **🔒 锁优化**: 减少锁竞争，提高并发性能
- **💾 内存管理**: 高效的内存使用和垃圾回收
- **📊 监控指标**: 实时性能监控和统计

## 📋 任务类型

### 1. 🖥️ 本地进程 (process) - ✅ 已实现
```json
{
  "name": "my-process",
  "type": "process",
  "command": "/usr/bin/python3",
  "args": ["script.py", "--arg", "value"],
  "env": {
    "PYTHONPATH": "/opt/myapp"
  },
  "working_dir": "/opt/myapp",
  "resources": {
    "cpu_limit": 1.0,
    "memory_limit": 512000000,
    "disk_limit": 1000000000
  }
}
```

**特性**:
- ✅ 支持环境变量设置
- ✅ 支持工作目录配置
- ✅ 支持资源限制
- ✅ 实时日志输出
- ✅ 进程生命周期管理

### 2. 🐳 容器 (container) - 🚧 开发中
```json
{
  "name": "my-container",
  "type": "container",
  "command": "docker",
  "args": ["run", "--rm", "-p", "8080:80", "nginx:alpine"],
  "resources": {
    "cpu_limit": 0.5,
    "memory_limit": 256000000
  }
}
```

**计划特性**:
- 🔄 Docker容器支持
- 🔄 容器镜像管理
- 🔄 网络配置
- 🔄 卷挂载

### 3. 🖥️ 虚拟机 (vm) - 📋 计划中
```json
{
  "name": "my-vm",
  "type": "vm",
  "command": "qemu-system-x86_64",
  "args": ["-m", "1024", "-hda", "disk.img"],
  "resources": {
    "cpu_limit": 2.0,
    "memory_limit": 1024000000
  }
}
```

**计划特性**:
- 📋 QEMU/KVM支持
- 📋 虚拟机镜像管理
- 📋 虚拟网络配置
- 📋 快照功能

## 🚀 部署选项

### 1. 📦 二进制部署
```bash
# 构建
make build

# 安装为系统服务
make install
make install-service

# 启动服务
sudo systemctl start cnet

# 查看状态
sudo systemctl status cnet
```

### 2. 🐳 Docker 部署
```bash
# 使用 Docker Compose
docker-compose up -d

# 或直接使用 Docker
docker run -d \
  --name cnet-agent \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  cnet-agent
```

### 3. ☸️ Kubernetes 部署
```bash
# 创建 ConfigMap
kubectl create configmap cnet-config --from-file=config.yaml

# 部署 Agent
kubectl apply -f k8s/
```

### 4. 🔧 开发环境
```bash
# 快速启动
./start.sh

# 运行演示
./examples/demo.sh

# 多节点测试
./examples/start_two_agents.sh
```

### 5. 🌐 集群部署
```bash
# 启动双Agent集群
./examples/start_two_agents.sh

# 停止集群
./examples/stop_agents.sh

# 清理日志
./examples/stop_agents.sh --clean
```

## 📊 监控和日志

### 📝 查看日志
```bash
# 系统服务日志
journalctl -u cnet -f

# Docker 日志
docker logs -f cnet-agent

# 任务日志
curl http://localhost:8080/api/tasks/{task-id}/logs

# 查看Agent日志文件
tail -f agent1.log
tail -f agent2.log
```

### 📈 监控指标
- **💻 CPU**: 使用率、核心数、频率
- **🧠 内存**: 使用情况、可用内存
- **💾 磁盘**: I/O统计、使用率
- **🌐 网络**: 流量统计、接口状态
- **📋 任务**: 执行状态、资源消耗
- **🌐 集群**: 节点状态、连接情况

### 🔍 日志位置
- **Agent日志**: 控制台输出 + 可选文件日志
- **任务日志**: `/tmp/cnet/logs/{task-id}/` 目录
- **系统日志**: 通过systemd或Docker管理

## 🛠️ 开发

### 📁 项目结构
```
cnet/
├── main.go                    # 主入口
├── internal/
│   ├── agent/                # Agent 核心
│   │   ├── api/             # HTTP API 服务器
│   │   ├── discovery/       # 节点发现服务
│   │   ├── resources/       # 资源监控服务
│   │   └── tasks/           # 任务管理服务
│   ├── config/              # 配置管理
│   └── logger/              # 日志管理
├── web/                      # Web UI
│   ├── templates/           # HTML 模板
│   └── static/              # 静态资源
├── examples/                # 示例和测试脚本
├── scripts/                 # 部署脚本
├── config.yaml             # 配置文件
├── Dockerfile              # Docker 构建
├── docker-compose.yml      # Docker Compose
├── Makefile               # 构建脚本
└── .gitignore             # Git 忽略文件
```

### 🔧 开发环境设置
```bash
# 克隆项目
git clone <repository-url>
cd cnet

# 安装依赖
make deps

# 运行测试
make test

# 构建
make build

# 运行
make run

# 运行演示
./examples/demo.sh
```

### 🧪 测试和演示
```bash
# 基础功能测试
./examples/test_agent.sh

# Web UI测试
./examples/test_webui.sh

# 节点发现测试
./examples/test_discovery.sh

# 多节点演示
./examples/start_two_agents.sh
```

### 🤝 贡献指南
1. Fork 项目
2. 创建功能分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request

## 📄 许可证

MIT License

## 🗺️ 路线图

### ✅ 已完成
- [x] 基础Agent架构
- [x] 本地进程任务执行
- [x] 资源监控 (CPU、内存、磁盘、网络)
- [x] 节点发现和注册
- [x] 层次化集群支持
- [x] 层次化标识分配和解析
- [x] 线程安全的discovery服务
- [x] 输入验证和错误处理
- [x] 性能优化和锁机制
- [x] RESTful API接口
- [x] Web UI界面
- [x] 集群节点显示
- [x] 任务日志管理
- [x] Docker支持
- [x] 配置管理

### 🚧 开发中
- [ ] 完整的容器支持 (Docker)
- [ ] 容器镜像管理
- [ ] 容器网络配置

### 📋 计划中
- [ ] 虚拟机支持 (QEMU/KVM)
- [ ] 高级调度算法
- [ ] 服务发现集成
- [ ] 指标导出 (Prometheus)
- [ ] 集群管理功能
- [ ] 高可用性支持
- [ ] 任务依赖管理
- [ ] 资源配额管理

## 📞 支持

如有问题或建议，请创建 Issue 或联系维护者。

## 🎉 项目状态

**CNET Agent** 是一个活跃开发中的项目，目前已经实现了核心功能：

- ✅ **工作负载管理**: 支持本地进程执行
- ✅ **资源监控**: 完整的系统资源监控
- ✅ **节点发现**: 分布式集群支持
- ✅ **层次化集群**: 支持多级层次化集群结构
- ✅ **线程安全**: 完全线程安全的discovery服务
- ✅ **性能优化**: 高效的锁机制和算法
- ✅ **输入验证**: 完整的输入验证和错误处理
- ✅ **Web UI**: 现代化管理界面
- ✅ **API接口**: 完整的RESTful API
- ✅ **Docker支持**: 容器化部署

### 🚀 最新更新

- **🌳 层次化集群**: 支持无限层级的节点层次结构
- **🔒 线程安全**: 完全线程安全的并发访问
- **⚡ 性能优化**: 优化的锁机制和算法
- **✅ 输入验证**: 完整的输入验证和错误处理
- **🔄 自动重注册**: 支持节点重复注册和状态更新

项目正在持续改进中，欢迎贡献代码和反馈！🚀
