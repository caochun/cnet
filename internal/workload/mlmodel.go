package workload

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// MLModelWorkload ML模型workload
type MLModelWorkload struct {
	BaseWorkload
	
	// 模型标识
	ModelType   string `json:"model_type"`   // "yolo", "tensorflow", "pytorch"
	ModelPath   string `json:"model_path"`   // 模型文件路径
	ModelConfig string `json:"model_config"` // 可选配置文件路径
	Framework   string `json:"framework"`    // "yolov8", "yolov5", "tf2"等
	
	// 服务配置
	ServicePort int    `json:"service_port"` // HTTP服务端口
	ServiceHost string `json:"service_host"` // 服务host（默认localhost）
	
	// 运行时信息
	Endpoint   string `json:"endpoint,omitempty"`    // 推理服务endpoint
	ProcessPID int    `json:"process_pid,omitempty"` // HTTP server进程PID
}

// NewMLModelWorkload 创建ML模型workload
func NewMLModelWorkload(name string, req CreateWorkloadRequest) *MLModelWorkload {
	now := time.Now()

	workload := &MLModelWorkload{
		BaseWorkload: BaseWorkload{
			ID:           uuid.New().String(),
			Name:         name,
			Type:         TypeMLModel,
			Status:       StatusPending,
			Requirements: req.Requirements,
			CreatedAt:    now,
			UpdatedAt:    now,
			Metadata:     req.Config,
		},
		ServicePort: 9000, // 默认端口
		ServiceHost: "localhost",
	}

	// 从config中提取ML模型特定配置
	if req.Config != nil {
		if modelType, ok := req.Config["model_type"].(string); ok {
			workload.ModelType = modelType
		}
		
		if modelPath, ok := req.Config["model_path"].(string); ok {
			workload.ModelPath = modelPath
		}
		
		if modelConfig, ok := req.Config["model_config"].(string); ok {
			workload.ModelConfig = modelConfig
		}

		if framework, ok := req.Config["framework"].(string); ok {
			workload.Framework = framework
		}

		if port, ok := req.Config["service_port"].(float64); ok {
			workload.ServicePort = int(port)
		}
		
		if host, ok := req.Config["service_host"].(string); ok {
			workload.ServiceHost = host
		}
	}

	return workload
}

// Validate 验证ML模型workload配置
func (w *MLModelWorkload) Validate() error {
	if w.ModelPath == "" {
		return fmt.Errorf("model path cannot be empty")
	}
	
	if w.ModelType == "" {
		return fmt.Errorf("model type cannot be empty")
	}

	if err := w.Requirements.Validate(); err != nil {
		return fmt.Errorf("invalid resource requirements: %w", err)
	}

	// 验证模型类型
	validTypes := map[string]bool{
		"yolo":       true,
		"tensorflow": true,
		"pytorch":    true,
	}
	if !validTypes[w.ModelType] {
		return fmt.Errorf("invalid model type: %s", w.ModelType)
	}

	// 验证端口
	if w.ServicePort <= 0 || w.ServicePort > 65535 {
		return fmt.Errorf("invalid service port: %d", w.ServicePort)
	}

	return nil
}
