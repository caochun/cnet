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
        
        // 新工作负载表单
        newWorkload: {
            name: '',
            type: 'process',
            requirements: {
                cpu: 1,
                memory: 512,
                gpu: 0,
                storage: 0
            },
            configJson: '{}'
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
        
        // 提交工作负载
        async submitWorkload() {
            try {
                const config = JSON.parse(this.newWorkload.configJson);
                
                const workloadData = {
                    name: this.newWorkload.name,
                    type: this.newWorkload.type,
                    requirements: this.newWorkload.requirements,
                    config: config
                };
                
                const response = await fetch('/api/workloads', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify(workloadData)
                });
                
                if (response.ok) {
                    this.showSubmitForm = false;
                    this.resetNewWorkloadForm();
                    await this.fetchWorkloads();
                    alert('工作负载提交成功！');
                } else {
                    const error = await response.json();
                    alert('提交失败: ' + (error.error || '未知错误'));
                }
            } catch (error) {
                console.error('提交工作负载失败:', error);
                alert('提交失败: ' + error.message);
            }
        },
        
        // 重置新工作负载表单
        resetNewWorkloadForm() {
            this.newWorkload = {
                name: '',
                type: 'process',
                requirements: {
                    cpu: 1,
                    memory: 512,
                    gpu: 0,
                    storage: 0
                },
                configJson: '{}'
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
