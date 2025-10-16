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

// YOLOInferenceExecutor YOLO模型推理服务执行器
// 管理YOLO HTTP推理服务进程
type YOLOInferenceExecutor struct {
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

// NewYOLOInferenceExecutor 创建YOLO执行器
func NewYOLOInferenceExecutor(logger *logrus.Logger) *YOLOInferenceExecutor {
	return &YOLOInferenceExecutor{
		logger:   logger,
		services: make(map[string]*YOLOService),
	}
}

// Init 初始化YOLO执行器
func (e *YOLOInferenceExecutor) Init(ctx context.Context) error {
	e.logger.Info("YOLO Executor initialized")
	return nil
}

// Execute 执行MLModel workload - 启动YOLO推理服务
func (e *YOLOInferenceExecutor) Execute(ctx context.Context, w workload.Workload) error {
	mw, ok := w.(*workload.MLModelWorkload)
	if !ok {
		return fmt.Errorf("invalid workload type, expected MLModelWorkload")
	}
	if mw.ModelType != "yolo" {
		return fmt.Errorf("invalid model type for YOLOExecutor: %s", mw.ModelType)
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	if _, exists := e.services[w.GetID()]; exists {
		return fmt.Errorf("YOLO service already running for workload: %s", w.GetID())
	}

	service, err := e.startService(ctx, mw)
	if err != nil {
		return fmt.Errorf("failed to start YOLO service: %w", err)
	}

	e.services[w.GetID()] = service
	mw.Endpoint = service.endpoint
	mw.ProcessPID = service.process.Process.Pid
	mw.SetStatus(workload.StatusRunning)

	e.logger.WithFields(logrus.Fields{
		"workload_id": w.GetID(),
		"model_path":  mw.ModelPath,
		"endpoint":    service.endpoint,
		"pid":         service.process.Process.Pid,
	}).Info("YOLO推理服务已启动")

	go e.startHealthCheck(service)
	return nil
}

func (e *YOLOInferenceExecutor) startService(ctx context.Context, mw *workload.MLModelWorkload) (*YOLOService, error) {
	serviceCtx, cancel := context.WithCancel(ctx)
	cmd := exec.Command(
		"./bin/cnet-inference-yolo",
		"--model", mw.ModelPath,
		"--port", fmt.Sprintf("%d", mw.ServicePort),
	)
	if mw.ModelConfig != "" {
		cmd.Args = append(cmd.Args, "--config", mw.ModelConfig)
	}
	cmd.Dir = "."
	logFile := fmt.Sprintf("yolo_service_%d.log", mw.ServicePort)
	outFile, err := os.Create(logFile)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}
	cmd.Stdout = outFile
	cmd.Stderr = outFile
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
	if err := e.waitForService(service, 60*time.Second); err != nil {
		cmd.Process.Kill()
		cancel()
		return nil, fmt.Errorf("service failed to start: %w", err)
	}
	return service, nil
}

func (e *YOLOInferenceExecutor) waitForService(service *YOLOService, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := e.checkServiceHealth(service); err == nil {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("service did not start within timeout")
}

func (e *YOLOInferenceExecutor) checkServiceHealth(service *YOLOService) error {
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

func (e *YOLOInferenceExecutor) startHealthCheck(service *YOLOService) {
	service.healthTicker = time.NewTicker(30 * time.Second)
	for {
		select {
		case <-service.ctx.Done():
			service.healthTicker.Stop()
			return
		case <-service.healthTicker.C:
			if err := e.checkServiceHealth(service); err != nil {
				e.logger.WithFields(logrus.Fields{"workload_id": service.workloadID, "error": err}).Warn("YOLO service health check failed, attempting restart")
				if err := e.restartService(service.workloadID); err != nil {
					e.logger.WithError(err).Error("Failed to restart YOLO service")
				}
			}
		}
	}
}

func (e *YOLOInferenceExecutor) restartService(workloadID string) error {
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
	e.logger.WithFields(logrus.Fields{"workload_id": workloadID, "restart_count": service.restartCount}).Info("Restarting YOLO service")
	if service.process != nil && service.process.Process != nil {
		service.process.Process.Kill()
	}
	return fmt.Errorf("restart not fully implemented")
}

func (e *YOLOInferenceExecutor) Stop(ctx context.Context, w workload.Workload) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	service, exists := e.services[w.GetID()]
	if !exists {
		return fmt.Errorf("service not found for workload: %s", w.GetID())
	}
	e.logger.WithField("workload_id", w.GetID()).Info("Stopping YOLO service")
	if service.healthTicker != nil {
		service.healthTicker.Stop()
	}
	if service.cancel != nil {
		service.cancel()
	}
	if service.process != nil && service.process.Process != nil {
		if err := service.process.Process.Kill(); err != nil {
			e.logger.WithError(err).Warn("Failed to kill YOLO process")
		}
		service.process.Wait()
	}
	delete(e.services, w.GetID())
	w.SetStatus(workload.StatusStopped)
	return nil
}

func (e *YOLOInferenceExecutor) GetLogs(ctx context.Context, w workload.Workload, lines int) ([]string, error) {
	return []string{"YOLO service logs (not implemented)"}, nil
}

func (e *YOLOInferenceExecutor) GetStatus(ctx context.Context, w workload.Workload) (workload.WorkloadStatus, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	service, exists := e.services[w.GetID()]
	if !exists {
		return workload.StatusStopped, nil
	}
	if service.process != nil && service.process.Process != nil {
		if err := service.process.Process.Signal(nil); err != nil {
			return workload.StatusFailed, nil
		}
		return workload.StatusRunning, nil
	}
	return w.GetStatus(), nil
}

func (e *YOLOInferenceExecutor) GetInferenceEndpoint(workloadID string) (string, error) {
	return e.GetEndpoint(workloadID)
}

func (e *YOLOInferenceExecutor) GetEndpoint(workloadID string) (string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	service, exists := e.services[workloadID]
	if !exists {
		return "", fmt.Errorf("service not found: %s", workloadID)
	}
	return service.endpoint, nil
}

func (e *YOLOInferenceExecutor) HealthCheck(ctx context.Context, workloadID string) error {
	e.mu.RLock()
	service, exists := e.services[workloadID]
	e.mu.RUnlock()
	if !exists {
		return fmt.Errorf("service not found: %s", workloadID)
	}
	return e.checkServiceHealth(service)
}
