# CNET Agent 层次化标识功能

## 🎯 功能概述

CNET Agent 现在支持层次化节点标识系统，允许上级节点为下级节点分配层次化的标识，并提供标识解析服务。这个功能类似于网络中的层次化地址分配，如 IP 地址的子网划分。

## 🏗️ 核心特性

### 1. 层次化标识分配
- **自动分配**: 上级节点自动为注册的下级节点分配层次化标识
- **格式**: 使用点号分隔的层次结构，如 `34.23.1.8`
- **继承**: 子节点标识继承父节点的前缀
- **唯一性**: 每个层次化标识在集群中唯一

### 2. 标识解析服务
- **快速定位**: 通过层次化标识快速定位节点
- **API地址**: 解析返回节点的完整API地址
- **状态信息**: 包含节点状态和最后活跃时间

### 3. 层次结构管理
- **父子关系**: 支持节点间的父子关系管理
- **层次级别**: 自动计算和显示节点的层次级别
- **组织结构**: 按层次组织显示所有节点

## 🔧 技术实现

### 数据结构扩展

```go
type Node struct {
    ID           string            `json:"id"`           // 原始节点ID
    HierarchyID  string            `json:"hierarchy_id"` // 层次化标识
    Name         string            `json:"name"`
    Address      string            `json:"address"`
    Port         int               `json:"port"`
    ParentID     string            `json:"parent_id"`     // 父节点ID
    Level        int               `json:"level"`         // 层次级别
    // ... 其他字段
}
```

### 核心算法

1. **标识分配算法**:
   ```go
   func (s *Service) assignHierarchyID(nodeID string) (string, int, string) {
       currentHierarchyID := s.getCurrentNodeHierarchyID()
       nextID := s.getNextChildID(currentHierarchyID)
       level := s.calculateLevel(nextID)
       return nextID, level, currentNodeID
   }
   ```

2. **子节点计数**:
   ```go
   func (s *Service) countChildNodes(parentHierarchyID string) int {
       count := 0
       for _, node := range s.nodes {
           if node.ParentID == s.config.Agent.NodeID && 
              len(node.HierarchyID) > len(parentHierarchyID) &&
              node.HierarchyID[:len(parentHierarchyID)] == parentHierarchyID {
               count++
           }
       }
       return count
   }
   ```

## 🌐 API 接口

### 1. 分配层次化标识
```bash
POST /api/discovery/hierarchy/assign
Content-Type: application/json

{
  "node_id": "worker-node-1"
}
```

**响应**:
```json
{
  "hierarchy_id": "34.23.1.8",
  "level": 3,
  "parent_id": "discovery-server"
}
```

### 2. 解析层次化标识
```bash
POST /api/discovery/hierarchy/resolve
Content-Type: application/json

{
  "hierarchy_id": "34.23.1.8"
}
```

**响应**:
```json
{
  "node_id": "worker-node-1",
  "address": "192.168.1.100",
  "port": 8081,
  "status": "active",
  "last_seen": "2024-01-15T10:30:00Z"
}
```

### 3. 按层次列出节点
```bash
GET /api/discovery/hierarchy/nodes
```

**响应**:
```json
{
  "discovery-server": [
    {
      "id": "worker-node-1",
      "hierarchy_id": "34.23.1.8",
      "name": "Worker Node 1",
      "level": 3,
      "parent_id": "discovery-server"
    }
  ]
}
```

## 💻 Web UI 功能

### 1. 节点列表增强
- 显示层次化标识列
- 显示层次级别
- 支持层次化标识的搜索和过滤

### 2. 层次化标识管理
- **分配标识**: 为节点分配层次化标识
- **解析标识**: 通过层次化标识查找节点
- **层次视图**: 按层次结构显示节点

### 3. 可视化改进
- 层次化标识高亮显示
- 层次级别徽章
- 父子关系可视化

## 🧪 测试和演示

### 1. 功能测试
```bash
# 运行层次化标识功能测试
./examples/test_hierarchy.sh
```

### 2. 完整演示
```bash
# 运行层次化标识完整演示
./examples/hierarchy_demo.sh
```

### 3. Web UI 演示
```bash
# 启动 Agent
./start.sh

# 访问 Web UI
open http://localhost:8080
```

## 📊 使用场景

### 1. 多级集群管理
- **数据中心级别**: 34.23.1 (数据中心1)
- **机架级别**: 34.23.1.8 (机架8)
- **服务器级别**: 34.23.1.8.15 (服务器15)

### 2. 地理分布管理
- **区域级别**: 34.23 (美国西部)
- **可用区级别**: 34.23.1 (可用区1)
- **节点级别**: 34.23.1.8 (节点8)

### 3. 服务发现
- **服务类型**: 34.23.1 (Web服务)
- **实例级别**: 34.23.1.8 (实例8)
- **版本级别**: 34.23.1.8.2 (版本2)

## 🔮 未来扩展

### 1. 高级功能
- **动态重新分配**: 支持层次化标识的动态重新分配
- **标识继承**: 支持标识的继承和传递
- **负载均衡**: 基于层次化标识的负载均衡

### 2. 性能优化
- **缓存机制**: 层次化标识解析的缓存
- **批量操作**: 支持批量分配和解析
- **异步处理**: 异步的标识分配和更新

### 3. 监控和告警
- **层次监控**: 按层次监控节点状态
- **告警规则**: 基于层次化标识的告警
- **指标收集**: 层次化指标收集和分析

## 📝 配置示例

### 1. 根节点配置
```yaml
agent:
  node_id: "discovery-server"
  node_name: "Discovery Server"
  # 根节点不需要层次化标识

discovery:
  enabled: true
  servers: []  # 作为根节点
```

### 2. 子节点配置
```yaml
agent:
  node_id: "worker-node-1"
  node_name: "Worker Node 1"

discovery:
  enabled: true
  servers:
    - "localhost:8080"  # 向根节点注册
```

## 🎉 总结

层次化标识功能为 CNET Agent 提供了强大的节点管理能力，支持：

- ✅ **层次化标识分配**: 自动为节点分配层次化标识
- ✅ **标识解析服务**: 快速定位和访问节点
- ✅ **层次结构管理**: 按层次组织和管理节点
- ✅ **Web UI 支持**: 完整的用户界面支持
- ✅ **API 接口**: 完整的 RESTful API
- ✅ **测试和演示**: 完整的测试和演示脚本

这个功能使得 CNET Agent 能够更好地支持大规模分布式集群的管理和运维。
