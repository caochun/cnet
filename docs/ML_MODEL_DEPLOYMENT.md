# CNET Agent 机器学习模型部署指南

## 🎯 概述

CNET Agent 现在支持在分布式节点上部署和运行机器学习模型。本指南将详细介绍如何部署、管理和使用机器学习模型。

## ✨ 功能特性

- **🤖 多种模型支持**: 支持线性回归、神经网络、深度学习等模型
- **📊 实时监控**: 实时监控模型训练和推理过程
- **🔄 动态部署**: 支持动态部署和停止模型
- **📈 资源管理**: 智能资源分配和限制
- **📝 日志记录**: 完整的训练和推理日志
- **🌐 API接口**: RESTful API管理模型
- **💻 Web UI**: 图形化管理界面

## 🏗️ 架构设计

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   CNET Agent    │    │   CNET Agent    │    │   CNET Agent    │
│   (Node 1)      │    │   (Node 2)      │    │   (Node 3)      │
├─────────────────┤    ├─────────────────┤    ├─────────────────┤
│ - ML Models     │    │ - ML Models     │    │ - ML Models     │
│ - Training      │    │ - Inference     │    │ - Training      │
│ - Monitoring    │    │ - Monitoring   │    │ - Monitoring    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 📋 支持的模型类型

### 1. 线性回归模型
- **用途**: 简单预测任务
- **特点**: 快速训练，资源占用少
- **适用场景**: 数值预测、趋势分析

### 2. 神经网络模型
- **用途**: 复杂非线性预测
- **特点**: 高精度，需要更多资源
- **适用场景**: 图像识别、自然语言处理

### 3. 自定义模型
- **用途**: 用户自定义的机器学习模型
- **特点**: 灵活配置，支持各种框架
- **适用场景**: 特定业务需求

## 🚀 快速开始

### 1. 环境准备

确保系统已安装Python和必要的机器学习库：

```bash
# 安装Python依赖
pip install -r examples/ml_models/requirements.txt
```

### 2. 启动CNET Agent

```bash
# 启动CNET Agent
./bin/cnet-agent -config config.yaml
```

### 3. 部署模型

#### 部署线性回归模型

```bash
curl -X POST "http://localhost:8080/api/ml/models" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "linear-regression-model",
    "model_type": "linear_regression",
    "model_path": "models/linear_regression_model.joblib",
    "script_path": "examples/ml_models/simple_linear_regression.py",
    "command": "python3",
    "args": ["examples/ml_models/simple_linear_regression.py", "train", "models/linear_regression_model.joblib", "1000"],
    "working_dir": ".",
    "env": {
      "PYTHONPATH": "examples/ml_models"
    },
    "resources": {
      "cpu_limit": 1.0,
      "memory_limit": 512000000,
      "disk_limit": 1000000000
    }
  }'
```

#### 部署神经网络模型

```bash
curl -X POST "http://localhost:8080/api/ml/models" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "neural-network-model",
    "model_type": "neural_network",
    "model_path": "models/neural_network_model.h5",
    "script_path": "examples/ml_models/neural_network.py",
    "command": "python3",
    "args": ["examples/ml_models/neural_network.py", "train", "models/neural_network_model.h5", "1000", "50"],
    "working_dir": ".",
    "env": {
      "PYTHONPATH": "examples/ml_models"
    },
    "resources": {
      "cpu_limit": 2.0,
      "memory_limit": 1024000000,
      "disk_limit": 2000000000
    }
  }'
```

## 📚 API 参考

### 机器学习模型管理

#### 1. 列出所有ML模型
```http
GET /api/ml/models
```

**响应示例:**
```json
[
  {
    "id": "task-uuid-1",
    "name": "linear-regression-model",
    "type": "ml_model",
    "status": "running",
    "command": "python3",
    "args": ["examples/ml_models/simple_linear_regression.py", "train", "models/linear_regression_model.joblib", "1000"],
    "env": {
      "PYTHONPATH": "examples/ml_models",
      "MODEL_PATH": "models/linear_regression_model.joblib",
      "MODEL_TYPE": "linear_regression"
    },
    "working_dir": ".",
    "resources": {
      "cpu_limit": 1.0,
      "memory_limit": 512000000,
      "disk_limit": 1000000000
    },
    "created_at": "2024-01-01T00:00:00Z",
    "started_at": "2024-01-01T00:00:01Z"
  }
]
```

#### 2. 部署ML模型
```http
POST /api/ml/models
```

**请求体:**
```json
{
  "name": "my-model",
  "model_type": "linear_regression",
  "model_path": "models/my_model.joblib",
  "script_path": "examples/ml_models/simple_linear_regression.py",
  "command": "python3",
  "args": ["script.py", "train", "model.joblib", "1000"],
  "env": {
    "PYTHONPATH": "examples/ml_models"
  },
  "working_dir": ".",
  "resources": {
    "cpu_limit": 1.0,
    "memory_limit": 512000000,
    "disk_limit": 1000000000
  }
}
```

#### 3. 获取ML模型信息
```http
GET /api/ml/models/{id}
```

#### 4. 停止ML模型
```http
DELETE /api/ml/models/{id}
```

#### 5. 使用ML模型进行预测
```http
POST /api/ml/models/{id}/predict
```

**请求体:**
```json
{
  "input_data": 5.5
}
```

**响应示例:**
```json
{
  "prediction_task_id": "prediction-task-uuid",
  "message": "Prediction task created"
}
```

#### 6. 获取ML模型日志
```http
GET /api/ml/models/{id}/logs?lines=100
```

## 🎨 Web UI 管理

访问 `http://localhost:8080` 打开Web管理界面：

### 功能特性
- **📊 模型概览**: 查看所有部署的模型
- **🚀 模型部署**: 图形化部署新模型
- **📈 实时监控**: 监控模型训练和推理状态
- **📝 日志查看**: 实时查看模型日志
- **⚙️ 资源管理**: 查看资源使用情况

## 📝 模型脚本开发

### 标准接口

所有机器学习模型脚本必须支持以下命令：

1. **训练模型**: `python script.py train <model_path> [args...]`
2. **模型预测**: `python script.py predict <model_path> <input_data>`
3. **模型评估**: `python script.py evaluate <model_path>`

### 输出格式

所有命令的输出必须是JSON格式：

```json
{
  "status": "success",
  "model_path": "models/model.joblib",
  "prediction": 5.5,
  "model_info": {
    "coefficient": 2.0,
    "intercept": 3.0
  }
}
```

### 示例脚本

参考 `examples/ml_models/` 目录下的示例脚本：
- `simple_linear_regression.py` - 线性回归模型
- `neural_network.py` - 神经网络模型

## 🔧 配置选项

### 资源限制

```json
{
  "resources": {
    "cpu_limit": 1.0,        // CPU核心数限制
    "memory_limit": 512000000, // 内存限制（字节）
    "disk_limit": 1000000000   // 磁盘限制（字节）
  }
}
```

### 环境变量

```json
{
  "env": {
    "PYTHONPATH": "examples/ml_models",
    "MODEL_PATH": "models/model.joblib",
    "MODEL_TYPE": "linear_regression"
  }
}
```

## 🚀 部署示例

### 使用演示脚本

```bash
# 运行完整的部署演示
chmod +x examples/ml_deployment_demo.sh
./examples/ml_deployment_demo.sh
```

### 手动部署步骤

1. **启动CNET Agent**
   ```bash
   ./bin/cnet-agent -config config.yaml
   ```

2. **部署模型**
   ```bash
   curl -X POST "http://localhost:8080/api/ml/models" \
     -H "Content-Type: application/json" \
     -d @model_deployment.json
   ```

3. **监控模型状态**
   ```bash
   curl -s "http://localhost:8080/api/ml/models" | jq '.'
   ```

4. **进行预测**
   ```bash
   curl -X POST "http://localhost:8080/api/ml/models/$MODEL_ID/predict" \
     -H "Content-Type: application/json" \
     -d '{"input_data": 5.5}'
   ```

## 🔍 故障排除

### 常见问题

1. **模型部署失败**
   - 检查Python环境和依赖
   - 验证脚本路径和权限
   - 查看任务日志

2. **预测结果异常**
   - 检查输入数据格式
   - 验证模型文件完整性
   - 查看预测任务日志

3. **资源不足**
   - 调整资源限制
   - 检查节点资源使用情况
   - 优化模型参数

### 日志查看

```bash
# 查看模型日志
curl -s "http://localhost:8080/api/ml/models/$MODEL_ID/logs"

# 查看所有任务日志
curl -s "http://localhost:8080/api/tasks" | jq '.[].log_file'
```

## 📈 性能优化

### 资源优化
- 合理设置CPU和内存限制
- 使用SSD存储提高I/O性能
- 启用GPU加速（如果可用）

### 模型优化
- 使用模型压缩技术
- 实现批量预测
- 缓存常用预测结果

## 🔮 未来规划

- [ ] GPU支持
- [ ] 模型版本管理
- [ ] 自动扩缩容
- [ ] 模型性能监控
- [ ] 分布式训练支持
- [ ] 模型市场

## 📞 支持

如有问题或建议，请：
1. 查看日志文件
2. 检查API响应
3. 参考示例代码
4. 提交Issue

---

**CNET Agent 机器学习模型部署功能让您轻松在分布式环境中部署和管理机器学习模型！** 🚀
