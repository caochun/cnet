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
        
        // UI状态
        showSubmitForm: false,
        showWorkloadModal: false,
        selectedWorkload: null,
        
        // 动态表单数据
        formData: {
            name: '',
            type: 'mlmodel',
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
                cascade_type: 'face'
            },
            requirements: {
                cpu: 2.0,
                memory: 2048,
                gpu: 0,
                storage: 0
            }
        },
        
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
            }
            
            return config;
        },
        
        // 提交工作负载
        async submitWorkload() {
            try {
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
        
        // 关闭提交表单
        closeSubmitForm() {
            this.showSubmitForm = false;
            this.resetForm();
        },
        
        // 重置表单
        resetForm() {
            this.formData = {
                name: '',
                type: 'mlmodel',
                config: { ...this.typeDefaults.mlmodel.config },
                requirements: { ...this.typeDefaults.mlmodel.requirements }
            };
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
