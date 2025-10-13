package manager

import (
	"context"
	"fmt"
	"sync"

	"cnet/internal/register"
	"cnet/internal/scheduler"
	"cnet/internal/workload"

	"github.com/sirupsen/logrus"
)

// Manager 管理器
type Manager struct {
	logger    *logrus.Logger
	scheduler *scheduler.Scheduler
	register  *register.Register
	mu        sync.RWMutex

	// workload存储
	workloads map[string]workload.Workload
}

// NewManager 创建管理器
func NewManager(logger *logrus.Logger, sched *scheduler.Scheduler, reg *register.Register) *Manager {
	return &Manager{
		logger:    logger,
		scheduler: sched,
		register:  reg,
		workloads: make(map[string]workload.Workload),
	}
}

// SubmitWorkload 提交workload
func (m *Manager) SubmitWorkload(ctx context.Context, req *workload.CreateWorkloadRequest) (workload.Workload, error) {
	// 验证请求
	if req.Name == "" {
		return nil, fmt.Errorf("workload name cannot be empty")
	}

	if err := req.Requirements.Validate(); err != nil {
		return nil, fmt.Errorf("invalid resource requirements: %w", err)
	}

	// 根据类型创建workload
	var w workload.Workload
	var err error

	switch req.Type {
	case workload.TypeProcess:
		command, ok := req.Config["command"].(string)
		if !ok {
			return nil, fmt.Errorf("command is required for process workload")
		}
		w = workload.NewProcessWorkload(req.Name, command, *req)

	case workload.TypeContainer:
		image, ok := req.Config["image"].(string)
		if !ok {
			return nil, fmt.Errorf("image is required for container workload")
		}
		w = workload.NewContainerWorkload(req.Name, image, *req)

	case workload.TypeMLModel:
		modelPath, ok := req.Config["model_path"].(string)
		if !ok {
			return nil, fmt.Errorf("model_path is required for ML model workload")
		}
		w = workload.NewMLModelWorkload(req.Name, modelPath, *req)

	case workload.TypeVision:
		inputPath, ok := req.Config["input_path"].(string)
		if !ok {
			return nil, fmt.Errorf("input_path is required for vision workload")
		}
		w = workload.NewVisionWorkload(req.Name, inputPath, *req)

	default:
		return nil, fmt.Errorf("unsupported workload type: %s", req.Type)
	}

	// 验证workload
	if err = w.Validate(); err != nil {
		return nil, fmt.Errorf("workload validation failed: %w", err)
	}

	// 保存workload
	m.mu.Lock()
	m.workloads[w.GetID()] = w
	m.mu.Unlock()

	m.logger.WithFields(logrus.Fields{
		"workload_id": w.GetID(),
		"name":        req.Name,
		"type":        req.Type,
	}).Info("Workload submitted")

	// 调度workload
	decision, err := m.scheduler.Schedule(ctx, w)
	if err != nil {
		return nil, fmt.Errorf("scheduling failed: %w", err)
	}

	// 根据调度决策执行
	switch decision.Action {
	case "local":
		err = m.scheduler.ExecuteLocal(ctx, w)
	case "delegate_child":
		err = m.scheduler.DelegateToChild(ctx, w, decision.NodeID)
	case "delegate_peer":
		err = m.scheduler.DelegateToPeer(ctx, w, decision.NodeID)
	case "reject":
		err = fmt.Errorf("workload rejected: %s", decision.Reason)
	default:
		err = fmt.Errorf("unknown schedule action: %s", decision.Action)
	}

	if err != nil {
		return nil, fmt.Errorf("workload execution failed: %w", err)
	}

	return w, nil
}

// GetWorkload 获取workload
func (m *Manager) GetWorkload(workloadID string) (workload.Workload, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	w, exists := m.workloads[workloadID]
	if !exists {
		return nil, fmt.Errorf("workload not found: %s", workloadID)
	}

	return w, nil
}

// ListWorkloads 列出所有workload
func (m *Manager) ListWorkloads() []workload.Workload {
	m.mu.RLock()
	defer m.mu.RUnlock()

	workloads := make([]workload.Workload, 0, len(m.workloads))
	for _, w := range m.workloads {
		workloads = append(workloads, w)
	}

	return workloads
}

// StopWorkload 停止workload
func (m *Manager) StopWorkload(ctx context.Context, workloadID string) error {
	// 检查workload是否存在
	m.mu.RLock()
	_, exists := m.workloads[workloadID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("workload not found: %s", workloadID)
	}

	// 调用scheduler停止workload
	if err := m.scheduler.StopWorkload(ctx, workloadID); err != nil {
		return fmt.Errorf("failed to stop workload: %w", err)
	}

	m.logger.WithField("workload_id", workloadID).Info("Workload stopped")

	return nil
}

// DeleteWorkload 删除workload
func (m *Manager) DeleteWorkload(ctx context.Context, workloadID string) error {
	// 先停止workload
	if err := m.StopWorkload(ctx, workloadID); err != nil {
		// 如果已经停止，忽略错误
		m.logger.WithError(err).Debug("Error stopping workload during deletion")
	}

	// 删除workload记录
	m.mu.Lock()
	delete(m.workloads, workloadID)
	m.mu.Unlock()

	m.logger.WithField("workload_id", workloadID).Info("Workload deleted")

	return nil
}

// GetWorkloadLogs 获取workload日志
func (m *Manager) GetWorkloadLogs(ctx context.Context, workloadID string, lines int) ([]string, error) {
	return m.scheduler.GetWorkloadLogs(ctx, workloadID, lines)
}

// GetResourceStats 获取资源统计信息
func (m *Manager) GetResourceStats() map[string]interface{} {
	localRes := m.register.GetLocalResources()
	childNodes := m.register.GetChildNodes()
	peerNodes := m.register.GetPeerNodes()
	workloads := m.ListWorkloads()

	stats := map[string]interface{}{
		"local_resources": map[string]interface{}{
			"total":     localRes.Total,
			"available": localRes.Available,
			"used":      localRes.Used,
		},
		"child_nodes_count": len(childNodes),
		"peer_nodes_count":  len(peerNodes),
		"workloads_count":   len(workloads),
	}

	// 统计workload状态
	statusCount := make(map[workload.WorkloadStatus]int)
	for _, w := range workloads {
		statusCount[w.GetStatus()]++
	}
	stats["workloads_by_status"] = statusCount

	return stats
}
