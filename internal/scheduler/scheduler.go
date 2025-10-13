package scheduler

import (
	"context"
	"fmt"
	"sync"

	"cnet/internal/executor"
	"cnet/internal/register"
	"cnet/internal/workload"

	"github.com/sirupsen/logrus"
)

// ScheduleDecision 调度决策
type ScheduleDecision struct {
	Action   string `json:"action"` // "local", "delegate_child", "delegate_peer"
	NodeID   string `json:"node_id"`
	NodeAddr string `json:"node_addr"`
	Reason   string `json:"reason"`
}

// Scheduler 调度器
type Scheduler struct {
	logger   *logrus.Logger
	register *register.Register
	factory  *executor.ExecutorFactory
	strategy ScheduleStrategy
	mu       sync.RWMutex

	// 正在运行的workload
	runningWorkloads map[string]workload.Workload

	// workload到allocation的映射
	workloadAllocations map[string]string
}

// NewScheduler 创建调度器
func NewScheduler(logger *logrus.Logger, reg *register.Register, factory *executor.ExecutorFactory) *Scheduler {
	return &Scheduler{
		logger:              logger,
		register:            reg,
		factory:             factory,
		strategy:            &DefaultScheduleStrategy{},
		runningWorkloads:    make(map[string]workload.Workload),
		workloadAllocations: make(map[string]string),
	}
}

// SetStrategy 设置调度策略
func (s *Scheduler) SetStrategy(strategy ScheduleStrategy) {
	s.strategy = strategy
}

// Schedule 调度workload
func (s *Scheduler) Schedule(ctx context.Context, w workload.Workload) (*ScheduleDecision, error) {
	// 验证workload
	if err := w.Validate(); err != nil {
		return nil, fmt.Errorf("workload validation failed: %w", err)
	}

	req := w.GetResourceRequirements()

	// 使用策略做出调度决策
	decision := s.strategy.MakeDecision(s.register, req)

	s.logger.WithFields(logrus.Fields{
		"workload_id": w.GetID(),
		"action":      decision.Action,
		"node_id":     decision.NodeID,
		"reason":      decision.Reason,
	}).Info("Schedule decision made")

	return decision, nil
}

// ExecuteLocal 在本地执行workload
func (s *Scheduler) ExecuteLocal(ctx context.Context, w workload.Workload) error {
	// 分配资源
	allocation, err := s.register.AllocateResources(w.GetID(), w.GetResourceRequirements())
	if err != nil {
		return fmt.Errorf("failed to allocate resources: %w", err)
	}

	// 获取对应的执行器
	exec, ok := s.factory.GetExecutor(w.GetType())
	if !ok {
		// 释放资源
		s.register.ReleaseResources(allocation.ID)
		return fmt.Errorf("no executor found for workload type: %s", w.GetType())
	}

	// 执行workload
	if err := exec.Execute(ctx, w); err != nil {
		// 释放资源
		s.register.ReleaseResources(allocation.ID)
		return fmt.Errorf("failed to execute workload: %w", err)
	}

	// 记录workload和allocation
	s.mu.Lock()
	s.runningWorkloads[w.GetID()] = w
	s.workloadAllocations[w.GetID()] = allocation.ID
	s.mu.Unlock()

	s.logger.WithFields(logrus.Fields{
		"workload_id":   w.GetID(),
		"allocation_id": allocation.ID,
		"type":          w.GetType(),
	}).Info("Workload executing locally")

	return nil
}

// DelegateToChild 委托给下级节点
func (s *Scheduler) DelegateToChild(ctx context.Context, w workload.Workload, nodeID string) error {
	// TODO: 实现向下级节点委托的逻辑
	// 1. 找到目标节点的地址
	// 2. 通过HTTP/gRPC发送workload
	// 3. 等待确认

	s.logger.WithFields(logrus.Fields{
		"workload_id": w.GetID(),
		"target_node": nodeID,
	}).Info("Delegating workload to child node (not implemented)")

	return fmt.Errorf("delegation to child not implemented yet")
}

// DelegateToPeer 委托给同级节点
func (s *Scheduler) DelegateToPeer(ctx context.Context, w workload.Workload, nodeID string) error {
	// TODO: 实现向同级节点委托的逻辑
	// 1. 找到目标节点的地址
	// 2. 通过HTTP/gRPC发送workload
	// 3. 等待确认

	s.logger.WithFields(logrus.Fields{
		"workload_id": w.GetID(),
		"target_node": nodeID,
	}).Info("Delegating workload to peer node (not implemented)")

	return fmt.Errorf("delegation to peer not implemented yet")
}

// StopWorkload 停止workload
func (s *Scheduler) StopWorkload(ctx context.Context, workloadID string) error {
	s.mu.RLock()
	w, exists := s.runningWorkloads[workloadID]
	allocationID := s.workloadAllocations[workloadID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("workload not found: %s", workloadID)
	}

	// 获取执行器
	exec, ok := s.factory.GetExecutor(w.GetType())
	if !ok {
		return fmt.Errorf("no executor found for workload type: %s", w.GetType())
	}

	// 停止workload
	if err := exec.Stop(ctx, w); err != nil {
		return fmt.Errorf("failed to stop workload: %w", err)
	}

	// 释放资源
	if err := s.register.ReleaseResources(allocationID); err != nil {
		s.logger.WithError(err).Warn("Failed to release resources")
	}

	// 移除记录
	s.mu.Lock()
	delete(s.runningWorkloads, workloadID)
	delete(s.workloadAllocations, workloadID)
	s.mu.Unlock()

	s.logger.WithField("workload_id", workloadID).Info("Workload stopped")

	return nil
}

// GetWorkload 获取workload信息
func (s *Scheduler) GetWorkload(workloadID string) (workload.Workload, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	w, exists := s.runningWorkloads[workloadID]
	if !exists {
		return nil, fmt.Errorf("workload not found: %s", workloadID)
	}

	return w, nil
}

// ListWorkloads 列出所有workload
func (s *Scheduler) ListWorkloads() []workload.Workload {
	s.mu.RLock()
	defer s.mu.RUnlock()

	workloads := make([]workload.Workload, 0, len(s.runningWorkloads))
	for _, w := range s.runningWorkloads {
		workloads = append(workloads, w)
	}

	return workloads
}

// GetWorkloadLogs 获取workload日志
func (s *Scheduler) GetWorkloadLogs(ctx context.Context, workloadID string, lines int) ([]string, error) {
	s.mu.RLock()
	w, exists := s.runningWorkloads[workloadID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("workload not found: %s", workloadID)
	}

	// 获取执行器
	exec, ok := s.factory.GetExecutor(w.GetType())
	if !ok {
		return nil, fmt.Errorf("no executor found for workload type: %s", w.GetType())
	}

	return exec.GetLogs(ctx, w, lines)
}
