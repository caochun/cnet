// CNET Agent Web UI JavaScript

class CNETApp {
    constructor() {
        this.apiBase = '/api';
        this.currentPage = 'dashboard';
        this.refreshInterval = null;
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.loadDashboard();
        this.startAutoRefresh();
    }

    setupEventListeners() {
        // Sidebar navigation
        document.querySelectorAll('.menu-item').forEach(item => {
            item.addEventListener('click', (e) => {
                const page = e.currentTarget.dataset.page;
                this.navigateToPage(page);
            });
        });

        // Sidebar toggle for mobile
        document.querySelector('.sidebar-toggle').addEventListener('click', () => {
            document.querySelector('.sidebar').classList.toggle('open');
        });

        // Refresh button
        document.getElementById('refresh-btn').addEventListener('click', () => {
            this.refreshCurrentPage();
        });

        // Refresh nodes button
        document.getElementById('refresh-nodes-btn').addEventListener('click', () => {
            this.loadDashboard();
        });

        // Create task button
        document.getElementById('create-task-btn').addEventListener('click', () => {
            this.showCreateTaskModal();
        });

        // ML Models buttons
        document.getElementById('refresh-models-btn').addEventListener('click', () => {
            this.loadModels();
        });

        document.getElementById('create-model-btn').addEventListener('click', () => {
            this.showCreateModelModal();
        });


        // Hierarchy management buttons
        document.getElementById('assign-hierarchy-btn').addEventListener('click', () => {
            this.showAssignHierarchyModal();
        });

        document.getElementById('resolve-hierarchy-btn').addEventListener('click', () => {
            this.showResolveHierarchyModal();
        });

        // Modal controls
        document.querySelectorAll('.modal-close').forEach(btn => {
            btn.addEventListener('click', (e) => {
                this.closeModal(e.target.closest('.modal'));
            });
        });

        // Create task form
        document.getElementById('submit-task-btn').addEventListener('click', () => {
            this.submitCreateTask();
        });

        document.getElementById('cancel-task-btn').addEventListener('click', () => {
            this.closeModal(document.getElementById('create-task-modal'));
        });

        // Task preset handler
        document.getElementById('task-preset').addEventListener('change', (e) => {
            this.handleTaskPreset(e.target.value);
        });

        // Hierarchy management form handlers
        document.getElementById('submit-hierarchy-btn').addEventListener('click', () => {
            this.submitAssignHierarchy();
        });

        document.getElementById('cancel-hierarchy-btn').addEventListener('click', () => {
            this.closeModal(document.getElementById('assign-hierarchy-modal'));
        });

        document.getElementById('submit-resolve-btn').addEventListener('click', () => {
            this.submitResolveHierarchy();
        });

        document.getElementById('cancel-resolve-btn').addEventListener('click', () => {
            this.closeModal(document.getElementById('resolve-hierarchy-modal'));
        });


        // Close modals on outside click
        document.querySelectorAll('.modal').forEach(modal => {
            modal.addEventListener('click', (e) => {
                if (e.target === modal) {
                    this.closeModal(modal);
                }
            });
        });
    }

    navigateToPage(page) {
        // Update active menu item
        document.querySelectorAll('.menu-item').forEach(item => {
            item.classList.remove('active');
        });
        document.querySelector(`[data-page="${page}"]`).classList.add('active');

        // Hide all pages
        document.querySelectorAll('.page').forEach(p => {
            p.style.display = 'none';
        });

        // Show target page
        document.getElementById(`${page}-page`).style.display = 'block';

        // Update page title
        document.getElementById('page-title').textContent = this.getPageTitle(page);

        this.currentPage = page;
        this.loadPageData(page);
    }

    getPageTitle(page) {
        const titles = {
            dashboard: 'Dashboard',
            tasks: 'Tasks',
            resources: 'Resources',
            nodes: 'Nodes',
            models: 'ML Models',
            logs: 'Logs'
        };
        return titles[page] || 'Dashboard';
    }

    loadPageData(page) {
        switch (page) {
            case 'dashboard':
                this.loadDashboard();
                break;
            case 'tasks':
                this.loadTasks();
                break;
            case 'resources':
                this.loadResources();
                break;
            case 'nodes':
                this.loadNodes();
                break;
            case 'models':
                this.loadModels();
                break;
            case 'logs':
                this.loadLogs();
                break;
        }
    }

    async loadDashboard() {
        try {
            // Load node info
            const nodeInfo = await this.apiCall('/node');
            document.getElementById('node-name').textContent = nodeInfo.node_name || 'Unknown';

            // Load node hierarchy information
            try {
                const hierarchyInfo = await this.apiCall('/node/hierarchy');
                document.getElementById('node-hierarchy-id').textContent = hierarchyInfo.hierarchy_id || 'N/A';
                document.getElementById('node-level').textContent = `Level ${hierarchyInfo.level || 0}`;
            } catch (error) {
                console.warn('Failed to load hierarchy info:', error);
                document.getElementById('node-hierarchy-id').textContent = 'N/A';
                document.getElementById('node-level').textContent = 'Level 0';
            }

            // Load resource usage
            const usage = await this.apiCall('/resources/usage');
            this.updateResourceUsage(usage);

            // Load tasks summary
            const tasks = await this.apiCall('/tasks');
            this.updateTasksSummary(tasks);

            // Load registered nodes
            const nodes = await this.apiCall('/discovery/nodes');
            this.updateRegisteredNodes(nodes);

            // Update agent status
            this.updateAgentStatus(true);
        } catch (error) {
            console.error('Failed to load dashboard:', error);
            this.updateAgentStatus(false);
        }
    }

    async loadTasks() {
        try {
            const tasks = await this.apiCall('/tasks');
            this.renderTasksTable(tasks);
        } catch (error) {
            console.error('Failed to load tasks:', error);
            this.showError('Failed to load tasks');
        }
    }

    async loadResources() {
        try {
            const resources = await this.apiCall('/resources');
            this.renderResourceDetails(resources);
        } catch (error) {
            console.error('Failed to load resources:', error);
            this.showError('Failed to load resources');
        }
    }

    async loadNodes() {
        try {
            const nodes = await this.apiCall('/discovery/nodes');
            this.renderNodesTable(nodes);
        } catch (error) {
            console.error('Failed to load nodes:', error);
            this.showError('Failed to load nodes');
        }
    }

    async loadLogs() {
        try {
            // For now, show a placeholder
            const logsContainer = document.getElementById('logs-container');
            logsContainer.innerHTML = `
                <div class="log-entry info">[INFO] CNET Agent started</div>
                <div class="log-entry info">[INFO] Resources service started</div>
                <div class="log-entry info">[INFO] Tasks service started</div>
                <div class="log-entry info">[INFO] Discovery service started</div>
            `;
        } catch (error) {
            console.error('Failed to load logs:', error);
            this.showError('Failed to load logs');
        }
    }

    updateResourceUsage(usage) {
        if (usage.cpu) {
            const cpuPercent = Math.round(usage.cpu.percent);
            document.getElementById('cpu-usage').style.width = `${cpuPercent}%`;
            document.getElementById('cpu-percent').textContent = `${cpuPercent}%`;
        }

        if (usage.memory) {
            const memoryPercent = Math.round(usage.memory.percent);
            document.getElementById('memory-usage').style.width = `${memoryPercent}%`;
            document.getElementById('memory-percent').textContent = `${memoryPercent}%`;
        }

        if (usage.disk) {
            const diskPercent = Math.round(usage.disk.percent);
            document.getElementById('disk-usage').style.width = `${diskPercent}%`;
            document.getElementById('disk-percent').textContent = `${diskPercent}%`;
        }
    }

    updateTasksSummary(tasks) {
        const totalTasks = tasks.length;
        const runningTasks = tasks.filter(t => t.status === 'running').length;
        const completedTasks = tasks.filter(t => t.status === 'completed').length;

        document.getElementById('total-tasks').textContent = totalTasks;
        document.getElementById('running-tasks').textContent = runningTasks;
        document.getElementById('completed-tasks').textContent = completedTasks;

        // Update recent tasks
        this.renderRecentTasks(tasks.slice(0, 5));
    }

    renderRecentTasks(tasks) {
        const container = document.getElementById('recent-tasks');
        
        if (tasks.length === 0) {
            container.innerHTML = `
                <div class="empty-state">
                    <i class="fas fa-tasks"></i>
                    <p>No recent tasks</p>
                </div>
            `;
            return;
        }

        const html = tasks.map(task => `
            <div class="task-item" style="display: flex; justify-content: space-between; align-items: center; padding: 0.5rem 0; border-bottom: 1px solid var(--border-color);">
                <div>
                    <div style="font-weight: 500;">${task.name}</div>
                    <div style="font-size: 0.875rem; color: var(--text-secondary);">${task.type} • ${task.command}</div>
                </div>
                <span class="status-badge ${task.status}">${task.status}</span>
            </div>
        `).join('');

        container.innerHTML = html;
    }

    updateRegisteredNodes(nodes) {
        const container = document.getElementById('registered-nodes');
        
        if (nodes.length === 0) {
            container.innerHTML = `
                <div class="empty-state">
                    <i class="fas fa-network-wired"></i>
                    <p>No registered nodes</p>
                </div>
            `;
            return;
        }

        const html = nodes.map(node => `
            <div class="node-item" style="display: flex; justify-content: space-between; align-items: center; padding: 0.75rem 0; border-bottom: 1px solid var(--border-color);">
                <div style="flex: 1;">
                    <div style="display: flex; align-items: center; gap: 0.5rem; margin-bottom: 0.25rem;">
                        <i class="fas fa-server" style="color: var(--primary-color);"></i>
                        <div style="font-weight: 500;">${node.name}</div>
                        <span class="status-badge ${node.status}">${node.status}</span>
                    </div>
                    <div style="font-size: 0.875rem; color: var(--text-secondary);">
                        ${node.address}:${node.port} • ${node.region} • ${node.datacenter}
                    </div>
                    <div style="font-size: 0.75rem; color: var(--text-secondary); margin-top: 0.25rem;">
                        Last seen: ${new Date(node.last_seen).toLocaleString()}
                    </div>
                </div>
                <div style="text-align: right;">
                    <div style="font-size: 0.75rem; color: var(--text-secondary); font-family: monospace;">
                        ${node.id.substring(0, 8)}...
                    </div>
                </div>
            </div>
        `).join('');

        container.innerHTML = html;
        
        // Update cluster stats
        this.updateClusterStats(nodes);
    }

    updateClusterStats(nodes) {
        const totalNodes = nodes.length;
        const activeNodes = nodes.filter(node => node.status === 'active').length;
        const regions = [...new Set(nodes.map(node => node.region))].length;

        document.getElementById('total-nodes').textContent = totalNodes;
        document.getElementById('active-nodes').textContent = activeNodes;
        document.getElementById('total-regions').textContent = regions;
    }

    renderTasksTable(tasks) {
        const tbody = document.getElementById('tasks-tbody');
        
        if (tasks.length === 0) {
            tbody.innerHTML = `
                <tr>
                    <td colspan="6" style="text-align: center; padding: 2rem; color: var(--text-secondary);">
                        No tasks found
                    </td>
                </tr>
            `;
            return;
        }

        const html = tasks.map(task => `
            <tr>
                <td style="font-family: monospace; font-size: 0.875rem;">${task.id.substring(0, 8)}...</td>
                <td>${task.name}</td>
                <td>${task.type}</td>
                <td><span class="status-badge ${task.status}">${task.status}</span></td>
                <td>${new Date(task.created_at).toLocaleString()}</td>
                <td>
                    <button class="btn btn-secondary" onclick="app.viewTask('${task.id}')" style="padding: 0.25rem 0.5rem; font-size: 0.75rem;">
                        <i class="fas fa-eye"></i>
                    </button>
                    ${task.status === 'running' ? `
                        <button class="btn btn-danger" onclick="app.stopTask('${task.id}')" style="padding: 0.25rem 0.5rem; font-size: 0.75rem; margin-left: 0.25rem;">
                            <i class="fas fa-stop"></i>
                        </button>
                    ` : ''}
                </td>
            </tr>
        `).join('');

        tbody.innerHTML = html;
    }

    renderResourceDetails(resources) {
        const container = document.getElementById('resource-details');
        
        const html = `
            <div class="resource-section">
                <h4>CPU Information</h4>
                <div class="resource-info">
                    <div class="resource-info-item">
                        <span class="resource-info-label">Count</span>
                        <span class="resource-info-value">${resources.cpu.count}</span>
                    </div>
                    <div class="resource-info-item">
                        <span class="resource-info-label">Model</span>
                        <span class="resource-info-value">${resources.cpu.model_name}</span>
                    </div>
                    <div class="resource-info-item">
                        <span class="resource-info-label">Frequency</span>
                        <span class="resource-info-value">${resources.cpu.mhz} MHz</span>
                    </div>
                </div>
            </div>
            
            <div class="resource-section">
                <h4>Memory Information</h4>
                <div class="resource-info">
                    <div class="resource-info-item">
                        <span class="resource-info-label">Total</span>
                        <span class="resource-info-value">${this.formatBytes(resources.memory.total)}</span>
                    </div>
                    <div class="resource-info-item">
                        <span class="resource-info-label">Available</span>
                        <span class="resource-info-value">${this.formatBytes(resources.memory.available)}</span>
                    </div>
                    <div class="resource-info-item">
                        <span class="resource-info-label">Used</span>
                        <span class="resource-info-value">${this.formatBytes(resources.memory.used)}</span>
                    </div>
                </div>
            </div>
            
            <div class="resource-section">
                <h4>Disk Information</h4>
                <div class="resource-info">
                    <div class="resource-info-item">
                        <span class="resource-info-label">Total</span>
                        <span class="resource-info-value">${this.formatBytes(resources.disk.total)}</span>
                    </div>
                    <div class="resource-info-item">
                        <span class="resource-info-label">Used</span>
                        <span class="resource-info-value">${this.formatBytes(resources.disk.used)}</span>
                    </div>
                    <div class="resource-info-item">
                        <span class="resource-info-label">Free</span>
                        <span class="resource-info-value">${this.formatBytes(resources.disk.free)}</span>
                    </div>
                </div>
            </div>
        `;

        container.innerHTML = html;
    }

    renderNodesTable(nodes) {
        const tbody = document.getElementById('nodes-tbody');
        
        if (nodes.length === 0) {
            tbody.innerHTML = `
                <tr>
                    <td colspan="8" style="text-align: center; padding: 2rem; color: var(--text-secondary);">
                        No nodes discovered
                    </td>
                </tr>
            `;
            return;
        }

        const html = nodes.map(node => `
            <tr>
                <td style="font-family: monospace; font-size: 0.875rem;">${node.id.substring(0, 8)}...</td>
                <td style="font-family: monospace; font-size: 0.875rem; color: var(--primary-color);">
                    ${node.hierarchy_id || 'N/A'}
                </td>
                <td>${node.name}</td>
                <td>${node.address}:${node.port}</td>
                <td>${node.region}</td>
                <td><span class="level-badge">Level ${node.level || 0}</span></td>
                <td><span class="status-badge ${node.status}">${node.status}</span></td>
                <td>${new Date(node.last_seen).toLocaleString()}</td>
            </tr>
        `).join('');

        tbody.innerHTML = html;
    }

    showCreateTaskModal() {
        document.getElementById('create-task-modal').classList.add('show');
    }

    async submitCreateTask() {
        const form = document.getElementById('create-task-form');
        const formData = new FormData(form);
        
        const taskData = {
            name: formData.get('name'),
            type: formData.get('type'),
            command: formData.get('command'),
            args: formData.get('args').split('\n').filter(arg => arg.trim()),
            env: this.parseEnvVars(formData.get('env'))
        };

        try {
            await this.apiCall('/tasks', 'POST', taskData);
            this.closeModal(document.getElementById('create-task-modal'));
            this.loadTasks();
            this.showSuccess('Task created successfully');
        } catch (error) {
            console.error('Failed to create task:', error);
            this.showError('Failed to create task');
        }
    }

    parseEnvVars(envStr) {
        if (!envStr.trim()) return {};
        try {
            return JSON.parse(envStr);
        } catch (error) {
            return {};
        }
    }

    async viewTask(taskId) {
        try {
            const task = await this.apiCall(`/tasks/${taskId}`);
            this.showTaskDetails(task);
        } catch (error) {
            console.error('Failed to load task details:', error);
            this.showError('Failed to load task details');
        }
    }

    showTaskDetails(task) {
        const content = document.getElementById('task-details-content');
        
        const html = `
            <div style="margin-bottom: 1rem;">
                <h4>Task Information</h4>
                <div class="resource-info">
                    <div class="resource-info-item">
                        <span class="resource-info-label">ID</span>
                        <span class="resource-info-value" style="font-family: monospace;">${task.id}</span>
                    </div>
                    <div class="resource-info-item">
                        <span class="resource-info-label">Name</span>
                        <span class="resource-info-value">${task.name}</span>
                    </div>
                    <div class="resource-info-item">
                        <span class="resource-info-label">Type</span>
                        <span class="resource-info-value">${task.type}</span>
                    </div>
                    <div class="resource-info-item">
                        <span class="resource-info-label">Status</span>
                        <span class="resource-info-value"><span class="status-badge ${task.status}">${task.status}</span></span>
                    </div>
                    <div class="resource-info-item">
                        <span class="resource-info-label">Command</span>
                        <span class="resource-info-value">${task.command}</span>
                    </div>
                    <div class="resource-info-item">
                        <span class="resource-info-label">Created</span>
                        <span class="resource-info-value">${new Date(task.created_at).toLocaleString()}</span>
                    </div>
                    ${task.started_at ? `
                        <div class="resource-info-item">
                            <span class="resource-info-label">Started</span>
                            <span class="resource-info-value">${new Date(task.started_at).toLocaleString()}</span>
                        </div>
                    ` : ''}
                    ${task.stopped_at ? `
                        <div class="resource-info-item">
                            <span class="resource-info-label">Stopped</span>
                            <span class="resource-info-value">${new Date(task.stopped_at).toLocaleString()}</span>
                        </div>
                    ` : ''}
                    ${task.exit_code !== null ? `
                        <div class="resource-info-item">
                            <span class="resource-info-label">Exit Code</span>
                            <span class="resource-info-value">${task.exit_code}</span>
                        </div>
                    ` : ''}
                </div>
            </div>
        `;

        content.innerHTML = html;

        // Show/hide stop button based on status
        const stopBtn = document.getElementById('stop-task-btn');
        if (task.status === 'running') {
            stopBtn.style.display = 'inline-flex';
            stopBtn.onclick = () => this.stopTask(task.id);
        } else {
            stopBtn.style.display = 'none';
        }

        document.getElementById('task-details-modal').classList.add('show');
    }

    async stopTask(taskId) {
        try {
            await this.apiCall(`/tasks/${taskId}`, 'DELETE');
            this.closeModal(document.getElementById('task-details-modal'));
            this.loadTasks();
            this.showSuccess('Task stopped successfully');
        } catch (error) {
            console.error('Failed to stop task:', error);
            this.showError('Failed to stop task');
        }
    }

    closeModal(modal) {
        modal.classList.remove('show');
    }

    updateAgentStatus(connected) {
        const statusDot = document.getElementById('agent-status');
        const statusText = document.getElementById('agent-status-text');
        
        if (connected) {
            statusDot.classList.add('connected');
            statusText.textContent = 'Connected';
        } else {
            statusDot.classList.remove('connected');
            statusText.textContent = 'Disconnected';
        }
    }

    startAutoRefresh() {
        this.refreshInterval = setInterval(() => {
            if (this.currentPage === 'dashboard') {
                this.loadDashboard();
            }
        }, 5000); // Refresh every 5 seconds
    }

    refreshCurrentPage() {
        this.loadPageData(this.currentPage);
    }

    async apiCall(endpoint, method = 'GET', data = null) {
        const options = {
            method,
            headers: {
                'Content-Type': 'application/json',
            }
        };

        if (data) {
            options.body = JSON.stringify(data);
        }

        const response = await fetch(`${this.apiBase}${endpoint}`, options);
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        return await response.json();
    }

    formatBytes(bytes) {
        if (bytes === 0) return '0 Bytes';
        const k = 1024;
        const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }

    showSuccess(message) {
        // Simple success notification
        const notification = document.createElement('div');
        notification.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            background-color: var(--success-color);
            color: white;
            padding: 1rem;
            border-radius: 0.375rem;
            box-shadow: var(--shadow-lg);
            z-index: 3000;
        `;
        notification.textContent = message;
        document.body.appendChild(notification);
        
        setTimeout(() => {
            document.body.removeChild(notification);
        }, 3000);
    }

    showError(message) {
        // Simple error notification
        const notification = document.createElement('div');
        notification.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            background-color: var(--danger-color);
            color: white;
            padding: 1rem;
            border-radius: 0.375rem;
            box-shadow: var(--shadow-lg);
            z-index: 3000;
        `;
        notification.textContent = message;
        document.body.appendChild(notification);
        
        setTimeout(() => {
            document.body.removeChild(notification);
        }, 5000);
    }

    // Hierarchy management methods
    async showAssignHierarchyModal() {
        try {
            // Load available nodes
            const nodes = await this.apiCall('/discovery/nodes');
            const select = document.getElementById('hierarchy-node-id');
            
            // Clear existing options
            select.innerHTML = '<option value="">Select a node...</option>';
            
            // Add node options
            nodes.forEach(node => {
                const option = document.createElement('option');
                option.value = node.id;
                option.textContent = `${node.name} (${node.id})`;
                select.appendChild(option);
            });
            
            this.showModal(document.getElementById('assign-hierarchy-modal'));
        } catch (error) {
            this.showError(`Failed to load nodes: ${error.message}`);
        }
    }

    async showResolveHierarchyModal() {
        this.showModal(document.getElementById('resolve-hierarchy-modal'));
    }

    async submitAssignHierarchy() {
        const form = document.getElementById('assign-hierarchy-form');
        const formData = new FormData(form);
        const nodeId = formData.get('node_id');
        
        if (!nodeId) {
            this.showError('Please select a node');
            return;
        }
        
        try {
            const response = await this.apiCall('/discovery/hierarchy/assign', 'POST', {
                node_id: nodeId
            });
            
            this.showSuccess(`Hierarchy ID assigned: ${response.hierarchy_id}`);
            this.closeModal(document.getElementById('assign-hierarchy-modal'));
            this.loadPageData('nodes');
        } catch (error) {
            this.showError(`Failed to assign hierarchy ID: ${error.message}`);
        }
    }

    async submitResolveHierarchy() {
        const form = document.getElementById('resolve-hierarchy-form');
        const formData = new FormData(form);
        const hierarchyId = formData.get('hierarchy_id');
        
        if (!hierarchyId) {
            this.showError('Please enter a hierarchy ID');
            return;
        }
        
        try {
            const response = await this.apiCall('/discovery/hierarchy/resolve', 'POST', {
                hierarchy_id: hierarchyId
            });
            
            // Display resolution result
            const resultDiv = document.getElementById('resolve-result');
            const contentDiv = document.getElementById('resolve-content');
            
            contentDiv.innerHTML = `
                <div class="resolve-result">
                    <p><strong>Node ID:</strong> ${response.node_id}</p>
                    <p><strong>Address:</strong> ${response.address}:${response.port}</p>
                    <p><strong>Status:</strong> <span class="status-${response.status}">${response.status}</span></p>
                    <p><strong>Last Seen:</strong> ${response.last_seen}</p>
                </div>
            `;
            
            resultDiv.style.display = 'block';
        } catch (error) {
            this.showError(`Failed to resolve hierarchy ID: ${error.message}`);
        }
    }

    // Task Preset Handler
    handleTaskPreset(preset) {
        const nameField = document.getElementById('task-name');
        const commandField = document.getElementById('task-command');
        const argsField = document.getElementById('task-args');
        const envField = document.getElementById('task-env');
        const workingDirField = document.getElementById('task-working-dir');
        const cpuLimitField = document.getElementById('task-cpu-limit');
        const memoryLimitField = document.getElementById('task-memory-limit');

        switch(preset) {
            case 'ml-linear-regression':
                nameField.value = 'linear-regression-model';
                commandField.value = 'python3';
                argsField.value = 'examples/ml_models/simple_linear_regression.py,train,models/linear_regression_model.joblib,1000';
                envField.value = 'PYTHONPATH=examples/ml_models,MODEL_PATH=models/linear_regression_model.joblib,MODEL_TYPE=linear_regression';
                workingDirField.value = '.';
                cpuLimitField.value = '1.0';
                memoryLimitField.value = '512';
                break;
            case 'ml-neural-network':
                nameField.value = 'neural-network-model';
                commandField.value = 'python3';
                argsField.value = 'examples/ml_models/neural_network.py,train,models/neural_network_model.h5,1000,50';
                envField.value = 'PYTHONPATH=examples/ml_models,MODEL_PATH=models/neural_network_model.h5,MODEL_TYPE=neural_network';
                workingDirField.value = '.';
                cpuLimitField.value = '2.0';
                memoryLimitField.value = '1024';
                break;
            case 'ml-custom':
                nameField.value = 'custom-ml-model';
                commandField.value = 'python3';
                argsField.value = 'your_script.py,train,models/your_model.joblib,1000';
                envField.value = 'PYTHONPATH=examples/ml_models,MODEL_PATH=models/your_model.joblib,MODEL_TYPE=custom';
                workingDirField.value = '.';
                cpuLimitField.value = '1.0';
                memoryLimitField.value = '512';
                break;
            default:
                // Clear fields for custom task
                nameField.value = '';
                commandField.value = '';
                argsField.value = '';
                envField.value = '';
                workingDirField.value = '.';
                cpuLimitField.value = '1.0';
                memoryLimitField.value = '512';
                break;
        }
    }

    // ML Models Management
    async loadModels() {
        try {
            const response = await fetch(`${this.apiBase}/ml/models`);
            const models = await response.json();
            
            const tbody = document.getElementById('models-tbody');
            tbody.innerHTML = '';
            
            models.forEach(model => {
                const row = document.createElement('tr');
                row.innerHTML = `
                    <td>${model.id.substring(0, 8)}...</td>
                    <td>${model.name}</td>
                    <td>${model.type}</td>
                    <td>${model.engine}</td>
                    <td><span class="status status-${model.status}">${model.status}</span></td>
                    <td>${model.endpoint || 'N/A'}</td>
                    <td>${new Date(model.created_at).toLocaleString()}</td>
                    <td>
                        <button class="btn btn-sm btn-info" onclick="app.viewModelLogs('${model.id}')">
                            <i class="fas fa-file-alt"></i> Logs
                        </button>
                        <button class="btn btn-sm btn-warning" onclick="app.viewModelInfo('${model.id}')">
                            <i class="fas fa-info"></i> Info
                        </button>
                        <button class="btn btn-sm btn-success" onclick="app.testModel('${model.id}')">
                            <i class="fas fa-play"></i> Test
                        </button>
                        <button class="btn btn-sm btn-danger" onclick="app.stopModel('${model.id}')">
                            <i class="fas fa-stop"></i> Stop
                        </button>
                    </td>
                `;
                tbody.appendChild(row);
            });
        } catch (error) {
            this.showError(`Failed to load models: ${error.message}`);
        }
    }

    showCreateModelModal() {
        const modal = document.createElement('div');
        modal.className = 'modal';
        modal.innerHTML = `
            <div class="modal-content">
                <div class="modal-header">
                    <h3>Create ML Model</h3>
                    <button class="modal-close">&times;</button>
                </div>
                <div class="modal-body">
                    <form id="create-model-form">
                        <div class="form-group">
                            <label for="model-name">Model Name</label>
                            <input type="text" id="model-name" required>
                        </div>
                        <div class="form-group">
                            <label for="model-type">Model Type</label>
                            <select id="model-type" required>
                                <option value="linear_regression">Linear Regression</option>
                                <option value="neural_network">Neural Network</option>
                                <option value="custom">Custom</option>
                            </select>
                        </div>
                        <div class="form-group">
                            <label for="model-engine">Inference Engine</label>
                            <select id="model-engine" required>
                                <option value="python">Python</option>
                                <option value="tensorflow">TensorFlow</option>
                                <option value="pytorch">PyTorch</option>
                            </select>
                        </div>
                        <div class="form-group">
                            <label for="model-path">Model Path</label>
                            <input type="text" id="model-path" placeholder="models/model.joblib" required>
                        </div>
                        <div class="form-group">
                            <label for="script-path">Script Path</label>
                            <input type="text" id="script-path" placeholder="examples/ml_models/script.py" required>
                        </div>
                        <div class="form-group">
                            <label for="model-cpu-limit">CPU Limit</label>
                            <input type="number" id="model-cpu-limit" value="1.0" step="0.1" min="0.1">
                        </div>
                        <div class="form-group">
                            <label for="model-memory-limit">Memory Limit (MB)</label>
                            <input type="number" id="model-memory-limit" value="512" min="64">
                        </div>
                        <div class="form-actions">
                            <button type="submit" class="btn btn-primary">Create Model</button>
                            <button type="button" class="btn btn-secondary modal-close">Cancel</button>
                        </div>
                    </form>
                </div>
            </div>
        `;
        
        document.body.appendChild(modal);
        
        // Event listeners
        modal.querySelector('.modal-close').addEventListener('click', () => {
            document.body.removeChild(modal);
        });
        
        modal.querySelector('#create-model-form').addEventListener('submit', (e) => {
            e.preventDefault();
            this.createModel();
        });
    }

    async createModel() {
        try {
            const formData = {
                name: document.getElementById('model-name').value,
                type: document.getElementById('model-type').value,
                engine: document.getElementById('model-engine').value,
                model_path: document.getElementById('model-path').value,
                script_path: document.getElementById('script-path').value,
                resources: {
                    cpu_limit: parseFloat(document.getElementById('model-cpu-limit').value),
                    memory_limit: parseInt(document.getElementById('model-memory-limit').value) * 1024 * 1024,
                    disk_limit: 1024 * 1024 * 1024,
                    gpu_limit: 0
                }
            };

            const response = await fetch(`${this.apiBase}/ml/models`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(formData)
            });

            if (response.ok) {
                this.showSuccess('Model created successfully');
                document.querySelector('.modal-close').click();
                this.loadModels();
            } else {
                const error = await response.json();
                this.showError(`Failed to create model: ${error.error}`);
            }
        } catch (error) {
            this.showError(`Failed to create model: ${error.message}`);
        }
    }

    async viewModelLogs(modelId) {
        try {
            const response = await fetch(`${this.apiBase}/ml/models/${modelId}/logs`);
            const logs = await response.json();
            
            const modal = document.createElement('div');
            modal.className = 'modal';
            modal.innerHTML = `
                <div class="modal-content">
                    <div class="modal-header">
                        <h3>Model Logs</h3>
                        <button class="modal-close">&times;</button>
                    </div>
                    <div class="modal-body">
                        <div class="logs-container" style="max-height: 400px; overflow-y: auto;">
                            ${logs.map(log => `<div class="log-line">${log}</div>`).join('')}
                        </div>
                    </div>
                </div>
            `;
            
            document.body.appendChild(modal);
            modal.querySelector('.modal-close').addEventListener('click', () => {
                document.body.removeChild(modal);
            });
        } catch (error) {
            this.showError(`Failed to load model logs: ${error.message}`);
        }
    }

    async viewModelInfo(modelId) {
        try {
            const response = await fetch(`${this.apiBase}/ml/models/${modelId}/info`);
            const info = await response.json();
            
            const modal = document.createElement('div');
            modal.className = 'modal';
            modal.innerHTML = `
                <div class="modal-content">
                    <div class="modal-header">
                        <h3>Model Information</h3>
                        <button class="modal-close">&times;</button>
                    </div>
                    <div class="modal-body">
                        <pre>${JSON.stringify(info, null, 2)}</pre>
                    </div>
                </div>
            `;
            
            document.body.appendChild(modal);
            modal.querySelector('.modal-close').addEventListener('click', () => {
                document.body.removeChild(modal);
            });
        } catch (error) {
            this.showError(`Failed to load model info: ${error.message}`);
        }
    }

    async testModel(modelId) {
        const inputData = prompt('Enter input data for testing (JSON format):');
        if (!inputData) return;
        
        try {
            const testData = JSON.parse(inputData);
            const response = await fetch(`${this.apiBase}/ml/models/${modelId}/predict`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ input_data: testData })
            });
            
            if (response.ok) {
                const result = await response.json();
                this.showSuccess(`Prediction: ${JSON.stringify(result.prediction)}`);
            } else {
                const error = await response.json();
                this.showError(`Prediction failed: ${error.error}`);
            }
        } catch (error) {
            this.showError(`Test failed: ${error.message}`);
        }
    }

    async stopModel(modelId) {
        if (!confirm('Are you sure you want to stop this model?')) return;
        
        try {
            const response = await fetch(`${this.apiBase}/ml/models/${modelId}`, {
                method: 'DELETE'
            });
            
            if (response.ok) {
                this.showSuccess('Model stopped successfully');
                this.loadModels();
            } else {
                const error = await response.json();
                this.showError(`Failed to stop model: ${error.error}`);
            }
        } catch (error) {
            this.showError(`Failed to stop model: ${error.message}`);
        }
    }

}

// Initialize the app when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.app = new CNETApp();
});
