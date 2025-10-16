package executor

// MLModelExecutor ML模型执行器接口
// 统一为服务型执行器，使用通用的 ServiceExecutor 契约
type MLModelExecutor interface {
	ServiceExecutor
}
