# Scripts 目录

本目录包含了CNET Agent项目的脚本文件。

## 📜 脚本列表

### 系统服务脚本
- **[cnet.service](./cnet.service)** - systemd服务配置文件，用于将CNET Agent安装为系统服务


## 🚀 使用方法

### 安装为系统服务
```bash
# 复制服务文件到系统目录
sudo cp scripts/cnet.service /etc/systemd/system/

# 重新加载systemd配置
sudo systemctl daemon-reload

# 启用并启动服务
sudo systemctl enable cnet
sudo systemctl start cnet

# 查看服务状态
sudo systemctl status cnet
```


## 📝 注意事项

- 确保脚本有执行权限：`chmod +x scripts/*.sh`
- 系统服务脚本需要root权限安装

## 🔧 开发说明

- 服务脚本用于生产环境部署
- 所有脚本都包含错误处理和日志输出
