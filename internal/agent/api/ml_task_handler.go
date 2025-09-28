package api

import (
	"encoding/json"
	"net/http"
	"time"

	"cnet/internal/agent/ml"
	"cnet/internal/agent/tasks"

	"github.com/gorilla/mux"
)

// MLTaskHandler handles ML-specific task operations
type MLTaskHandler struct {
	*BaseTaskHandler
}

// NewMLTaskHandler creates a new ML task handler
func NewMLTaskHandler(server *Server) *MLTaskHandler {
	return &MLTaskHandler{
		BaseTaskHandler: NewBaseTaskHandler(server),
	}
}

// ListTasks handles ML task listing requests (filtered by type=ml)
func (h *MLTaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	// Get all tasks and filter for ML tasks
	allTasks, err := h.server.tasks.ListTasks()
	if err != nil {
		h.server.writeError(w, http.StatusInternalServerError, "Failed to list tasks", err)
		return
	}

	// Filter for ML tasks
	var mlTasks []*tasks.Task
	for _, task := range allTasks {
		if task.Type == "ml" {
			mlTasks = append(mlTasks, task)
		}
	}

	h.server.writeJSON(w, http.StatusOK, mlTasks)
}

// CreateTask handles ML task creation requests
func (h *MLTaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req ml.CreateModelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.server.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Convert ML model request to task request
	taskReq := &tasks.CreateTaskRequest{
		Name:       req.Name,
		Type:       "ml",
		Command:    "python3", // ML tasks use Python
		Args:       []string{req.ScriptPath, req.ModelPath},
		Env:        req.Config.Env,
		WorkingDir: "", // ML tasks don't have a specific working directory
		Resources: tasks.ResourceLimits{
			CPULimit:    req.Resources.CPULimit,
			MemoryLimit: req.Resources.MemoryLimit,
			DiskLimit:   req.Resources.DiskLimit,
		},
	}

	task, err := h.server.tasks.CreateTask(taskReq)
	if err != nil {
		h.server.writeError(w, http.StatusInternalServerError, "Failed to create ML task", err)
		return
	}

	h.server.writeJSON(w, http.StatusCreated, task)
}

// GetTask handles ML task retrieval requests
func (h *MLTaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "ML task not found", err)
		return
	}

	// Verify this is an ML task
	if task.Type != "ml" {
		h.server.writeError(w, http.StatusBadRequest, "Task is not an ML task", nil)
		return
	}

	h.server.writeJSON(w, http.StatusOK, task)
}

// StopTask handles ML task stopping requests
func (h *MLTaskHandler) StopTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Verify this is an ML task
	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "ML task not found", err)
		return
	}

	if task.Type != "ml" {
		h.server.writeError(w, http.StatusBadRequest, "Task is not an ML task", nil)
		return
	}

	if err := h.server.tasks.StopTask(taskID); err != nil {
		h.server.writeError(w, http.StatusInternalServerError, "Failed to stop ML task", err)
		return
	}

	h.server.writeJSON(w, http.StatusOK, map[string]string{"message": "ML task stopped"})
}

// GetTaskLogs handles ML task log requests
func (h *MLTaskHandler) GetTaskLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Verify this is an ML task
	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "ML task not found", err)
		return
	}

	if task.Type != "ml" {
		h.server.writeError(w, http.StatusBadRequest, "Task is not an ML task", nil)
		return
	}

	// Use base handler for logs
	h.BaseTaskHandler.GetTaskLogs(w, r)
}

// GetTaskInfo handles ML task information requests with ML-specific details
func (h *MLTaskHandler) GetTaskInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Verify this is an ML task
	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "ML task not found", err)
		return
	}

	if task.Type != "ml" {
		h.server.writeError(w, http.StatusBadRequest, "Task is not an ML task", nil)
		return
	}

	// Return task info with ML-specific details
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
		"ml_specific": map[string]interface{}{
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

// GetTaskHealth handles ML task health check requests
func (h *MLTaskHandler) GetTaskHealth(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Verify this is an ML task
	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "ML task not found", err)
		return
	}

	if task.Type != "ml" {
		h.server.writeError(w, http.StatusBadRequest, "Task is not an ML task", nil)
		return
	}

	// Check task health
	if task.Status != "running" {
		h.server.writeError(w, http.StatusServiceUnavailable, "ML task is not running", nil)
		return
	}

	response := map[string]interface{}{
		"status":    "healthy",
		"task_id":   taskID,
		"timestamp": time.Now().Format(time.RFC3339),
		"ml_specific": map[string]interface{}{
			"inference_ready": true,
		},
	}

	h.server.writeJSON(w, http.StatusOK, response)
}

// Predict handles ML task prediction requests (ML-specific operation)
func (h *MLTaskHandler) Predict(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Verify this is an ML task
	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "ML task not found", err)
		return
	}

	if task.Type != "ml" {
		h.server.writeError(w, http.StatusBadRequest, "Task is not an ML task", nil)
		return
	}

	var req ml.InferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.server.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Use ML service for inference
	response, err := h.server.ml.Infer(taskID, &req)
	if err != nil {
		h.server.writeError(w, http.StatusInternalServerError, "Inference failed", err)
		return
	}

	h.server.writeJSON(w, http.StatusOK, response)
}
