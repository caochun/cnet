# CNET Agent 机器学习模型作为进程任务部署指南

## 🎯 设计理念

CNET Agent 采用统一的任务负载部署架构，支持三种执行方式：
- **进程（process）** - 本地进程执行
- **容器（container）** - Docker容器执行  
- **虚拟机（vm）** - 虚拟机执行

机器学习模型应该作为这三种执行方式中的一种来部署，而不是作为独立的类型。本指南展示如何将机器学习模型作为**进程任务**来部署。

## ✅ 正确的实现方式

### 1. 作为进程任务部署

机器学习模型通过标准的任务API部署，类型为`process`：

```bash
curl -X POST "http://localhost:8080/api/tasks" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "linear-regression-model",
    "type": "process",
    "command": "python3",
    "args": ["examples/ml_models/simple_linear_regression.py", "train", "models/model.joblib", "1000"],
    "env": {
      "PYTHONPATH": "examples/ml_models",
      "MODEL_PATH": "models/model.joblib",
      "MODEL_TYPE": "linear_regression"
    },
    "working_dir": ".",
    "resources": {
      "cpu_limit": 1.0,
      "memory_limit": 512000000,
      "disk_limit": 1000000000
    }
  }'
```

### 2. 使用Web UI预设

在Web UI的任务创建表单中，选择"Task Preset"：
- **ML: Linear Regression** - 线性回归模型预设
- **ML: Neural Network** - 神经网络模型预设  
- **ML: Custom Model** - 自定义模型预设

### 3. 模型预测

使用相同的进程任务方式创建预测任务：

```bash
curl -X POST "http://localhost:8080/api/tasks" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "model-prediction",
    "type": "process",
    "command": "python3",
    "args": ["examples/ml_models/simple_linear_regression.py", "predict", "models/model.joblib", "5.5"],
    "env": {
      "PYTHONPATH": "examples/ml_models",
      "MODEL_PATH": "models/model.joblib",
      "MODEL_TYPE": "linear_regression"
    },
    "working_dir": ".",
    "resources": {
      "cpu_limit": 0.5,
      "memory_limit": 256000000,
      "disk_limit": 500000000
    }
  }'
```

## 🏗️ 架构优势

### 1. 统一管理
- 所有任务（包括ML模型）使用相同的API接口
- 统一的监控、日志和资源管理
- 一致的Web UI界面

### 2. 灵活部署
- 支持进程、容器、虚拟机三种执行方式
- 可以根据需求选择最适合的执行环境
- 支持资源限制和隔离

### 3. 扩展性
- 易于添加新的执行器类型
- 支持复杂的ML工作流
- 可以与其他任务类型混合部署

## 📋 支持的模型类型

### 1. 线性回归模型
```bash
# 训练
python3 examples/ml_models/simple_linear_regression.py train models/model.joblib 1000

# 预测  
python3 examples/ml_models/simple_linear_regression.py predict models/model.joblib 5.5

# 评估
python3 examples/ml_models/simple_linear_regression.py evaluate models/model.joblib
```

### 2. 神经网络模型
```bash
# 训练
python3 examples/ml_models/neural_network.py train models/model.h5 1000 50

# 预测
python3 examples/ml_models/neural_network.py predict models/model.h5 "3.5,2.1"

# 评估
python3 examples/ml_models/neural_network.py evaluate models/model.h5
```

### 3. 自定义模型
```bash
# 训练
python3 your_script.py train models/your_model.pkl 1000

# 预测
python3 your_script.py predict models/your_model.pkl input_data

# 评估
python3 your_script.py evaluate models/your_model.pkl
```

## 🚀 使用示例

### 1. 快速开始

```bash
# 运行演示脚本
./examples/ml_as_process_demo.sh
```

### 2. Web UI操作

1. 访问 `http://localhost:8080`
2. 点击"Tasks"页面
3. 点击"Create Task"按钮
4. 选择"Task Preset"为"ML: Linear Regression"
5. 点击"Create Task"部署模型

### 3. API操作

```bash
# 列出所有任务
curl -s "http://localhost:8080/api/tasks" | jq '.'

# 获取特定任务
curl -s "http://localhost:8080/api/tasks/{task_id}" | jq '.'

# 查看任务日志
curl -s "http://localhost:8080/api/tasks/{task_id}/logs" | jq -r '.[]'

# 停止任务
curl -X DELETE "http://localhost:8080/api/tasks/{task_id}"
```

## 🔧 配置选项

### 资源限制
```json
{
  "resources": {
    "cpu_limit": 1.0,        // CPU核心数
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

## 📊 监控和管理

### 1. 任务状态
- **pending** - 等待执行
- **running** - 正在运行
- **completed** - 执行完成
- **failed** - 执行失败
- **stopped** - 被停止

### 2. 资源监控
- 实时CPU、内存、磁盘使用情况
- 网络I/O统计
- 任务执行时间

### 3. 日志管理
- 完整的训练和推理日志
- 错误信息和调试信息
- 实时日志查看

## 🎯 最佳实践

### 1. 模型训练
- 使用适当的资源限制
- 设置合理的超时时间
- 监控训练进度和性能

### 2. 模型预测
- 使用较小的资源限制
- 设置快速超时
- 批量处理预测请求

### 3. 资源管理
- 根据模型复杂度分配资源
- 监控资源使用情况
- 避免资源竞争

## 📝 总结

CNET Agent的机器学习模型部署功能遵循统一的任务负载架构：

✅ **统一管理**: 所有任务使用相同的API和界面  
✅ **灵活部署**: 支持进程、容器、虚拟机三种方式  
✅ **资源控制**: 精确的资源限制和监控  
✅ **易于扩展**: 支持复杂的ML工作流  
✅ **标准化**: 遵循CNET Agent的设计理念  

这种设计确保了机器学习模型与CNET Agent的整体架构完美集成，提供了统一、灵活、可扩展的ML模型部署解决方案。

---

**CNET Agent 让机器学习模型部署变得简单而统一！** 🚀
