package api

import (
	"encoding/json"
	"net/http"
	"time"

	"cnet/internal/agent/tasks"

	"github.com/gorilla/mux"
)

// VMTaskHandler handles VM-specific task operations
type VMTaskHandler struct {
	*BaseTaskHandler
}

// NewVMTaskHandler creates a new VM task handler
func NewVMTaskHandler(server *Server) *VMTaskHandler {
	return &VMTaskHandler{
		BaseTaskHandler: NewBaseTaskHandler(server),
	}
}

// ListTasks handles VM task listing requests (filtered by type=vm)
func (h *VMTaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	// Get all tasks and filter for VM tasks
	allTasks, err := h.server.tasks.ListTasks()
	if err != nil {
		h.server.writeError(w, http.StatusInternalServerError, "Failed to list tasks", err)
		return
	}

	// Filter for VM tasks
	var vmTasks []*tasks.Task
	for _, task := range allTasks {
		if task.Type == "vm" {
			vmTasks = append(vmTasks, task)
		}
	}

	h.server.writeJSON(w, http.StatusOK, vmTasks)
}

// CreateTask handles VM task creation requests
func (h *VMTaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req tasks.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.server.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Ensure task type is vm
	req.Type = "vm"

	task, err := h.server.tasks.CreateTask(&req)
	if err != nil {
		h.server.writeError(w, http.StatusInternalServerError, "Failed to create VM task", err)
		return
	}

	h.server.writeJSON(w, http.StatusCreated, task)
}

// GetTask handles VM task retrieval requests
func (h *VMTaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "VM task not found", err)
		return
	}

	// Verify this is a VM task
	if task.Type != "vm" {
		h.server.writeError(w, http.StatusBadRequest, "Task is not a VM task", nil)
		return
	}

	h.server.writeJSON(w, http.StatusOK, task)
}

// StopTask handles VM task stopping requests
func (h *VMTaskHandler) StopTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Verify this is a VM task
	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "VM task not found", err)
		return
	}

	if task.Type != "vm" {
		h.server.writeError(w, http.StatusBadRequest, "Task is not a VM task", nil)
		return
	}

	if err := h.server.tasks.StopTask(taskID); err != nil {
		h.server.writeError(w, http.StatusInternalServerError, "Failed to stop VM task", err)
		return
	}

	h.server.writeJSON(w, http.StatusOK, map[string]string{"message": "VM task stopped"})
}

// GetTaskLogs handles VM task log requests
func (h *VMTaskHandler) GetTaskLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Verify this is a VM task
	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "VM task not found", err)
		return
	}

	if task.Type != "vm" {
		h.server.writeError(w, http.StatusBadRequest, "Task is not a VM task", nil)
		return
	}

	// Use base handler for logs
	h.BaseTaskHandler.GetTaskLogs(w, r)
}

// GetTaskInfo handles VM task information requests with VM-specific details
func (h *VMTaskHandler) GetTaskInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Verify this is a VM task
	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "VM task not found", err)
		return
	}

	if task.Type != "vm" {
		h.server.writeError(w, http.StatusBadRequest, "Task is not a VM task", nil)
		return
	}

	// Return task info with VM-specific details
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
		"vm_specific": map[string]interface{}{
			"image":     task.Command, // VM image name
			"config":    task.Args,    // VM configuration
			"memory_mb": task.Resources.MemoryLimit / (1024 * 1024),
			"cpu_cores": int(task.Resources.CPULimit),
			"disk_gb":   task.Resources.DiskLimit / (1024 * 1024 * 1024),
		},
	}

	h.server.writeJSON(w, http.StatusOK, info)
}

// GetTaskHealth handles VM task health check requests
func (h *VMTaskHandler) GetTaskHealth(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Verify this is a VM task
	task, err := h.server.tasks.GetTask(taskID)
	if err != nil {
		h.server.writeError(w, http.StatusNotFound, "VM task not found", err)
		return
	}

	if task.Type != "vm" {
		h.server.writeError(w, http.StatusBadRequest, "Task is not a VM task", nil)
		return
	}

	// Check task health
	if task.Status != "running" {
		h.server.writeError(w, http.StatusServiceUnavailable, "VM task is not running", nil)
		return
	}

	response := map[string]interface{}{
		"status":    "healthy",
		"task_id":   taskID,
		"timestamp": time.Now().Format(time.RFC3339),
		"vm_specific": map[string]interface{}{
			"vm_ready": true,
		},
	}

	h.server.writeJSON(w, http.StatusOK, response)
}
