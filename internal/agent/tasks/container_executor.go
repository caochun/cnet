package tasks

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"cnet/internal/logger"
)

// ContainerExecutor executes Docker containers
type ContainerExecutor struct {
	logger *logger.Logger
}

// NewContainerExecutor creates a new ContainerExecutor
func NewContainerExecutor(logger *logger.Logger) *ContainerExecutor {
	return &ContainerExecutor{
		logger: logger,
	}
}

// Execute executes a Docker container
func (e *ContainerExecutor) Execute(ctx context.Context, task *Task) error {
	// Create log file
	logFile, err := os.Create(task.LogFile)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer logFile.Close()

	// Build Docker command
	// task.Command should be the Docker image name
	// task.Args should contain Docker run arguments
	args := []string{"run", "--rm"}

	// Add container name
	args = append(args, "--name", "cnet-"+task.ID)

	// Add environment variables
	for key, value := range task.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
	}

	// Add working directory if specified
	if task.WorkingDir != "" {
		args = append(args, "-w", task.WorkingDir)
	}

	// Add the image name and any additional arguments
	args = append(args, task.Command)
	args = append(args, task.Args...)

	e.logger.Infof("Running Docker container for task %s: docker %s", task.ID, strings.Join(args, " "))

	// Try docker first, then podman
	var cmd *exec.Cmd
	if _, err := exec.LookPath("docker"); err == nil {
		// Use docker if available
		cmd = exec.CommandContext(ctx, "docker", args...)
		e.logger.Infof("Using docker for container execution")
	} else {
		// Fall back to podman
		cmd = exec.CommandContext(ctx, "podman", args...)
		e.logger.Infof("Using podman for container execution")
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Start the container
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Docker container: %w", err)
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

// Stop stops a Docker container
func (e *ContainerExecutor) Stop(task *Task) error {
	// Try to stop the container by name
	containerName := "cnet-" + task.ID
	cmd := exec.Command("docker", "stop", containerName)

	if err := cmd.Run(); err != nil {
		e.logger.Warnf("Failed to stop container %s: %v", containerName, err)
		// Continue to kill the process if container stop failed
	}

	// Also kill the process if it's still running
	if task.Process != nil {
		return task.Process.Kill()
	}

	return nil
}

// GetLogs retrieves logs for a Docker container
func (e *ContainerExecutor) GetLogs(task *Task, lines int) ([]string, error) {
	// First try to get logs from the log file
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
