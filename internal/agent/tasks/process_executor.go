package tasks

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"

	"cnet/internal/logger"
)

// ProcessExecutor executes local processes
type ProcessExecutor struct {
	logger *logger.Logger
}

// NewProcessExecutor creates a new ProcessExecutor
func NewProcessExecutor(logger *logger.Logger) *ProcessExecutor {
	return &ProcessExecutor{
		logger: logger,
	}
}

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
