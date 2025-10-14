package executor

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"cnet/internal/workload"

	"github.com/sirupsen/logrus"
)

// ProcessExecutor 进程执行器
type ProcessExecutor struct {
	logger    *logrus.Logger
	processes map[string]*exec.Cmd
	logFiles  map[string]string
}

// NewProcessExecutor 创建进程执行器
func NewProcessExecutor(logger *logrus.Logger) *ProcessExecutor {
	return &ProcessExecutor{
		logger:    logger,
		processes: make(map[string]*exec.Cmd),
		logFiles:  make(map[string]string),
	}
}

// Init 初始化进程执行器
func (e *ProcessExecutor) Init(ctx context.Context) error {
	e.logger.Debug("Process Executor initialized")
	return nil
}

// Execute 执行进程workload
func (e *ProcessExecutor) Execute(ctx context.Context, w workload.Workload) error {
	pw, ok := w.(*workload.ProcessWorkload)
	if !ok {
		return fmt.Errorf("invalid workload type, expected ProcessWorkload")
	}

	// 创建命令
	cmd := exec.CommandContext(ctx, pw.Command, pw.Args...)

	// 设置环境变量
	cmd.Env = os.Environ()
	for k, v := range pw.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// 设置工作目录
	if pw.WorkingDir != "" {
		cmd.Dir = pw.WorkingDir
	}

	// 创建日志文件
	logDir := filepath.Join(os.TempDir(), "cnet", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	logFile := filepath.Join(logDir, fmt.Sprintf("%s.log", w.GetID()))
	f, err := os.Create(logFile)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer f.Close()

	// 重定向输出到日志文件
	cmd.Stdout = f
	cmd.Stderr = f

	// 启动进程
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	// 保存进程信息
	e.processes[w.GetID()] = cmd
	e.logFiles[w.GetID()] = logFile
	pw.PID = cmd.Process.Pid
	pw.SetStatus(workload.StatusRunning)

	e.logger.WithFields(logrus.Fields{
		"workload_id": w.GetID(),
		"pid":         pw.PID,
		"command":     pw.Command,
	}).Info("Process started")

	// 等待进程结束
	go func() {
		err := cmd.Wait()
		if err != nil {
			pw.SetStatus(workload.StatusFailed)
			e.logger.WithError(err).WithField("workload_id", w.GetID()).Error("Process failed")
		} else {
			pw.SetStatus(workload.StatusCompleted)
			e.logger.WithField("workload_id", w.GetID()).Info("Process completed")
		}

		// 获取退出码
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				exitCode := status.ExitStatus()
				pw.ExitCode = &exitCode
			}
		} else if err == nil {
			exitCode := 0
			pw.ExitCode = &exitCode
		}
	}()

	return nil
}

// Stop 停止进程workload
func (e *ProcessExecutor) Stop(ctx context.Context, w workload.Workload) error {
	cmd, exists := e.processes[w.GetID()]
	if !exists {
		return fmt.Errorf("process not found for workload: %s", w.GetID())
	}

	if cmd.Process == nil {
		return fmt.Errorf("process already stopped")
	}

	// 发送终止信号
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		// 如果SIGTERM失败，强制杀死
		if killErr := cmd.Process.Kill(); killErr != nil {
			return fmt.Errorf("failed to kill process: %w", killErr)
		}
	}

	w.SetStatus(workload.StatusStopped)
	e.logger.WithField("workload_id", w.GetID()).Info("Process stopped")

	return nil
}

// GetLogs 获取进程日志
func (e *ProcessExecutor) GetLogs(ctx context.Context, w workload.Workload, lines int) ([]string, error) {
	logFile, exists := e.logFiles[w.GetID()]
	if !exists {
		return nil, fmt.Errorf("log file not found for workload: %s", w.GetID())
	}

	file, err := os.Open(logFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// 读取最后N行
	var logLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		logLines = append(logLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	// 返回最后N行
	if lines > 0 && len(logLines) > lines {
		logLines = logLines[len(logLines)-lines:]
	}

	return logLines, nil
}

// GetStatus 获取进程状态
func (e *ProcessExecutor) GetStatus(ctx context.Context, w workload.Workload) (workload.WorkloadStatus, error) {
	return w.GetStatus(), nil
}
