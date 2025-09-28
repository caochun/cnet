# 配置文件目录

本目录包含了CNET Agent项目的各种配置文件。

## 📁 配置文件列表

### 主配置文件
- **[config.yaml](./config.yaml)** - 主配置文件（根节点discovery-server）

### 层次化集群配置
- **[config_level2.yaml](./config_level2.yaml)** - Level 2节点配置
- **[config_level3.yaml](./config_level3.yaml)** - Level 3节点配置
- **[config_level4_node1.yaml](./config_level4_node1.yaml)** - Level 4节点1配置
- **[config_level4_node2.yaml](./config_level4_node2.yaml)** - Level 4节点2配置

### 双Agent配置
- **[config_agent1.yaml](./config_agent1.yaml)** - Agent 1配置（发现服务器）
- **[config_agent2.yaml](./config_agent2.yaml)** - Agent 2配置（工作节点）

## 🚀 使用方法

### 启动根节点
```bash
./bin/cnet-agent -config configs/config.yaml
```

### 启动层次化集群
```bash
# 启动根节点
./bin/cnet-agent -config configs/config.yaml > discovery-server.log 2>&1 &
sleep 3

# 启动Level 2节点
./bin/cnet-agent -config configs/config_level2.yaml > level2.log 2>&1 &
sleep 3

# 启动Level 3节点
./bin/cnet-agent -config configs/config_level3.yaml > level3.log 2>&1 &
sleep 3

# 启动Level 4节点
./bin/cnet-agent -config configs/config_level4_node1.yaml > level4_node1.log 2>&1 &
./bin/cnet-agent -config configs/config_level4_node2.yaml > level4_node2.log 2>&1 &
```

### 启动双Agent集群
```bash
# 启动Agent 1 (发现服务器)
./bin/cnet-agent -config configs/config_agent1.yaml > agent1.log 2>&1 &
sleep 5

# 启动Agent 2 (工作节点)
./bin/cnet-agent -config configs/config_agent2.yaml > agent2.log 2>&1 &
```

## 📝 配置说明

### 层次化集群配置
- **根节点**: `config.yaml` - 作为discovery-server，不向其他服务器注册
- **Level 2**: `config_level2.yaml` - 向根节点注册
- **Level 3**: `config_level3.yaml` - 向Level 2节点注册
- **Level 4**: `config_level4_node*.yaml` - 向Level 3节点注册

### 双Agent配置
- **Agent 1**: `config_agent1.yaml` - 作为发现服务器
- **Agent 2**: `config_agent2.yaml` - 向Agent 1注册

## 🔧 自定义配置

可以根据需要修改这些配置文件：
- 修改端口号避免冲突
- 调整节点名称和ID
- 配置不同的区域和数据中心
- 设置心跳间隔和超时时间

## 📋 注意事项

- 确保端口不冲突
- 检查discovery.servers配置是否正确
- 层次化集群需要按顺序启动节点
- 双Agent集群需要先启动发现服务器
