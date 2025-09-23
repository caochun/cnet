package tasks

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"cnet/internal/config"
	"cnet/internal/logger"

	"github.com/google/uuid"
)

// Service represents the tasks management service
type Service struct {
	config    *config.Config
	logger    *logger.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.RWMutex
	tasks     map[string]*Task
	executors map[string]Executor
}

// Task represents a running task
type Task struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Status     string            `json:"status"`
	Command    string            `json:"command"`
	Args       []string          `json:"args"`
	Env        map[string]string `json:"env"`
	WorkingDir string            `json:"working_dir"`
	Resources  ResourceLimits    `json:"resources"`
	CreatedAt  time.Time         `json:"created_at"`
	StartedAt  *time.Time        `json:"started_at,omitempty"`
	StoppedAt  *time.Time        `json:"stopped_at,omitempty"`
	ExitCode   *int              `json:"exit_code,omitempty"`
	Process    *os.Process       `json:"-"`
	LogFile    string            `json:"log_file"`
}

// ResourceLimits represents resource limits for a task
type ResourceLimits struct {
	CPULimit    float64 `json:"cpu_limit"`
	MemoryLimit int64   `json:"memory_limit"`
	DiskLimit   int64   `json:"disk_limit"`
}

// CreateTaskRequest represents a request to create a task
type CreateTaskRequest struct {
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Command    string            `json:"command"`
	Args       []string          `json:"args"`
	Env        map[string]string `json:"env"`
	WorkingDir string            `json:"working_dir"`
	Resources  ResourceLimits    `json:"resources"`
}

// Executor interface for different task types
type Executor interface {
	Execute(ctx context.Context, task *Task) error
	Stop(task *Task) error
	GetLogs(task *Task, lines int) ([]string, error)
}

// ProcessExecutor executes local processes
type ProcessExecutor struct {
	logger *logger.Logger
}

// ContainerExecutor executes Docker containers
type ContainerExecutor struct {
	logger *logger.Logger
}

// VMExecutor executes virtual machines (placeholder)
type VMExecutor struct {
	logger *logger.Logger
}

// New creates a new tasks service
func New(cfg *config.Config, log *logger.Logger) (*Service, error) {
	ctx, cancel := context.WithCancel(context.Background())

	service := &Service{
		config:    cfg,
		logger:    log,
		ctx:       ctx,
		cancel:    cancel,
		tasks:     make(map[string]*Task),
		executors: make(map[string]Executor),
	}

	// Register executors
	service.executors["process"] = &ProcessExecutor{logger: log}
	service.executors["container"] = &ContainerExecutor{logger: log}
	service.executors["vm"] = &VMExecutor{logger: log}

	return service, nil
}

// Start starts the tasks service
func (s *Service) Start(ctx context.Context) error {
	// Create logs directory
	logsDir := filepath.Join(os.TempDir(), "cnet", "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	return nil
}

// Stop stops the tasks service
func (s *Service) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Stop all running tasks
	for _, task := range s.tasks {
		if task.Status == "running" {
			if err := s.stopTask(task); err != nil {
				s.logger.Errorf("Failed to stop task %s: %v", task.ID, err)
			}
		}
	}

	s.cancel()
	return nil
}

// CreateTask creates a new task
func (s *Service) CreateTask(req *CreateTaskRequest) (*Task, error) {
	taskID := uuid.New().String()

	task := &Task{
		ID:         taskID,
		Name:       req.Name,
		Type:       req.Type,
		Status:     "pending",
		Command:    req.Command,
		Args:       req.Args,
		Env:        req.Env,
		WorkingDir: req.WorkingDir,
		Resources:  req.Resources,
		CreatedAt:  time.Now(),
		LogFile:    filepath.Join(os.TempDir(), "cnet", "logs", taskID+".log"),
	}

	s.mu.Lock()
	s.tasks[taskID] = task
	s.mu.Unlock()

	// Start task execution in goroutine
	go s.executeTask(task)

	return task, nil
}

// GetTask retrieves a task by ID
func (s *Service) GetTask(id string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, exists := s.tasks[id]
	if !exists {
		return nil, fmt.Errorf("task not found")
	}

	return task, nil
}

// ListTasks lists all tasks
func (s *Service) ListTasks() ([]*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// StopTask stops a task
func (s *Service) StopTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[id]
	if !exists {
		return fmt.Errorf("task not found")
	}

	return s.stopTask(task)
}

// GetTaskLogs retrieves task logs
func (s *Service) GetTaskLogs(id string, lines int) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, exists := s.tasks[id]
	if !exists {
		return nil, fmt.Errorf("task not found")
	}

	executor, exists := s.executors[task.Type]
	if !exists {
		return nil, fmt.Errorf("executor not found for type: %s", task.Type)
	}

	return executor.GetLogs(task, lines)
}

// executeTask executes a task
func (s *Service) executeTask(task *Task) {
	executor, exists := s.executors[task.Type]
	if !exists {
		s.logger.Errorf("Executor not found for type: %s", task.Type)
		task.Status = "failed"
		return
	}

	// Update task status
	s.mu.Lock()
	task.Status = "running"
	now := time.Now()
	task.StartedAt = &now
	s.mu.Unlock()

	// Execute task
	if err := executor.Execute(s.ctx, task); err != nil {
		s.logger.Errorf("Task execution failed: %v", err)
		s.mu.Lock()
		task.Status = "failed"
		now := time.Now()
		task.StoppedAt = &now
		s.mu.Unlock()
		return
	}

	// Task completed successfully
	s.mu.Lock()
	task.Status = "completed"
	now = time.Now()
	task.StoppedAt = &now
	s.mu.Unlock()
}

// stopTask stops a task
func (s *Service) stopTask(task *Task) error {
	executor, exists := s.executors[task.Type]
	if !exists {
		return fmt.Errorf("executor not found for type: %s", task.Type)
	}

	if err := executor.Stop(task); err != nil {
		return fmt.Errorf("failed to stop task: %w", err)
	}

	task.Status = "stopped"
	now := time.Now()
	task.StoppedAt = &now

	return nil
}

// ProcessExecutor implementation

// Execute executes a local process
func (e *ProcessExecutor) Execute(ctx context.Context, task *Task) error {
	cmd := exec.CommandContext(ctx, task.Command, task.Args...)

	// Set working directory
	if task.WorkingDir != "" {
		cmd.Dir = task.WorkingDir
	}

	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range task.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Create log file
	logFile, err := os.Create(task.LogFile)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer logFile.Close()

	// Set up output
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Start process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	task.Process = cmd.Process

	// Wait for completion
	err = cmd.Wait()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode := exitError.ExitCode()
			task.ExitCode = &exitCode
		}
		return err
	}

	exitCode := 0
	task.ExitCode = &exitCode
	return nil
}

// Stop stops a local process
func (e *ProcessExecutor) Stop(task *Task) error {
	if task.Process != nil {
		return task.Process.Kill()
	}
	return nil
}

// GetLogs retrieves logs for a local process
func (e *ProcessExecutor) GetLogs(task *Task, lines int) ([]string, error) {
	file, err := os.Open(task.LogFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// Read all lines
	var allLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	// Return last N lines
	start := 0
	if len(allLines) > lines {
		start = len(allLines) - lines
	}

	return allLines[start:], nil
}

// ContainerExecutor implementation (placeholder)

// Execute executes a Docker container
func (e *ContainerExecutor) Execute(ctx context.Context, task *Task) error {
	// TODO: Implement Docker container execution
	return fmt.Errorf("container execution not implemented")
}

// Stop stops a Docker container
func (e *ContainerExecutor) Stop(task *Task) error {
	// TODO: Implement Docker container stopping
	return fmt.Errorf("container stopping not implemented")
}

// GetLogs retrieves logs for a Docker container
func (e *ContainerExecutor) GetLogs(task *Task, lines int) ([]string, error) {
	// TODO: Implement Docker container log retrieval
	return nil, fmt.Errorf("container log retrieval not implemented")
}

// VMExecutor implementation (placeholder)

// Execute executes a virtual machine
func (e *VMExecutor) Execute(ctx context.Context, task *Task) error {
	// TODO: Implement VM execution
	return fmt.Errorf("VM execution not implemented")
}

// Stop stops a virtual machine
func (e *VMExecutor) Stop(task *Task) error {
	// TODO: Implement VM stopping
	return fmt.Errorf("VM stopping not implemented")
}

// GetLogs retrieves logs for a virtual machine
func (e *VMExecutor) GetLogs(task *Task, lines int) ([]string, error) {
	// TODO: Implement VM log retrieval
	return nil, fmt.Errorf("VM log retrieval not implemented")
}
