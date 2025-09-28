package api

import (
	"encoding/json"
	"net/http"
	"time"

	"cnet/internal/agent/tasks"

	"github.com/gorilla/mux"
)

// ProcessTaskHandler handles process-specific task operations
type ProcessTaskHandler struct {
	*BaseTaskHandler
}

// NewProcessTaskHandler creates a new process task handler
func NewProcessTaskHandler(server *Server) *ProcessTaskHandler {
	return &ProcessTaskHandler{
		BaseTaskHandler: NewBaseTaskHandler(server),
	}
}

// ListTasks handles process task listing requests (filtered by type=process)
func (h *ProcessTaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	// Get all tasks and filter for process tasks
	allTasks, err := h.server.tasks.ListTasks()
	if err != nil {
		h.server.writeError(w, http.StatusInternalServerError, "Failed to list tasks", err)
		return
	}

	// Filter for process tasks
	var processTasks []*tasks.Task
	for _, task := range allTasks {
		if task.Type == "process" {
			processTasks = append(processTasks, task)
		}
	}

	h.server.writeJSON(w, http.StatusOK, processTasks)
}

// CreateTask handles process task creation requests
func (h *ProcessTaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req tasks.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.server.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Ensure task type is process
	req.Type = "process"

	task, err := h.server.tasks.CreateTask(&req)
	if err != nil {
		h.server.writeError(w, http.StatusInternalServerError, "Failed to create process task", err)
		return
	}

	h.server.writeJSON(w, http.StatusCreated, task)
}

// GetTask handles process task retrieval requests
func (h *ProcessTaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "Process task not found", err)
		return
	}

	// Verify this is a process task
	if task.Type != "process" {
		h.server.writeError(w, http.StatusBadRequest, "Task is not a process task", nil)
		return
	}

	h.server.writeJSON(w, http.StatusOK, task)
}

// StopTask handles process task stopping requests
func (h *ProcessTaskHandler) StopTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Verify this is a process task
	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "Process task not found", err)
		return
	}

	if task.Type != "process" {
		h.server.writeError(w, http.StatusBadRequest, "Task is not a process task", nil)
		return
	}

	if err := h.server.tasks.StopTask(taskID); err != nil {
		h.server.writeError(w, http.StatusInternalServerError, "Failed to stop process task", err)
		return
	}

	h.server.writeJSON(w, http.StatusOK, map[string]string{"message": "Process task stopped"})
}

// GetTaskLogs handles process task log requests
func (h *ProcessTaskHandler) GetTaskLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Verify this is a process task
	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "Process task not found", err)
		return
	}

	if task.Type != "process" {
		h.server.writeError(w, http.StatusBadRequest, "Task is not a process task", nil)
		return
	}

	// Use base handler for logs
	h.BaseTaskHandler.GetTaskLogs(w, r)
}

// GetTaskInfo handles process task information requests with process-specific details
func (h *ProcessTaskHandler) GetTaskInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Verify this is a process task
	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "Process task not found", err)
		return
	}

	if task.Type != "process" {
		h.server.writeError(w, http.StatusBadRequest, "Task is not a process task", nil)
		return
	}

	// Return task info with process-specific details
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
		"process_specific": map[string]interface{}{
			"command": task.Command,
			"args":    task.Args,
			"pid": func() *int {
				if task.Process != nil {
					pid := task.Process.Pid
					return &pid
				} else {
					return nil
				}
			}(),
			"executable": task.Command,
		},
	}

	h.server.writeJSON(w, http.StatusOK, info)
}

// GetTaskHealth handles process task health check requests
func (h *ProcessTaskHandler) GetTaskHealth(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Verify this is a process task
	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "Process task not found", err)
		return
	}

	if task.Type != "process" {
		h.server.writeError(w, http.StatusBadRequest, "Task is not a process task", nil)
		return
	}

	// Check task health
	if task.Status != "running" {
		h.server.writeError(w, http.StatusServiceUnavailable, "Process task is not running", nil)
		return
	}

	response := map[string]interface{}{
		"status":    "healthy",
		"task_id":   taskID,
		"timestamp": time.Now().Format(time.RFC3339),
		"process_specific": map[string]interface{}{
			"process_ready": true,
			"pid": func() *int {
				if task.Process != nil {
					pid := task.Process.Pid
					return &pid
				} else {
					return nil
				}
			}(),
		},
	}

	h.server.writeJSON(w, http.StatusOK, response)
}
