package executor

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"cnet/internal/workload"

	"github.com/sirupsen/logrus"
)

// YOLOExecutor YOLO模型推理服务执行器
// 实现MLModelExecutor接口，管理YOLO HTTP推理服务进程
type YOLOExecutor struct {
	logger   *logrus.Logger
	mu       sync.RWMutex
	services map[string]*YOLOService // workload_id -> service
}

// YOLOService YOLO推理服务实例
type YOLOService struct {
	workloadID      string
	modelPath       string
	modelType       string // yolo11, yolo8, yolo5
	endpoint        string
	process         *exec.Cmd
	port            int
	lastHealthCheck time.Time
	healthTicker    *time.Ticker
	restartCount    int
	ctx             context.Context
	cancel          context.CancelFunc
}

// NewYOLOExecutor 创建YOLO执行器
func NewYOLOExecutor(logger *logrus.Logger) *YOLOExecutor {
	return &YOLOExecutor{
		logger:   logger,
		services: make(map[string]*YOLOService),
	}
}

// Init 初始化YOLO执行器
func (e *YOLOExecutor) Init(ctx context.Context) error {
	e.logger.Info("YOLO Executor initialized")
	// TODO: 可以在这里检查GoCV依赖、YOLO模型文件等
	return nil
}

// Execute 执行MLModel workload - 启动YOLO推理服务
func (e *YOLOExecutor) Execute(ctx context.Context, w workload.Workload) error {
	mw, ok := w.(*workload.MLModelWorkload)
	if !ok {
		return fmt.Errorf("invalid workload type, expected MLModelWorkload")
	}

	// 验证是YOLO模型
	if mw.ModelType != "yolo" {
		return fmt.Errorf("invalid model type for YOLOExecutor: %s", mw.ModelType)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// 检查是否已经运行
	if _, exists := e.services[w.GetID()]; exists {
		return fmt.Errorf("YOLO service already running for workload: %s", w.GetID())
	}

	// 启动推理服务进程
	service, err := e.startService(ctx, mw)
	if err != nil {
		return fmt.Errorf("failed to start YOLO service: %w", err)
	}

	// 保存服务实例
	e.services[w.GetID()] = service

	// 更新workload状态
	mw.Endpoint = service.endpoint
	mw.ProcessPID = service.process.Process.Pid
	mw.SetStatus(workload.StatusRunning)

	e.logger.WithFields(logrus.Fields{
		"workload_id": w.GetID(),
		"model_path":  mw.ModelPath,
		"endpoint":    service.endpoint,
		"pid":         service.process.Process.Pid,
	}).Info("YOLO推理服务已启动")

	// 启动健康检查
	go e.startHealthCheck(service)

	return nil
}

// startService 启动YOLO推理服务进程
func (e *YOLOExecutor) startService(ctx context.Context, mw *workload.MLModelWorkload) (*YOLOService, error) {
	serviceCtx, cancel := context.WithCancel(ctx)

	// 构造启动命令（不使用CommandContext，避免自动kill）
	cmd := exec.Command(
		"./bin/cnet-inference-yolo",
		"--model", mw.ModelPath,
		"--port", fmt.Sprintf("%d", mw.ServicePort),
	)

	if mw.ModelConfig != "" {
		cmd.Args = append(cmd.Args, "--config", mw.ModelConfig)
	}

	// 设置工作目录和输出
	cmd.Dir = "." // 当前目录
	
	// 创建日志文件捕获输出
	logFile := fmt.Sprintf("yolo_service_%d.log", mw.ServicePort)
	outFile, err := os.Create(logFile)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}
	cmd.Stdout = outFile
	cmd.Stderr = outFile

	// 启动进程
	if err := cmd.Start(); err != nil {
		cancel()
		outFile.Close()
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	service := &YOLOService{
		workloadID:   mw.GetID(),
		modelPath:    mw.ModelPath,
		modelType:    mw.Framework,
		endpoint:     fmt.Sprintf("http://%s:%d", mw.ServiceHost, mw.ServicePort),
		process:      cmd,
		port:         mw.ServicePort,
		restartCount: 0,
		ctx:          serviceCtx,
		cancel:       cancel,
	}

	// 等待服务启动（健康检查）
	if err := e.waitForService(service, 60*time.Second); err != nil {
		cmd.Process.Kill()
		cancel()
		return nil, fmt.Errorf("service failed to start: %w", err)
	}

	return service, nil
}

// waitForService 等待服务启动
func (e *YOLOExecutor) waitForService(service *YOLOService, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if err := e.checkServiceHealth(service); err == nil {
			return nil
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("service did not start within timeout")
}

// checkServiceHealth 检查服务健康状态
func (e *YOLOExecutor) checkServiceHealth(service *YOLOService) error {
	resp, err := http.Get(service.endpoint + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unhealthy status code: %d", resp.StatusCode)
	}

	service.lastHealthCheck = time.Now()
	return nil
}

// startHealthCheck 启动健康检查
func (e *YOLOExecutor) startHealthCheck(service *YOLOService) {
	service.healthTicker = time.NewTicker(30 * time.Second)

	for {
		select {
		case <-service.ctx.Done():
			service.healthTicker.Stop()
			return
		case <-service.healthTicker.C:
			if err := e.checkServiceHealth(service); err != nil {
				e.logger.WithFields(logrus.Fields{
					"workload_id": service.workloadID,
					"error":       err,
				}).Warn("YOLO service health check failed, attempting restart")

				if err := e.restartService(service.workloadID); err != nil {
					e.logger.WithError(err).Error("Failed to restart YOLO service")
				}
			}
		}
	}
}

// restartService 重启服务
func (e *YOLOExecutor) restartService(workloadID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	service, exists := e.services[workloadID]
	if !exists {
		return fmt.Errorf("service not found: %s", workloadID)
	}

	service.restartCount++
	if service.restartCount > 3 {
		return fmt.Errorf("max restart attempts (%d) reached", service.restartCount)
	}

	e.logger.WithFields(logrus.Fields{
		"workload_id":   workloadID,
		"restart_count": service.restartCount,
	}).Info("Restarting YOLO service")

	// 停止旧进程
	if service.process != nil && service.process.Process != nil {
		service.process.Process.Kill()
	}

	// TODO: 重新启动服务
	// 需要获取原始的workload信息来重新启动
	// 暂时返回错误，完整实现需要保存workload引用

	return fmt.Errorf("restart not fully implemented")
}

// Stop 停止YOLO workload
func (e *YOLOExecutor) Stop(ctx context.Context, w workload.Workload) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	service, exists := e.services[w.GetID()]
	if !exists {
		return fmt.Errorf("service not found for workload: %s", w.GetID())
	}

	e.logger.WithField("workload_id", w.GetID()).Info("Stopping YOLO service")

	// 停止健康检查
	if service.healthTicker != nil {
		service.healthTicker.Stop()
	}

	// 取消context
	if service.cancel != nil {
		service.cancel()
	}

	// 停止进程
	if service.process != nil && service.process.Process != nil {
		if err := service.process.Process.Kill(); err != nil {
			e.logger.WithError(err).Warn("Failed to kill YOLO process")
		}
		service.process.Wait() // 等待进程退出
	}

	// 清理
	delete(e.services, w.GetID())
	w.SetStatus(workload.StatusStopped)

	return nil
}

// GetLogs 获取YOLO服务日志
func (e *YOLOExecutor) GetLogs(ctx context.Context, w workload.Workload, lines int) ([]string, error) {
	// TODO: 实现日志获取（从进程stdout/stderr）
	return []string{"YOLO service logs (not implemented)"}, nil
}

// GetStatus 获取YOLO服务状态
func (e *YOLOExecutor) GetStatus(ctx context.Context, w workload.Workload) (workload.WorkloadStatus, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	service, exists := e.services[w.GetID()]
	if !exists {
		return workload.StatusStopped, nil
	}

	// 检查进程是否还在运行
	if service.process != nil && service.process.Process != nil {
		// 检查进程状态（发送signal 0）
		if err := service.process.Process.Signal(nil); err != nil {
			// 进程已不存在
			return workload.StatusFailed, nil
		}
		return workload.StatusRunning, nil
	}

	return w.GetStatus(), nil
}

// GetInferenceEndpoint 获取推理服务endpoint
func (e *YOLOExecutor) GetInferenceEndpoint(workloadID string) (string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	service, exists := e.services[workloadID]
	if !exists {
		return "", fmt.Errorf("service not found: %s", workloadID)
	}

	return service.endpoint, nil
}

// HealthCheck 检查推理服务健康状态
func (e *YOLOExecutor) HealthCheck(ctx context.Context, workloadID string) error {
	e.mu.RLock()
	service, exists := e.services[workloadID]
	e.mu.RUnlock()

	if !exists {
		return fmt.Errorf("service not found: %s", workloadID)
	}

	return e.checkServiceHealth(service)
}

