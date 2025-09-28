# CNET Agent .gitignore 说明

## 📁 文件概述

这个.gitignore文件为CNET Agent项目提供了完整的版本控制忽略规则，确保敏感文件、构建产物、日志文件等不会被意外提交到Git仓库。

## 🔧 主要忽略类别

### 1. Go语言相关
```
# 二进制文件
*.exe, *.exe~, *.dll, *.so, *.dylib
bin/, dist/

# 测试文件
*.test, *.out

# 工作空间文件
go.work
```

### 2. IDE和编辑器文件
```
# VS Code
.vscode/

# IntelliJ IDEA
.idea/, *.iml, *.ipr, *.iws

# Vim
*.swp, *.swo, *~

# Sublime Text
*.sublime-project, *.sublime-workspace
```

### 3. 操作系统文件
```
# macOS
.DS_Store, .DS_Store?, ._*, .Spotlight-V100, .Trashes

# Windows
ehthumbs.db, Thumbs.db

# Linux
.fuse_hidden*, .directory, .Trash-*, .nfs*
```

### 4. CNET Agent特定文件

#### 配置文件
```
# 动态生成的配置文件
config_*.yaml
config_agent*.yaml
config_with_discovery.yaml
```

#### 日志文件
```
# 所有日志文件
*.log
agent1.log
agent2.log
agent*.log
```

#### 运行时文件
```
# 临时目录
/tmp/
/var/tmp/
/tmp/cnet/
/var/tmp/cnet/

# 进程文件
*.pid
*.lock
```

#### 测试文件
```
# 测试输出
test-results/
test-output/
coverage.out
coverage.html
```

### 5. 开发工具文件
```
# 环境变量
.env, .env.*, .env.local

# 缓存文件
.cache, .parcel-cache, .nyc_output

# 依赖目录
node_modules/, bower_components/, vendor/
```

### 6. 构建和部署文件
```
# Docker
docker-data/

# Kubernetes
k8s/generated/

# 归档文件
*.tar, *.tar.gz, *.zip, *.rar, *.7z
```

## 🎯 CNET Agent特殊考虑

### 为什么忽略这些文件？

1. **配置文件**：
   - `config_*.yaml`：动态生成的测试配置
   - 包含敏感信息或临时配置

2. **日志文件**：
   - `*.log`：运行时生成的日志
   - 可能包含敏感信息
   - 文件大小可能很大

3. **二进制文件**：
   - `bin/`：编译后的可执行文件
   - 可以通过源码重新构建

4. **临时文件**：
   - `/tmp/cnet/`：CNET Agent的日志目录
   - 运行时生成的临时数据

## 📋 使用建议

### 1. 保留的文件
- `config.yaml`：默认配置文件（不包含敏感信息）
- `README.md`：项目文档
- `Dockerfile`：容器构建文件
- `docker-compose.yml`：服务编排文件

### 2. 需要手动管理的文件
- 生产环境配置文件
- 敏感的环境变量文件
- 私钥和证书文件

### 3. 开发环境
- 使用 `config.yaml` 作为基础配置
- 测试时生成临时配置文件
- 日志文件自动忽略

## 🔍 验证.gitignore

### 检查忽略状态
```bash
# 查看被忽略的文件
git check-ignore bin/ agent1.log config_agent1.yaml

# 查看Git状态
git status --porcelain
```

### 强制添加被忽略的文件
```bash
# 如果需要添加被忽略的文件
git add -f filename
```

## 🚀 最佳实践

### 1. 配置文件管理
- 提供 `config.yaml.example` 作为模板
- 在README中说明配置方法
- 敏感配置使用环境变量

### 2. 日志管理
- 使用日志轮转
- 设置日志级别
- 避免提交日志文件

### 3. 构建管理
- 在CI/CD中构建二进制文件
- 使用Docker进行部署
- 提供构建脚本

## 📝 总结

这个.gitignore文件确保了：

✅ **安全性**：敏感文件不会被提交  
✅ **清洁性**：仓库只包含源码和必要文件  
✅ **效率性**：减少不必要的文件跟踪  
✅ **兼容性**：支持多种开发环境  
✅ **专业性**：符合Go项目最佳实践  

通过这个.gitignore文件，CNET Agent项目可以保持一个干净、安全、高效的Git仓库！🎉
