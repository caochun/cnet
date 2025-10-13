package executor

import (
	"cnet/internal/workload"
	"context"
)

// Executor 执行器接口
type Executor interface {
	// Execute 执行workload
	Execute(ctx context.Context, w workload.Workload) error

	// Stop 停止workload
	Stop(ctx context.Context, w workload.Workload) error

	// GetLogs 获取workload日志
	GetLogs(ctx context.Context, w workload.Workload, lines int) ([]string, error)

	// GetStatus 获取workload状态
	GetStatus(ctx context.Context, w workload.Workload) (workload.WorkloadStatus, error)
}

// ExecutorFactory 执行器工厂
type ExecutorFactory struct {
	executors map[workload.WorkloadType]Executor
}

// NewExecutorFactory 创建执行器工厂
func NewExecutorFactory() *ExecutorFactory {
	return &ExecutorFactory{
		executors: make(map[workload.WorkloadType]Executor),
	}
}

// Register 注册执行器
func (f *ExecutorFactory) Register(wType workload.WorkloadType, executor Executor) {
	f.executors[wType] = executor
}

// GetExecutor 获取执行器
func (f *ExecutorFactory) GetExecutor(wType workload.WorkloadType) (Executor, bool) {
	executor, ok := f.executors[wType]
	return executor, ok
}
