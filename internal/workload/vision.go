package workload

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// VisionWorkload 计算机视觉workload
type VisionWorkload struct {
	BaseWorkload
	Task         string            `json:"task"`              // 任务类型：detection, classification, segmentation, tracking
	InputPath    string            `json:"input_path"`        // 输入图片/视频路径
	OutputPath   string            `json:"output_path"`       // 输出路径
	ModelPath    string            `json:"model_path"`        // 模型文件路径
	ModelType    string            `json:"model_type"`        // 模型类型：yolo, cascade, dnn等
	Confidence   float32           `json:"confidence"`        // 置信度阈值（0.0-1.0）
	NMSThreshold float32           `json:"nms_threshold"`     // NMS阈值
	Config       map[string]string `json:"config"`            // 其他配置参数
	Results      interface{}       `json:"results,omitempty"` // 检测结果
}

// VisionTask 支持的视觉任务类型
const (
	TaskDetection      = "detection"      // 目标检测
	TaskClassification = "classification" // 图像分类
	TaskSegmentation   = "segmentation"   // 图像分割
	TaskTracking       = "tracking"       // 目标跟踪
	TaskFaceDetection  = "face_detection" // 人脸检测
	TaskOCR            = "ocr"            // 文字识别
)

// ModelType 支持的模型类型
const (
	ModelYOLO    = "yolo"    // YOLO系列
	ModelCascade = "cascade" // Haar/LBP级联分类器
	ModelDNN     = "dnn"     // OpenCV DNN模块
	ModelCustom  = "custom"  // 自定义模型
)

// NewVisionWorkload 创建Vision workload
func NewVisionWorkload(name, inputPath string, req CreateWorkloadRequest) *VisionWorkload {
	now := time.Now()

	workload := &VisionWorkload{
		BaseWorkload: BaseWorkload{
			ID:           uuid.New().String(),
			Name:         name,
			Type:         "vision",
			Status:       StatusPending,
			Requirements: req.Requirements,
			CreatedAt:    now,
			UpdatedAt:    now,
			Metadata:     req.Config,
		},
		InputPath:    inputPath,
		Confidence:   0.5, // 默认置信度
		NMSThreshold: 0.4, // 默认NMS阈值
		Config:       make(map[string]string),
	}

	// 从config中提取Vision特定配置
	if req.Config != nil {
		if task, ok := req.Config["task"].(string); ok {
			workload.Task = task
		}

		if outputPath, ok := req.Config["output_path"].(string); ok {
			workload.OutputPath = outputPath
		}

		if modelPath, ok := req.Config["model_path"].(string); ok {
			workload.ModelPath = modelPath
		}

		if modelType, ok := req.Config["model_type"].(string); ok {
			workload.ModelType = modelType
		}

		if conf, ok := req.Config["confidence"].(float64); ok {
			workload.Confidence = float32(conf)
		}

		if nms, ok := req.Config["nms_threshold"].(float64); ok {
			workload.NMSThreshold = float32(nms)
		}

		if config, ok := req.Config["vision_config"].(map[string]interface{}); ok {
			for k, v := range config {
				workload.Config[k] = fmt.Sprint(v)
			}
		}
	}

	return workload
}

// Validate 验证Vision workload配置
func (w *VisionWorkload) Validate() error {
	if w.InputPath == "" {
		return fmt.Errorf("input path cannot be empty")
	}

	if w.Task == "" {
		return fmt.Errorf("task type cannot be empty")
	}

	// 验证任务类型
	validTasks := map[string]bool{
		TaskDetection:      true,
		TaskClassification: true,
		TaskSegmentation:   true,
		TaskTracking:       true,
		TaskFaceDetection:  true,
		TaskOCR:            true,
	}
	if !validTasks[w.Task] {
		return fmt.Errorf("invalid task type: %s", w.Task)
	}

	// 验证模型类型（如果指定）
	if w.ModelType != "" {
		validModels := map[string]bool{
			ModelYOLO:    true,
			ModelCascade: true,
			ModelDNN:     true,
			ModelCustom:  true,
		}
		if !validModels[w.ModelType] {
			return fmt.Errorf("invalid model type: %s", w.ModelType)
		}
	}

	// 验证置信度
	if w.Confidence < 0 || w.Confidence > 1 {
		return fmt.Errorf("confidence must be between 0 and 1, got: %f", w.Confidence)
	}

	// 验证NMS阈值
	if w.NMSThreshold < 0 || w.NMSThreshold > 1 {
		return fmt.Errorf("nms_threshold must be between 0 and 1, got: %f", w.NMSThreshold)
	}

	if err := w.Requirements.Validate(); err != nil {
		return fmt.Errorf("invalid resource requirements: %w", err)
	}

	return nil
}
