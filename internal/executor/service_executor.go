package executor

import (
	"context"
)

// ServiceExecutor 服务型执行器：对外提供网络端点的长驻服务
// 说明：用于统一抽象 ML 推理服务、数据网关等“对外提供HTTP/gRPC端点”的工作负载
type ServiceExecutor interface {
	Executor
	// GetEndpoint 返回该workload对应的服务访问地址
	GetEndpoint(workloadID string) (string, error)
	// HealthCheck 主动进行一次健康检查
	HealthCheck(ctx context.Context, workloadID string) error
}
