package workload

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// OpenCVWorkload OpenCV推理服务workload
type OpenCVWorkload struct {
	BaseWorkload

	// 服务配置
	ServicePort int    `json:"service_port"` // HTTP服务端口
	ServiceHost string `json:"service_host"` // 服务host（默认localhost）

	// OpenCV配置
	CascadeType string `json:"cascade_type"` // "face", "eye", "smile"
	CascadePath string `json:"cascade_path"` // Haar Cascade模型路径

	// 运行时信息
	Endpoint   string `json:"endpoint,omitempty"`    // 推理服务endpoint
	ProcessPID int    `json:"process_pid,omitempty"` // HTTP server进程PID
}

// CascadeType 支持的Cascade类型
const (
	CascadeTypeFace  = "face"  // 人脸检测
	CascadeTypeEye   = "eye"   // 眼睛检测
	CascadeTypeSmile = "smile" // 笑脸检测
)

// NewOpenCVWorkload 创建OpenCV workload
func NewOpenCVWorkload(name string, req CreateWorkloadRequest) *OpenCVWorkload {
	now := time.Now()

	workload := &OpenCVWorkload{
		BaseWorkload: BaseWorkload{
			ID:           uuid.New().String(),
			Name:         name,
			Type:         TypeOpenCV,
			Status:       StatusPending,
			Requirements: req.Requirements,
			CreatedAt:    now,
			UpdatedAt:    now,
			Metadata:     req.Config,
		},
		ServicePort: 9000, // 默认端口
		ServiceHost: "localhost",
		CascadeType: CascadeTypeFace, // 默认人脸检测
	}

	// 从config中提取OpenCV特定配置
	if req.Config != nil {
		if port, ok := req.Config["service_port"].(float64); ok {
			workload.ServicePort = int(port)
		}

		if host, ok := req.Config["service_host"].(string); ok {
			workload.ServiceHost = host
		}

		if cascadeType, ok := req.Config["cascade_type"].(string); ok {
			workload.CascadeType = cascadeType
		}

		if cascadePath, ok := req.Config["cascade_path"].(string); ok {
			workload.CascadePath = cascadePath
		}
	}

	return workload
}

// Validate 验证OpenCV workload配置
func (w *OpenCVWorkload) Validate() error {
	if w.ServicePort <= 0 || w.ServicePort > 65535 {
		return fmt.Errorf("invalid service port: %d", w.ServicePort)
	}

	// 验证cascade类型
	validCascades := map[string]bool{
		CascadeTypeFace:  true,
		CascadeTypeEye:   true,
		CascadeTypeSmile: true,
	}
	if !validCascades[w.CascadeType] {
		return fmt.Errorf("invalid cascade type: %s", w.CascadeType)
	}

	if err := w.Requirements.Validate(); err != nil {
		return fmt.Errorf("invalid resource requirements: %w", err)
	}

	return nil
}
