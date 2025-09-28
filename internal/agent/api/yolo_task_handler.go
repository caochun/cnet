package api

import (
	"encoding/json"
	"net/http"
	"time"

	"cnet/internal/agent/tasks"

	"github.com/gorilla/mux"
)

// YOLOTaskHandler handles YOLO-specific task operations
type YOLOTaskHandler struct {
	*BaseTaskHandler
}

// NewYOLOTaskHandler creates a new YOLO task handler
func NewYOLOTaskHandler(server *Server) *YOLOTaskHandler {
	return &YOLOTaskHandler{
		BaseTaskHandler: NewBaseTaskHandler(server),
	}
}

// YOLOInferenceRequest represents a YOLO inference request
type YOLOInferenceRequest struct {
	ImagePath string                 `json:"image_path,omitempty"`
	ImageData string                 `json:"image_data,omitempty"` // base64 encoded image
	ImageURL  string                 `json:"image_url,omitempty"`
	Config    YOLOConfig             `json:"config,omitempty"`
	Options   map[string]interface{} `json:"options,omitempty"`
}

// YOLOConfig represents YOLO model configuration
type YOLOConfig struct {
	ModelPath      string                 `json:"model_path"`
	ModelType      string                 `json:"model_type"`     // yolov5, yolov8, etc.
	Confidence     float64                `json:"confidence"`     // confidence threshold
	IOUThreshold   float64                `json:"iou_threshold"`  // IoU threshold for NMS
	ImageSize      int                    `json:"image_size"`     // input image size
	MaxDetections  int                    `json:"max_detections"` // maximum number of detections
	Classes        []string               `json:"classes"`        // class names
	Preprocessing  map[string]interface{} `json:"preprocessing"`
	Postprocessing map[string]interface{} `json:"postprocessing"`
}

// YOLODetection represents a single detection result
type YOLODetection struct {
	ClassID     int     `json:"class_id"`
	ClassName   string  `json:"class_name"`
	Confidence  float64 `json:"confidence"`
	BoundingBox struct {
		X      float64 `json:"x"`      // center x
		Y      float64 `json:"y"`      // center y
		Width  float64 `json:"width"`  // width
		Height float64 `json:"height"` // height
	} `json:"bounding_box"`
}

// YOLOInferenceResponse represents a YOLO inference response
type YOLOInferenceResponse struct {
	Detections     []YOLODetection        `json:"detections"`
	ImageInfo      map[string]interface{} `json:"image_info"`
	ProcessingTime float64                `json:"processing_time"`
	ModelInfo      map[string]interface{} `json:"model_info"`
	Timestamp      time.Time              `json:"timestamp"`
}

// ListTasks handles YOLO task listing requests (filtered by type=yolo)
func (h *YOLOTaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	// Get all tasks and filter for YOLO tasks
	allTasks, err := h.server.tasks.ListTasks()
	if err != nil {
		h.server.writeError(w, http.StatusInternalServerError, "Failed to list tasks", err)
		return
	}

	// Filter for YOLO tasks
	var yoloTasks []*tasks.Task
	for _, task := range allTasks {
		if task.Type == "yolo" {
			yoloTasks = append(yoloTasks, task)
		}
	}

	h.server.writeJSON(w, http.StatusOK, yoloTasks)
}

// CreateTask handles YOLO task creation requests
func (h *YOLOTaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name       string               `json:"name"`
		ModelPath  string               `json:"model_path"`
		ScriptPath string               `json:"script_path"`
		Config     YOLOConfig           `json:"config"`
		Resources  tasks.ResourceLimits `json:"resources"`
		Env        map[string]string    `json:"env"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.server.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Convert YOLO task request to task request
	taskReq := &tasks.CreateTaskRequest{
		Name:       req.Name,
		Type:       "yolo",
		Command:    "python3", // YOLO tasks use Python
		Args:       []string{req.ScriptPath, req.ModelPath},
		Env:        req.Env,
		WorkingDir: "", // YOLO tasks don't have a specific working directory
		Resources:  req.Resources,
	}

	task, err := h.server.tasks.CreateTask(taskReq)
	if err != nil {
		h.server.writeError(w, http.StatusInternalServerError, "Failed to create YOLO task", err)
		return
	}

	h.server.writeJSON(w, http.StatusCreated, task)
}

// GetTask handles YOLO task retrieval requests
func (h *YOLOTaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "YOLO task not found", err)
		return
	}

	// Verify this is a YOLO task
	if task.Type != "yolo" {
		h.server.writeError(w, http.StatusBadRequest, "Task is not a YOLO task", nil)
		return
	}

	h.server.writeJSON(w, http.StatusOK, task)
}

// StopTask handles YOLO task stopping requests
func (h *YOLOTaskHandler) StopTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Verify this is a YOLO task
	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "YOLO task not found", err)
		return
	}

	if task.Type != "yolo" {
		h.server.writeError(w, http.StatusBadRequest, "Task is not a YOLO task", nil)
		return
	}

	if err := h.server.tasks.StopTask(taskID); err != nil {
		h.server.writeError(w, http.StatusInternalServerError, "Failed to stop YOLO task", err)
		return
	}

	h.server.writeJSON(w, http.StatusOK, map[string]string{"message": "YOLO task stopped"})
}

// GetTaskLogs handles YOLO task log requests
func (h *YOLOTaskHandler) GetTaskLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Verify this is a YOLO task
	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "YOLO task not found", err)
		return
	}

	if task.Type != "yolo" {
		h.server.writeError(w, http.StatusBadRequest, "Task is not a YOLO task", nil)
		return
	}

	// Use base handler for logs
	h.BaseTaskHandler.GetTaskLogs(w, r)
}

// GetTaskInfo handles YOLO task information requests with YOLO-specific details
func (h *YOLOTaskHandler) GetTaskInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Verify this is a YOLO task
	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "YOLO task not found", err)
		return
	}

	if task.Type != "yolo" {
		h.server.writeError(w, http.StatusBadRequest, "Task is not a YOLO task", nil)
		return
	}

	// Return task info with YOLO-specific details
	info := map[string]interface{}{
		"task_id":     task.ID,
		"name":        task.Name,
		"type":        task.Type,
		"status":      task.Status,
		"created_at":  task.CreatedAt,
		"started_at":  task.StartedAt,
		"stopped_at":  task.StoppedAt,
		"exit_code":   task.ExitCode,
		"resources":   task.Resources,
		"env":         task.Env,
		"working_dir": task.WorkingDir,
		"yolo_specific": map[string]interface{}{
			"model_path": func() string {
				if len(task.Args) > 1 {
					return task.Args[1]
				} else {
					return ""
				}
			}(),
			"script_path": func() string {
				if len(task.Args) > 0 {
					return task.Args[0]
				} else {
					return ""
				}
			}(),
		},
	}

	h.server.writeJSON(w, http.StatusOK, info)
}

// GetTaskHealth handles YOLO task health check requests
func (h *YOLOTaskHandler) GetTaskHealth(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Verify this is a YOLO task
	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "YOLO task not found", err)
		return
	}

	if task.Type != "yolo" {
		h.server.writeError(w, http.StatusBadRequest, "Task is not a YOLO task", nil)
		return
	}

	// Check task health
	if task.Status != "running" {
		h.server.writeError(w, http.StatusServiceUnavailable, "YOLO task is not running", nil)
		return
	}

	response := map[string]interface{}{
		"status":    "healthy",
		"task_id":   taskID,
		"timestamp": time.Now().Format(time.RFC3339),
		"yolo_specific": map[string]interface{}{
			"inference_ready": true,
			"model_loaded":    true,
		},
	}

	h.server.writeJSON(w, http.StatusOK, response)
}

// Predict handles YOLO task prediction requests (YOLO-specific operation)
func (h *YOLOTaskHandler) Predict(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Verify this is a YOLO task
	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "YOLO task not found", err)
		return
	}

	if task.Type != "yolo" {
		h.server.writeError(w, http.StatusBadRequest, "Task is not a YOLO task", nil)
		return
	}

	var req YOLOInferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.server.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// TODO: Implement actual YOLO inference
	// This would make HTTP requests to the YOLO inference server
	response := YOLOInferenceResponse{
		Detections: []YOLODetection{
			{
				ClassID:    0,
				ClassName:  "person",
				Confidence: 0.95,
				BoundingBox: struct {
					X      float64 `json:"x"`
					Y      float64 `json:"y"`
					Width  float64 `json:"width"`
					Height float64 `json:"height"`
				}{
					X:      100.0,
					Y:      150.0,
					Width:  80.0,
					Height: 200.0,
				},
			},
		},
		ImageInfo: map[string]interface{}{
			"width":  640,
			"height": 480,
		},
		ProcessingTime: 0.1,
		ModelInfo: map[string]interface{}{
			"model_type": "yolov8",
		},
		Timestamp: time.Now(),
	}

	h.server.writeJSON(w, http.StatusOK, response)
}

// GetModelInfo handles YOLO model information requests
func (h *YOLOTaskHandler) GetModelInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Verify this is a YOLO task
	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "YOLO task not found", err)
		return
	}

	if task.Type != "yolo" {
		h.server.writeError(w, http.StatusBadRequest, "Task is not a YOLO task", nil)
		return
	}

	// Return YOLO model information
	modelInfo := map[string]interface{}{
		"model_type": "yolov8",
		"classes": []string{
			"person", "bicycle", "car", "motorcycle", "airplane", "bus", "train", "truck",
			"boat", "traffic light", "fire hydrant", "stop sign", "parking meter", "bench",
			"bird", "cat", "dog", "horse", "sheep", "cow", "elephant", "bear", "zebra",
			"giraffe", "backpack", "umbrella", "handbag", "tie", "suitcase", "frisbee",
		},
		"input_size":           640,
		"confidence_threshold": 0.5,
		"iou_threshold":        0.45,
		"max_detections":       100,
	}

	h.server.writeJSON(w, http.StatusOK, modelInfo)
}
