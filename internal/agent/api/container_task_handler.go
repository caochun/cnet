package api

import (
	"encoding/json"
	"net/http"
	"time"

	"cnet/internal/agent/tasks"

	"github.com/gorilla/mux"
)

// ContainerTaskHandler handles container-specific task operations
type ContainerTaskHandler struct {
	*BaseTaskHandler
}

// NewContainerTaskHandler creates a new container task handler
func NewContainerTaskHandler(server *Server) *ContainerTaskHandler {
	return &ContainerTaskHandler{
		BaseTaskHandler: NewBaseTaskHandler(server),
	}
}

// ListTasks handles container task listing requests (filtered by type=container)
func (h *ContainerTaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	// Get all tasks and filter for container tasks
	allTasks, err := h.server.tasks.ListTasks()
	if err != nil {
		h.server.writeError(w, http.StatusInternalServerError, "Failed to list tasks", err)
		return
	}

	// Filter for container tasks
	var containerTasks []*tasks.Task
	for _, task := range allTasks {
		if task.Type == "container" {
			containerTasks = append(containerTasks, task)
		}
	}

	h.server.writeJSON(w, http.StatusOK, containerTasks)
}

// CreateTask handles container task creation requests
func (h *ContainerTaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req tasks.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.server.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Ensure task type is container
	req.Type = "container"

	task, err := h.server.tasks.CreateTask(&req)
	if err != nil {
		h.server.writeError(w, http.StatusInternalServerError, "Failed to create container task", err)
		return
	}

	h.server.writeJSON(w, http.StatusCreated, task)
}

// GetTask handles container task retrieval requests
func (h *ContainerTaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "Container task not found", err)
		return
	}

	// Verify this is a container task
	if task.Type != "container" {
		h.server.writeError(w, http.StatusBadRequest, "Task is not a container task", nil)
		return
	}

	h.server.writeJSON(w, http.StatusOK, task)
}

// StopTask handles container task stopping requests
func (h *ContainerTaskHandler) StopTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Verify this is a container task
	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "Container task not found", err)
		return
	}

	if task.Type != "container" {
		h.server.writeError(w, http.StatusBadRequest, "Task is not a container task", nil)
		return
	}

	if err := h.server.tasks.StopTask(taskID); err != nil {
		h.server.writeError(w, http.StatusInternalServerError, "Failed to stop container task", err)
		return
	}

	h.server.writeJSON(w, http.StatusOK, map[string]string{"message": "Container task stopped"})
}

// GetTaskLogs handles container task log requests
func (h *ContainerTaskHandler) GetTaskLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Verify this is a container task
	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "Container task not found", err)
		return
	}

	if task.Type != "container" {
		h.server.writeError(w, http.StatusBadRequest, "Task is not a container task", nil)
		return
	}

	// Use base handler for logs
	h.BaseTaskHandler.GetTaskLogs(w, r)
}

// GetTaskInfo handles container task information requests with container-specific details
func (h *ContainerTaskHandler) GetTaskInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Verify this is a container task
	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "Container task not found", err)
		return
	}

	if task.Type != "container" {
		h.server.writeError(w, http.StatusBadRequest, "Task is not a container task", nil)
		return
	}

	// Return task info with container-specific details
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
		"container_specific": map[string]interface{}{
			"image":   task.Command, // Docker image name
			"command": task.Args,    // Container command and args
			"ports":   []string{},   // TODO: Extract from task config
			"volumes": []string{},   // TODO: Extract from task config
		},
	}

	h.server.writeJSON(w, http.StatusOK, info)
}

// GetTaskHealth handles container task health check requests
func (h *ContainerTaskHandler) GetTaskHealth(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Verify this is a container task
	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "Container task not found", err)
		return
	}

	if task.Type != "container" {
		h.server.writeError(w, http.StatusBadRequest, "Task is not a container task", nil)
		return
	}

	// Check task health
	if task.Status != "running" {
		h.server.writeError(w, http.StatusServiceUnavailable, "Container task is not running", nil)
		return
	}

	response := map[string]interface{}{
		"status":    "healthy",
		"task_id":   taskID,
		"timestamp": time.Now().Format(time.RFC3339),
		"container_specific": map[string]interface{}{
			"container_ready": true,
		},
	}

	h.server.writeJSON(w, http.StatusOK, response)
}
