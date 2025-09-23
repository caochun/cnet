# CNET Agent ProcessExecutor 日志文件位置说明

## 📁 日志文件位置

ProcessExecutor 执行进程时，日志文件存储在以下位置：

### 基本路径
```
{系统临时目录}/cnet/logs/{task-id}.log
```

### 具体位置
- **macOS/Linux**: `/tmp/cnet/logs/` 或 `$TMPDIR/cnet/logs/`
- **Windows**: `%TEMP%\cnet\logs\`

### 实际示例
在 macOS 系统上，日志文件通常位于：
```
/var/folders/{随机字符串}/T/cnet/logs/{task-id}.log
```

## 🔧 日志文件创建过程

### 1. 任务创建时
```go
// 在 CreateTask 方法中
task.LogFile = filepath.Join(os.TempDir(), "cnet", "logs", taskID+".log")
```

### 2. 进程执行时
```go
// 在 ProcessExecutor.Execute 方法中
logFile, err := os.Create(task.LogFile)
if err != nil {
    return fmt.Errorf("failed to create log file: %w", err)
}
defer logFile.Close()

// 将进程输出重定向到日志文件
cmd.Stdout = logFile
cmd.Stderr = logFile
```

## 📝 日志文件内容

### 包含的内容
- **标准输出 (stdout)**: 进程的正常输出
- **标准错误 (stderr)**: 进程的错误输出
- **所有输出**: stdout 和 stderr 都重定向到同一个文件

### 文件格式
- **文件名**: `{task-id}.log`
- **编码**: UTF-8
- **权限**: 644 (rw-r--r--)

## 🌐 查看日志的方法

### 1. 通过 Web UI
1. 访问 http://localhost:8080
2. 进入 "Tasks" 页面
3. 点击任务ID查看详情
4. 在任务详情中查看日志

### 2. 通过 API
```bash
# 获取任务日志
curl http://localhost:8080/api/tasks/{task-id}/logs

# 获取指定行数的日志
curl "http://localhost:8080/api/tasks/{task-id}/logs?lines=100"
```

### 3. 直接访问文件
```bash
# 查看日志文件
cat /tmp/cnet/logs/{task-id}.log

# 实时查看日志
tail -f /tmp/cnet/logs/{task-id}.log
```

## 🔍 日志文件管理

### 自动创建
- 日志目录在 Agent 启动时自动创建
- 日志文件在任务创建时自动生成
- 文件路径存储在任务的 `LogFile` 字段中

### 文件保留
- 日志文件会一直保留，直到手动删除
- 不会自动清理，需要定期维护
- 可以通过配置设置清理策略

### 权限管理
- 日志文件对所有用户可读
- 只有 Agent 进程可以写入
- 建议定期清理旧日志文件

## 💡 最佳实践

### 1. 日志轮转
建议实现日志轮转机制：
```bash
# 使用 logrotate 管理日志
/var/log/cnet/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
}
```

### 2. 日志清理
定期清理旧日志文件：
```bash
# 清理7天前的日志
find /tmp/cnet/logs -name "*.log" -mtime +7 -delete
```

### 3. 监控日志大小
监控日志文件大小，避免磁盘空间不足：
```bash
# 检查日志目录大小
du -sh /tmp/cnet/logs
```

## 🚨 注意事项

1. **磁盘空间**: 日志文件会持续增长，需要监控磁盘使用
2. **文件权限**: 确保 Agent 有权限创建和写入日志文件
3. **并发访问**: 多个任务可能同时写入日志，注意文件锁定
4. **清理策略**: 建议设置自动清理策略，避免日志文件过多

## 📊 示例

### 查看当前日志文件
```bash
# 列出所有日志文件
ls -la /tmp/cnet/logs/

# 查看最新日志
ls -lt /tmp/cnet/logs/ | head -5

# 统计日志文件数量
find /tmp/cnet/logs -name "*.log" | wc -l
```

### 通过 API 查看日志
```bash
# 获取任务列表
curl http://localhost:8080/api/tasks

# 获取特定任务的日志
curl http://localhost:8080/api/tasks/94ed6336-19da-497a-b3a3-506f516c05b5/logs
```

这样，你就可以清楚地知道 ProcessExecutor 执行进程时日志文件的具体位置和如何访问它们了！
