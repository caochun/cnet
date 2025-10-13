package workload

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// MLModelWorkload ML模型workload
type MLModelWorkload struct {
	BaseWorkload
	ModelPath  string            `json:"model_path"`  // 模型文件路径
	ModelType  string            `json:"model_type"`  // 模型类型（tensorflow, pytorch, onnx等）
	Framework  string            `json:"framework"`   // 推理框架
	ScriptPath string            `json:"script_path"` // 推理脚本路径
	Config     map[string]string `json:"config"`      // 模型配置
	Endpoint   string            `json:"endpoint"`    // 推理服务端点
	Port       int               `json:"port"`        // 服务端口
}

// NewMLModelWorkload 创建ML模型workload
func NewMLModelWorkload(name, modelPath string, req CreateWorkloadRequest) *MLModelWorkload {
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
		ModelPath: modelPath,
		Config:    make(map[string]string),
	}

	// 从config中提取ML模型特定配置
	if req.Config != nil {
		if modelType, ok := req.Config["model_type"].(string); ok {
			workload.ModelType = modelType
		}

		if framework, ok := req.Config["framework"].(string); ok {
			workload.Framework = framework
		}

		if scriptPath, ok := req.Config["script_path"].(string); ok {
			workload.ScriptPath = scriptPath
		}

		if port, ok := req.Config["port"].(float64); ok {
			workload.Port = int(port)
		}

		if config, ok := req.Config["model_config"].(map[string]interface{}); ok {
			for k, v := range config {
				workload.Config[k] = fmt.Sprint(v)
			}
		}
	}

	return workload
}

// Validate 验证ML模型workload配置
func (w *MLModelWorkload) Validate() error {
	if w.ModelPath == "" {
		return fmt.Errorf("model path cannot be empty")
	}

	if err := w.Requirements.Validate(); err != nil {
		return fmt.Errorf("invalid resource requirements: %w", err)
	}

	// 验证模型类型
	validTypes := map[string]bool{
		"tensorflow": true,
		"pytorch":    true,
		"onnx":       true,
		"sklearn":    true,
	}
	if w.ModelType != "" && !validTypes[w.ModelType] {
		return fmt.Errorf("invalid model type: %s", w.ModelType)
	}

	// 验证端口
	if w.Port != 0 && (w.Port <= 0 || w.Port > 65535) {
		return fmt.Errorf("invalid port: %d", w.Port)
	}

	return nil
}
