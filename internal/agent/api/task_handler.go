package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"cnet/internal/agent/tasks"

	"github.com/gorilla/mux"
)

// TaskHandler interface for different task types
type TaskHandler interface {
	// Basic task operations
	ListTasks(w http.ResponseWriter, r *http.Request)
	CreateTask(w http.ResponseWriter, r *http.Request)
	GetTask(w http.ResponseWriter, r *http.Request)
	StopTask(w http.ResponseWriter, r *http.Request)
	GetTaskLogs(w http.ResponseWriter, r *http.Request)

	// Task-specific operations (optional)
	GetTaskInfo(w http.ResponseWriter, r *http.Request)
	GetTaskHealth(w http.ResponseWriter, r *http.Request)
}

// BaseTaskHandler provides common functionality for all task handlers
type BaseTaskHandler struct {
	server *Server
}

// NewBaseTaskHandler creates a new base task handler
func NewBaseTaskHandler(server *Server) *BaseTaskHandler {
	return &BaseTaskHandler{
		server: server,
	}
}

// ListTasks handles task listing requests
func (h *BaseTaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	taskList, err := h.server.tasks.ListTasks()
	if err != nil {
		h.server.writeError(w, http.StatusInternalServerError, "Failed to list tasks", err)
		return
	}

	h.server.writeJSON(w, http.StatusOK, taskList)
}

// CreateTask handles task creation requests
func (h *BaseTaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req tasks.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.server.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	task, err := h.server.tasks.CreateTask(&req)
	if err != nil {
		h.server.writeError(w, http.StatusInternalServerError, "Failed to create task", err)
		return
	}

	h.server.writeJSON(w, http.StatusCreated, task)
}

// GetTask handles task retrieval requests
func (h *BaseTaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "Task not found", err)
		return
	}

	h.server.writeJSON(w, http.StatusOK, task)
}

// StopTask handles task stopping requests
func (h *BaseTaskHandler) StopTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	if err := h.server.tasks.StopTask(taskID); err != nil {
		h.server.writeError(w, http.StatusInternalServerError, "Failed to stop task", err)
		return
	}

	h.server.writeJSON(w, http.StatusOK, map[string]string{"message": "Task stopped"})
}

// GetTaskLogs handles task log requests
func (h *BaseTaskHandler) GetTaskLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Parse query parameters
	lines := 100
	if linesStr := r.URL.Query().Get("lines"); linesStr != "" {
		if l, err := strconv.Atoi(linesStr); err == nil && l > 0 {
			lines = l
		}
	}

	logs, err := h.server.tasks.GetTaskLogs(taskID, lines)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "Task logs not found", err)
		return
	}

	h.server.writeJSON(w, http.StatusOK, logs)
}

// GetTaskInfo handles task information requests (default implementation)
func (h *BaseTaskHandler) GetTaskInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "Task not found", err)
		return
	}

	// Return basic task info
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
	}

	h.server.writeJSON(w, http.StatusOK, info)
}

// GetTaskHealth handles task health check requests (default implementation)
func (h *BaseTaskHandler) GetTaskHealth(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "Task not found", err)
		return
	}

	// Check task health
	if task.Status != "running" {
		h.server.writeError(w, http.StatusServiceUnavailable, "Task is not running", nil)
		return
	}

	response := map[string]interface{}{
		"status":    "healthy",
		"task_id":   taskID,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	h.server.writeJSON(w, http.StatusOK, response)
}
