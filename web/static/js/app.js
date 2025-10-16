function dashboard() {
    return {
        // 数据状态
        agentStatus: 'unknown',
        agentInfo: {},
        parentNode: null,
        peerNodes: [],
        localResources: { cpu: 0, memory: 0, gpu: 0, storage: 0 },
        workloads: [],
        lastUpdate: '',
            // DataGateway 环境
            gatewayEndpoint: '',
            gatewayBucket: 'cnet',
        
        // UI状态
        showSubmitForm: false,
        showWorkloadModal: false,
        selectedWorkload: null,
        
        // 动态表单数据
        formData: {
            name: '',
            type: 'data',
            config: {
                // MLModel
                model_type: 'yolo',
                model_path: 'models/yolo11n.onnx',
                service_port: 9001,
                // Process
                command: '',
                args_str: '',
                // Container
                image: '',
                // OpenCV
                cascade_type: 'face',
                // Data
                upload_method: 'file',
                data_type: 'file',
                access_mode: 'readonly',
                tags: '',
                download_url: '',
                file_name: '',
                file_path: ''
            },
            requirements: {
                cpu: 0.1,
                memory: 128,
                gpu: 0,
                storage: 0
            }
        },
        
        // 文件选择相关
        selectedFile: null,
        selectedDirectory: [],
        
        // 类型默认配置
        typeDefaults: {
            mlmodel: {
                config: { model_type: 'yolo', model_path: 'models/yolo11n.onnx', service_port: 9001 },
                requirements: { cpu: 2.0, memory: 2048, gpu: 0, storage: 0 }
            },
            process: {
                config: { command: 'sleep', args_str: '60' },
                requirements: { cpu: 1.0, memory: 512, gpu: 0, storage: 0 }
            },
            opencv: {
                config: { cascade_type: 'face', service_port: 9000 },
                requirements: { cpu: 1.0, memory: 512, gpu: 0, storage: 0 }
            },
            container: {
                config: { image: 'nginx:alpine' },
                requirements: { cpu: 1.0, memory: 512, gpu: 0, storage: 0 }
            },
            data: {
                config: { 
                    upload_method: 'file', 
                    data_type: 'file', 
                    access_mode: 'readonly',
                    tags: ''
                },
                requirements: { cpu: 0.1, memory: 128, gpu: 0, storage: 0 }
            },
            datagateway: {
                config: {
                    service_port: 9091,
                    service_host: '127.0.0.1',
                    base_path: '/tmp/cnet_data',
                    bucket: 'cnet',
                    auth_token: ''
                },
                requirements: { cpu: 0.2, memory: 128, gpu: 0, storage: 0 }
            }
        },
        
        // 计算属性
        get runningWorkloads() {
            return this.workloads.filter(w => w.status === 'running').length;
        },
        
        get completedWorkloads() {
            return this.workloads.filter(w => w.status === 'completed').length;
        },
        
        get failedWorkloads() {
            return this.workloads.filter(w => w.status === 'failed').length;
        },
        
        // 初始化
        init() {
            this.refreshData();
            // 每30秒自动刷新
            setInterval(() => {
                this.refreshData();
            }, 30000);
        },
        
        // 刷新所有数据
        async refreshData() {
            try {
                await Promise.all([
                    this.fetchAgentHealth(),
                    this.fetchLocalResources(),
                    this.fetchWorkloads(),
                    this.fetchNodes()
                ]);
                this.lastUpdate = new Date().toLocaleString('zh-CN');
            } catch (error) {
                console.error('刷新数据失败:', error);
                this.agentStatus = 'error';
            }
        },
        
        // 获取Agent健康状态
        async fetchAgentHealth() {
            try {
                const response = await fetch('/api/health');
                const data = await response.json();
                this.agentStatus = data.status === 'healthy' ? 'healthy' : 'unhealthy';
            } catch (error) {
                console.error('获取健康状态失败:', error);
                this.agentStatus = 'error';
            }
        },
        
        // 获取本地资源信息
        async fetchLocalResources() {
            try {
                const response = await fetch('/api/resources');
                const data = await response.json();
                if (data.resources) {
                    this.localResources = {
                        cpu: data.resources.available.cpu || 0,
                        memory: data.resources.available.memory || 0,
                        gpu: data.resources.available.gpu || 0,
                        storage: data.resources.available.storage || 0
                    };
                    // 更新agent信息
                    this.agentInfo = {
                        node_id: data.resources.node_id || '-',
                        address: window.location.host
                    };
                }
            } catch (error) {
                console.error('获取资源信息失败:', error);
            }
        },
        
        // 获取工作负载列表
        async fetchWorkloads() {
            try {
                const response = await fetch('/api/workloads');
                const data = await response.json();
                this.workloads = data.workloads || [];
                // 记录首个运行中的 DataGateway 入口与 bucket
                const gw = this.workloads.find(w => w.type === 'datagateway' && w.status === 'running');
                if (gw && gw.endpoint) {
                    this.gatewayEndpoint = gw.endpoint;
                    if (gw.config && gw.config.bucket) {
                        this.gatewayBucket = gw.config.bucket;
                    }
                }
            } catch (error) {
                console.error('获取工作负载失败:', error);
            }
        },
        
        // 获取节点信息
        async fetchNodes() {
            try {
                const response = await fetch('/api/nodes');
                const data = await response.json();
                // 这里需要根据实际API结构调整
                this.parentNode = data.parent || null;
                this.peerNodes = data.peers || [];
            } catch (error) {
                console.error('获取节点信息失败:', error);
            }
        },
        
        // 类型改变时更新默认值
        onTypeChange() {
            const defaults = this.typeDefaults[this.formData.type];
            if (defaults) {
                this.formData.config = { ...this.formData.config, ...defaults.config };
                this.formData.requirements = { ...defaults.requirements };
            }
        },
        
        // 文件选择处理
        handleFileSelect(event) {
            const file = event.target.files[0];
            if (file) {
                this.selectedFile = file;
                this.formData.config.file_name = file.name;
                // 自动计算存储需求
                this.formData.requirements.storage = file.size;
            }
        },

        // 目录选择处理
        handleDirectorySelect(event) {
            const files = Array.from(event.target.files);
            if (files.length > 0) {
                this.selectedDirectory = files;
                this.formData.config.file_count = files.length;
                this.formData.config.total_size = this.getTotalSize(files);
                
                // 自动计算存储需求
                this.formData.requirements.storage = this.getTotalSize(files);
            }
        },

        // 计算总大小
        getTotalSize(files) {
            return files.reduce((total, file) => total + file.size, 0);
        },

        // 格式化文件大小
        formatFileSize(bytes) {
            if (bytes === 0) return '0 Bytes';
            const k = 1024;
            const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        },

        // 构建特定类型的config对象
        buildConfig() {
            const config = {};
            
            switch (this.formData.type) {
                case 'mlmodel':
                    config.model_type = this.formData.config.model_type;
                    config.model_path = this.formData.config.model_path;
                    config.service_port = parseInt(this.formData.config.service_port);
                    if (this.formData.config.framework) {
                        config.framework = this.formData.config.framework;
                    }
                    break;
                    
                case 'process':
                    config.command = this.formData.config.command;
                    if (this.formData.config.args_str) {
                        config.args = this.formData.config.args_str.split(' ').filter(a => a);
                    }
                    break;
                    
                case 'opencv':
                    config.cascade_type = this.formData.config.cascade_type;
                    config.service_port = parseInt(this.formData.config.service_port);
                    if (this.formData.config.cascade_path) {
                        config.cascade_path = this.formData.config.cascade_path;
                    }
                    break;
                    
                case 'container':
                    config.image = this.formData.config.image;
                    break;
                    
                case 'data':
                    config.upload_method = this.formData.config.upload_method;
                    config.data_type = this.formData.config.data_type;
                    config.access_mode = this.formData.config.access_mode;
                    
                    // 处理标签
                    if (this.formData.config.tags) {
                        config.tags = this.formData.config.tags.split(',').map(t => t.trim()).filter(t => t);
                    }
                    
                    if (this.formData.config.upload_method === 'file') {
                        config.file_name = this.formData.config.file_name;
                    } else if (this.formData.config.upload_method === 'url') {
                        config.download_url = this.formData.config.download_url;
                        config.file_name = this.formData.config.file_name;
                    } else if (this.formData.config.upload_method === 'path') {
                        config.file_path = this.formData.config.file_path;
                    } else if (this.formData.config.upload_method === 'directory') {
                        config.file_count = this.formData.config.file_count;
                        config.total_size = this.formData.config.total_size;
                    }
                    break;
                case 'datagateway':
                    config.service_port = parseInt(this.formData.config.service_port);
                    config.service_host = this.formData.config.service_host || '127.0.0.1';
                    config.base_path = this.formData.config.base_path || '/tmp/cnet_data';
                    config.bucket = this.formData.config.bucket || 'cnet';
                    if (this.formData.config.auth_token) {
                        config.auth_token = this.formData.config.auth_token;
                    }
                    break;
            }
            
            return config;
        },
        
        // 提交工作负载
        async submitWorkload() {
            try {
                // 如果是数据类型且是文件上传，使用FormData
                if (this.formData.type === 'data' && this.formData.config.upload_method === 'file' && this.selectedFile) {
                    await this.submitDataWorkload();
                    return;
                }
                // 如果是数据类型且是目录上传，使用FormData（多文件）
                if (this.formData.type === 'data' && this.formData.config.upload_method === 'directory' && this.selectedDirectory && this.selectedDirectory.length > 0) {
                    await this.submitDataDirectoryWorkload();
                    return;
                }
                
                // 其他类型使用JSON
                const workloadData = {
                    name: this.formData.name,
                    type: this.formData.type,
                    requirements: {
                        cpu: parseFloat(this.formData.requirements.cpu),
                        memory: parseInt(this.formData.requirements.memory) * 1024 * 1024, // MB转字节
                        gpu: parseInt(this.formData.requirements.gpu),
                        storage: parseInt(this.formData.requirements.storage) * 1024 * 1024
                    },
                    config: this.buildConfig()
                };
                
                const response = await fetch('/api/workloads', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify(workloadData)
                });
                
                if (response.ok) {
                    const result = await response.json();
                    this.closeSubmitForm();
                    await this.fetchWorkloads();
                    
                    // 根据类型显示不同的成功消息
                    if (this.formData.type === 'mlmodel') {
                        alert(`ML模型部署成功！\n\nEndpoint: ${result.endpoint || 'N/A'}\n\n您可以通过此endpoint调用推理服务`);
                    } else if (this.formData.type === 'datagateway') {
                        alert(`数据网关已启动！\n\nEndpoint: ${result.endpoint || 'N/A'}\n\nS3 列举: ${result.endpoint || ''}/s3/${this.formData.config.bucket || 'cnet'}?list-type=2`);
                    } else if (this.formData.type === 'data') {
                        alert(`数据上传成功！\n\n数据键: ${result.data_key || 'N/A'}\n大小: ${this.formatFileSize(result.size || 0)}`);
                    } else {
                        alert('工作负载提交成功！');
                    }
                } else {
                    const error = await response.json();
                    alert('提交失败: ' + (error.error || error.details || '未知错误'));
                }
            } catch (error) {
                console.error('提交工作负载失败:', error);
                alert('提交失败: ' + error.message);
            }
        },

        // 提交数据workload（文件上传）
        async submitDataWorkload() {
            const formData = new FormData();
            
            // 基本信息
            formData.append('name', this.formData.name);
            formData.append('type', 'data');
            formData.append('requirements', JSON.stringify({
                cpu: parseFloat(this.formData.requirements.cpu),
                memory: parseInt(this.formData.requirements.memory) * 1024 * 1024,
                gpu: parseInt(this.formData.requirements.gpu),
                storage: parseInt(this.formData.requirements.storage) // 文件大小已经是字节，不需要转换
            }));
            formData.append('config', JSON.stringify(this.buildConfig()));
            
            // 文件数据
            formData.append('file', this.selectedFile);
            
            try {
                const response = await fetch('/api/workloads', {
                    method: 'POST',
                    body: formData
                });
                
                if (response.ok) {
                    const result = await response.json();
                    this.closeSubmitForm();
                    await this.fetchWorkloads();
                    
                    alert(`数据上传成功！\n\n数据键: ${result.data_key || 'N/A'}\n大小: ${this.formatFileSize(result.size || 0)}`);
                } else {
                    const error = await response.json();
                    alert('上传失败: ' + (error.error || error.details || '未知错误'));
                }
            } catch (error) {
                console.error('上传失败:', error);
                alert('上传失败: ' + error.message);
            }
        },

        // 提交数据workload（目录上传）
        async submitDataDirectoryWorkload() {
            const formData = new FormData();

            // 基本信息
            formData.append('name', this.formData.name);
            formData.append('type', 'data');
            formData.append('requirements', JSON.stringify({
                cpu: parseFloat(this.formData.requirements.cpu),
                memory: parseInt(this.formData.requirements.memory) * 1024 * 1024,
                gpu: parseInt(this.formData.requirements.gpu),
                storage: parseInt(this.formData.requirements.storage) // 目录总大小（字节）
            }));
            // 明确指定目录上传
            const cfg = { ...this.buildConfig(), upload_method: 'directory' };
            formData.append('config', JSON.stringify(cfg));

            // 追加所有文件，文件名使用相对路径，字段名必须为 'files'
            for (const file of this.selectedDirectory) {
                const rel = file.webkitRelativePath || file.name;
                formData.append('files', file, rel);
            }

            try {
                const response = await fetch('/api/workloads', {
                    method: 'POST',
                    body: formData
                });

                if (response.ok) {
                    const result = await response.json();
                    this.closeSubmitForm();
                    await this.fetchWorkloads();
                    alert(`目录上传成功！\n\n数据键: ${result.data_key || 'N/A'}\n大小: ${this.formatFileSize(result.size || 0)}\n路径: ${result.path || '-'}`);
                } else {
                    const error = await response.json();
                    alert('上传失败: ' + (error.error || error.details || '未知错误'));
                }
            } catch (error) {
                console.error('上传失败:', error);
                alert('上传失败: ' + error.message);
            }
        },
        
        // 关闭提交表单
        closeSubmitForm() {
            this.showSubmitForm = false;
            this.resetForm();
        },
        
        // 重置表单
        resetForm() {
            this.formData = {
                name: '',
                type: 'data',
                config: { ...this.typeDefaults.data.config },
                requirements: { ...this.typeDefaults.data.requirements }
            };
            this.selectedFile = null;
            this.selectedDirectory = [];
        },
        
        // 停止工作负载
        async stopWorkload(workloadId) {
            try {
                const response = await fetch(`/api/workloads/${workloadId}/stop`, {
                    method: 'POST'
                });
                
                if (response.ok) {
                    await this.fetchWorkloads();
                    alert('工作负载已停止');
                } else {
                    alert('停止失败');
                }
            } catch (error) {
                console.error('停止工作负载失败:', error);
                alert('停止失败: ' + error.message);
            }
        },
        
        // 查看工作负载详情
        async viewWorkload(workload) {
            try {
                const response = await fetch(`/api/workloads/${workload.id}`);
                const data = await response.json();
                this.selectedWorkload = data;
                this.showWorkloadModal = true;
            } catch (error) {
                console.error('获取工作负载详情失败:', error);
                alert('获取详情失败: ' + error.message);
            }
        },
        
        // 生成 Data 直链/列举链接（基于运行中的 DataGateway）
        buildDataLinks(selected) {
            if (!this.gatewayEndpoint || !selected || !selected.config) return null;
            const dataKey = selected.config.data_key || selected.data_key;
            if (!dataKey) return null;
            const bucket = this.gatewayBucket || 'cnet';
            const listUrl = `${this.gatewayEndpoint}/s3/${bucket}?list-type=2&prefix=${encodeURIComponent(dataKey)}`;
            // 文件名未知时仅返回列举链接；若存在 file_name/path 也补充下载示例
            const name = selected.config.file_name || selected.file_name || '';
            const downloadUrl = name ? `${this.gatewayEndpoint}/s3/${bucket}/${encodeURIComponent(dataKey)}/${encodeURIComponent(name)}` : '';
            return { listUrl, downloadUrl, bucket, dataKey };
        },
        
        // 复制到剪贴板
        async copyToClipboard(text) {
            try {
                await navigator.clipboard.writeText(text);
                alert('已复制到剪贴板');
            } catch (e) {
                console.error('复制失败', e);
            }
        },
        
        // 格式化内存大小
        formatMemory(bytes) {
            if (bytes === 0) return '0 B';
            const k = 1024;
            const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        },
        
        // 格式化日期
        formatDate(dateString) {
            if (!dateString) return '-';
            return new Date(dateString).toLocaleString('zh-CN');
        },
        
        // 获取状态颜色
        getStatusColor(status) {
            const colors = {
                'pending': 'bg-yellow-100 text-yellow-800',
                'running': 'bg-blue-100 text-blue-800',
                'completed': 'bg-green-100 text-green-800',
                'failed': 'bg-red-100 text-red-800',
                'stopped': 'bg-gray-100 text-gray-800'
            };
            return colors[status] || 'bg-gray-100 text-gray-800';
        },
        
        // 获取工作负载类型颜色
        getWorkloadTypeColor(type) {
            const colors = {
                'process': 'bg-blue-100 text-blue-800',
                'container': 'bg-purple-100 text-purple-800',
                'mlmodel': 'bg-green-100 text-green-800',
                'vision': 'bg-orange-100 text-orange-800'
            };
            return colors[type] || 'bg-gray-100 text-gray-800';
        }
    };
}
