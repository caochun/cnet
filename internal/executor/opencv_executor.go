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

// OpenCVExecutor OpenCV推理服务执行器
// 管理OpenCV HTTP推理服务进程
type OpenCVExecutor struct {
	logger   *logrus.Logger
	mu       sync.RWMutex
	services map[string]*OpenCVService // workload_id -> service
}

// OpenCVService OpenCV推理服务实例
type OpenCVService struct {
	workloadID      string
	cascadeType     string
	endpoint        string
	process         *exec.Cmd
	port            int
	lastHealthCheck time.Time
	healthTicker    *time.Ticker
	restartCount    int
	ctx             context.Context
	cancel          context.CancelFunc
}

// NewOpenCVExecutor 创建OpenCV执行器
func NewOpenCVExecutor(logger *logrus.Logger) *OpenCVExecutor {
	return &OpenCVExecutor{
		logger:   logger,
		services: make(map[string]*OpenCVService),
	}
}

// Init 初始化OpenCV执行器
func (e *OpenCVExecutor) Init(ctx context.Context) error {
	e.logger.Info("OpenCV Executor initialized")
	// TODO: 可以在这里检查GoCV依赖、OpenCV版本等
	return nil
}

// Execute 执行OpenCV workload - 启动推理服务
func (e *OpenCVExecutor) Execute(ctx context.Context, w workload.Workload) error {
	ow, ok := w.(*workload.OpenCVWorkload)
	if !ok {
		return fmt.Errorf("invalid workload type, expected OpenCVWorkload")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// 检查是否已经运行
	if _, exists := e.services[w.GetID()]; exists {
		return fmt.Errorf("OpenCV service already running for workload: %s", w.GetID())
	}

	// 启动推理服务进程
	service, err := e.startService(ctx, ow)
	if err != nil {
		return fmt.Errorf("failed to start OpenCV service: %w", err)
	}

	// 保存服务实例
	e.services[w.GetID()] = service

	// 更新workload状态
	ow.Endpoint = service.endpoint
	ow.ProcessPID = service.process.Process.Pid
	ow.SetStatus(workload.StatusRunning)

	e.logger.WithFields(logrus.Fields{
		"workload_id": w.GetID(),
		"endpoint":    service.endpoint,
		"pid":         service.process.Process.Pid,
	}).Info("OpenCV推理服务已启动")

	// 启动健康检查
	go e.startHealthCheck(service)

	return nil
}

// startService 启动OpenCV推理服务进程
func (e *OpenCVExecutor) startService(ctx context.Context, ow *workload.OpenCVWorkload) (*OpenCVService, error) {
	serviceCtx, cancel := context.WithCancel(ctx)

	// 构造启动命令（不使用CommandContext，避免自动kill）
	cmd := exec.Command(
		"./bin/cnet-inference-opencv",
		"--port", fmt.Sprintf("%d", ow.ServicePort),
		"--cascade-type", ow.CascadeType,
	)

	if ow.CascadePath != "" {
		cmd.Args = append(cmd.Args, "--cascade-path", ow.CascadePath)
	}

	// 设置工作目录和输出
	cmd.Dir = "." // 当前目录
	
	// 创建日志文件捕获输出
	logFile := fmt.Sprintf("opencv_service_%d.log", ow.ServicePort)
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

	service := &OpenCVService{
		workloadID:   ow.GetID(),
		cascadeType:  ow.CascadeType,
		endpoint:     fmt.Sprintf("http://%s:%d", ow.ServiceHost, ow.ServicePort),
		process:      cmd,
		port:         ow.ServicePort,
		restartCount: 0,
		ctx:          serviceCtx,
		cancel:       cancel,
	}

	// 等待服务启动（健康检查）
	if err := e.waitForService(service, 30*time.Second); err != nil {
		cmd.Process.Kill()
		cancel()
		return nil, fmt.Errorf("service failed to start: %w", err)
	}

	return service, nil
}

// waitForService 等待服务启动
func (e *OpenCVExecutor) waitForService(service *OpenCVService, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if err := e.checkServiceHealth(service); err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("service did not start within timeout")
}

// checkServiceHealth 检查服务健康状态
func (e *OpenCVExecutor) checkServiceHealth(service *OpenCVService) error {
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
func (e *OpenCVExecutor) startHealthCheck(service *OpenCVService) {
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
				}).Warn("Health check failed, attempting restart")

				if err := e.restartService(service.workloadID); err != nil {
					e.logger.WithError(err).Error("Failed to restart service")
				}
			}
		}
	}
}

// restartService 重启服务
func (e *OpenCVExecutor) restartService(workloadID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	service, exists := e.services[workloadID]
	if !exists {
		return fmt.Errorf("service not found: %s", workloadID)
	}

	service.restartCount++
	if service.restartCount > 3 {
		return fmt.Errorf("max restart attempts reached")
	}

	e.logger.WithFields(logrus.Fields{
		"workload_id":   workloadID,
		"restart_count": service.restartCount,
	}).Info("Restarting OpenCV service")

	// 停止旧进程
	if service.process != nil && service.process.Process != nil {
		service.process.Process.Kill()
	}

	// TODO: 重新启动服务
	// 需要获取原始的workload信息来重新启动
	// 暂时返回错误，完整实现需要保存workload引用

	return fmt.Errorf("restart not fully implemented")
}

// Stop 停止OpenCV workload
func (e *OpenCVExecutor) Stop(ctx context.Context, w workload.Workload) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	service, exists := e.services[w.GetID()]
	if !exists {
		return fmt.Errorf("service not found for workload: %s", w.GetID())
	}

	e.logger.WithField("workload_id", w.GetID()).Info("Stopping OpenCV service")

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
			e.logger.WithError(err).Warn("Failed to kill process")
		}
		service.process.Wait() // 等待进程退出
	}

	// 清理
	delete(e.services, w.GetID())
	w.SetStatus(workload.StatusStopped)

	return nil
}

// GetLogs 获取OpenCV服务日志
func (e *OpenCVExecutor) GetLogs(ctx context.Context, w workload.Workload, lines int) ([]string, error) {
	// TODO: 实现日志获取（从进程stdout/stderr）
	return []string{"OpenCV service logs (not implemented)"}, nil
}

// GetStatus 获取OpenCV服务状态
func (e *OpenCVExecutor) GetStatus(ctx context.Context, w workload.Workload) (workload.WorkloadStatus, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	service, exists := e.services[w.GetID()]
	if !exists {
		return workload.StatusStopped, nil
	}

	// 检查进程是否还在运行
	if service.process != nil && service.process.Process != nil {
		// 检查进程状态
		if err := service.process.Process.Signal(nil); err != nil {
			// 进程已不存在
			return workload.StatusFailed, nil
		}
		return workload.StatusRunning, nil
	}

	return w.GetStatus(), nil
}

// GetInferenceEndpoint 获取推理服务endpoint
func (e *OpenCVExecutor) GetInferenceEndpoint(workloadID string) (string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	service, exists := e.services[workloadID]
	if !exists {
		return "", fmt.Errorf("service not found: %s", workloadID)
	}

	return service.endpoint, nil
}

// HealthCheck 检查推理服务健康状态
func (e *OpenCVExecutor) HealthCheck(ctx context.Context, workloadID string) error {
	e.mu.RLock()
	service, exists := e.services[workloadID]
	e.mu.RUnlock()

	if !exists {
		return fmt.Errorf("service not found: %s", workloadID)
	}

	return e.checkServiceHealth(service)
}
