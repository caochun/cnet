package executor

import (
	"context"
	"fmt"

	"cnet/internal/workload"

	"github.com/sirupsen/logrus"
)

// MLModelExecutorDispatcher ML模型执行器分发器
// 根据model_type选择对应的具体executor
type MLModelExecutorDispatcher struct {
	logger    *logrus.Logger
	executors map[string]MLModelExecutor // model_type -> executor
}

// NewMLModelExecutorDispatcher 创建ML模型执行器分发器
func NewMLModelExecutorDispatcher(logger *logrus.Logger) *MLModelExecutorDispatcher {
	dispatcher := &MLModelExecutorDispatcher{
		logger:    logger,
		executors: make(map[string]MLModelExecutor),
	}

	// 注册各种ML模型executor
	dispatcher.RegisterExecutor("yolo", NewYOLOExecutor(logger))
	// dispatcher.RegisterExecutor("tensorflow", NewTensorFlowExecutor(logger))
	// dispatcher.RegisterExecutor("pytorch", NewPyTorchExecutor(logger))

	return dispatcher
}

// RegisterExecutor 注册ML模型executor
func (d *MLModelExecutorDispatcher) RegisterExecutor(modelType string, executor MLModelExecutor) {
	d.executors[modelType] = executor
}

// Init 初始化所有ML模型executor
func (d *MLModelExecutorDispatcher) Init(ctx context.Context) error {
	for modelType, executor := range d.executors {
		if err := executor.Init(ctx); err != nil {
			return fmt.Errorf("failed to init %s executor: %w", modelType, err)
		}
	}
	d.logger.Info("ML Model Executor Dispatcher initialized")
	return nil
}

// Execute 执行MLModel workload - 根据model_type分发
func (d *MLModelExecutorDispatcher) Execute(ctx context.Context, w workload.Workload) error {
	mw, ok := w.(*workload.MLModelWorkload)
	if !ok {
		return fmt.Errorf("invalid workload type, expected MLModelWorkload")
	}

	executor, exists := d.executors[mw.ModelType]
	if !exists {
		return fmt.Errorf("no executor found for model type: %s", mw.ModelType)
	}

	return executor.Execute(ctx, w)
}

// Stop 停止MLModel workload
func (d *MLModelExecutorDispatcher) Stop(ctx context.Context, w workload.Workload) error {
	mw, ok := w.(*workload.MLModelWorkload)
	if !ok {
		return fmt.Errorf("invalid workload type, expected MLModelWorkload")
	}

	executor, exists := d.executors[mw.ModelType]
	if !exists {
		return fmt.Errorf("no executor found for model type: %s", mw.ModelType)
	}

	return executor.Stop(ctx, w)
}

// GetLogs 获取日志
func (d *MLModelExecutorDispatcher) GetLogs(ctx context.Context, w workload.Workload, lines int) ([]string, error) {
	mw, ok := w.(*workload.MLModelWorkload)
	if !ok {
		return nil, fmt.Errorf("invalid workload type, expected MLModelWorkload")
	}

	executor, exists := d.executors[mw.ModelType]
	if !exists {
		return nil, fmt.Errorf("no executor found for model type: %s", mw.ModelType)
	}

	return executor.GetLogs(ctx, w, lines)
}

// GetStatus 获取状态
func (d *MLModelExecutorDispatcher) GetStatus(ctx context.Context, w workload.Workload) (workload.WorkloadStatus, error) {
	mw, ok := w.(*workload.MLModelWorkload)
	if !ok {
		return "", fmt.Errorf("invalid workload type, expected MLModelWorkload")
	}

	executor, exists := d.executors[mw.ModelType]
	if !exists {
		return "", fmt.Errorf("no executor found for model type: %s", mw.ModelType)
	}

	return executor.GetStatus(ctx, w)
}
