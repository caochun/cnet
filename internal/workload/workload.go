package workload

import (
	"cnet/internal/register"
	"time"
)

// WorkloadType workload类型
type WorkloadType string

const (
	TypeContainer WorkloadType = "container"
	TypeProcess   WorkloadType = "process"
	TypeMLModel   WorkloadType = "mlmodel"
	TypeOpenCV    WorkloadType = "opencv"
)

// WorkloadStatus workload状态
type WorkloadStatus string

const (
	StatusPending   WorkloadStatus = "pending"
	StatusRunning   WorkloadStatus = "running"
	StatusCompleted WorkloadStatus = "completed"
	StatusFailed    WorkloadStatus = "failed"
	StatusStopped   WorkloadStatus = "stopped"
)

// Workload workload接口
type Workload interface {
	// GetID 获取workload ID
	GetID() string

	// GetType 获取workload类型
	GetType() WorkloadType

	// GetResourceRequirements 获取资源需求
	GetResourceRequirements() register.ResourceRequirements

	// GetStatus 获取状态
	GetStatus() WorkloadStatus

	// SetStatus 设置状态
	SetStatus(status WorkloadStatus)

	// GetMetadata 获取元数据
	GetMetadata() map[string]interface{}

	// Validate 验证workload配置
	Validate() error
}

// BaseWorkload workload基础实现
type BaseWorkload struct {
	ID           string                        `json:"id"`
	Name         string                        `json:"name"`
	Type         WorkloadType                  `json:"type"`
	Status       WorkloadStatus                `json:"status"`
	Requirements register.ResourceRequirements `json:"requirements"`
	CreatedAt    time.Time                     `json:"created_at"`
	UpdatedAt    time.Time                     `json:"updated_at"`
	Metadata     map[string]interface{}        `json:"metadata"`
}

// GetID 实现Workload接口
func (w *BaseWorkload) GetID() string {
	return w.ID
}

// GetType 实现Workload接口
func (w *BaseWorkload) GetType() WorkloadType {
	return w.Type
}

// GetResourceRequirements 实现Workload接口
func (w *BaseWorkload) GetResourceRequirements() register.ResourceRequirements {
	return w.Requirements
}

// GetStatus 实现Workload接口
func (w *BaseWorkload) GetStatus() WorkloadStatus {
	return w.Status
}

// SetStatus 实现Workload接口
func (w *BaseWorkload) SetStatus(status WorkloadStatus) {
	w.Status = status
	w.UpdatedAt = time.Now()
}

// GetMetadata 实现Workload接口
func (w *BaseWorkload) GetMetadata() map[string]interface{} {
	return w.Metadata
}

// CreateWorkloadRequest 创建workload的请求
type CreateWorkloadRequest struct {
	Name         string                        `json:"name"`
	Type         WorkloadType                  `json:"type"`
	Requirements register.ResourceRequirements `json:"requirements"`
	Config       map[string]interface{}        `json:"config"`
}
