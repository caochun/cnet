package executor

import (
	"context"
)

// MLModelExecutor ML模型执行器接口
// 定义在agent上部署和运行机器学习模型的通用行为
type MLModelExecutor interface {
	// 继承基础Executor接口
	Executor

	// GetInferenceEndpoint 获取推理服务的endpoint
	GetInferenceEndpoint(workloadID string) (string, error)

	// HealthCheck 检查推理服务健康状态
	HealthCheck(ctx context.Context, workloadID string) error
}
