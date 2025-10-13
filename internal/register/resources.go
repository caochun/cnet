package register

import (
	"fmt"
	"time"
)

// ResourceType 资源类型
type ResourceType string

const (
	ResourceTypeCPU     ResourceType = "cpu"
	ResourceTypeGPU     ResourceType = "gpu"
	ResourceTypeMemory  ResourceType = "memory"
	ResourceTypeStorage ResourceType = "storage"
)

// Resources 资源定义
type Resources struct {
	CPU     float64 `json:"cpu"`     // CPU核心数
	GPU     int     `json:"gpu"`     // GPU数量
	Memory  int64   `json:"memory"`  // 内存（字节）
	Storage int64   `json:"storage"` // 存储空间（字节）
}

// ResourceRequirements workload的资源需求
type ResourceRequirements struct {
	CPU     float64 `json:"cpu"`
	GPU     int     `json:"gpu"`
	Memory  int64   `json:"memory"`
	Storage int64   `json:"storage"`
}

// NodeResources 节点资源信息
type NodeResources struct {
	NodeID      string            `json:"node_id"`
	NodeType    string            `json:"node_type"` // "local", "child", "peer"
	Total       Resources         `json:"total"`
	Available   Resources         `json:"available"`
	Used        Resources         `json:"used"`
	LastUpdated time.Time         `json:"last_updated"`
	Status      string            `json:"status"` // "active", "inactive", "unreachable"
	Address     string            `json:"address"`
	Metadata    map[string]string `json:"metadata"`
}

// Clone 克隆资源
func (r *Resources) Clone() Resources {
	return Resources{
		CPU:     r.CPU,
		GPU:     r.GPU,
		Memory:  r.Memory,
		Storage: r.Storage,
	}
}

// Sub 资源减法
func (r *Resources) Sub(req ResourceRequirements) Resources {
	return Resources{
		CPU:     r.CPU - req.CPU,
		GPU:     r.GPU - req.GPU,
		Memory:  r.Memory - req.Memory,
		Storage: r.Storage - req.Storage,
	}
}

// Add 资源加法
func (r *Resources) Add(req ResourceRequirements) Resources {
	return Resources{
		CPU:     r.CPU + req.CPU,
		GPU:     r.GPU + req.GPU,
		Memory:  r.Memory + req.Memory,
		Storage: r.Storage + req.Storage,
	}
}

// CanSatisfy 检查资源是否满足需求
func (r *Resources) CanSatisfy(req ResourceRequirements) bool {
	return r.CPU >= req.CPU &&
		r.GPU >= req.GPU &&
		r.Memory >= req.Memory &&
		r.Storage >= req.Storage
}

// Validate 验证资源需求
func (req *ResourceRequirements) Validate() error {
	if req.CPU < 0 {
		return fmt.Errorf("CPU requirement cannot be negative: %f", req.CPU)
	}
	if req.GPU < 0 {
		return fmt.Errorf("GPU requirement cannot be negative: %d", req.GPU)
	}
	if req.Memory < 0 {
		return fmt.Errorf("Memory requirement cannot be negative: %d", req.Memory)
	}
	if req.Storage < 0 {
		return fmt.Errorf("Storage requirement cannot be negative: %d", req.Storage)
	}
	return nil
}

// String 资源字符串表示
func (r *Resources) String() string {
	return fmt.Sprintf("CPU: %.2f, GPU: %d, Memory: %dMB, Storage: %dGB",
		r.CPU, r.GPU, r.Memory/(1024*1024), r.Storage/(1024*1024*1024))
}
