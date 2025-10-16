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

// DataGatewayExecutor 提供只读S3子集的网关服务
type DataGatewayExecutor struct {
	logger  *logrus.Logger
	mu      sync.RWMutex
	servers map[string]*gatewayServer // workload_id -> server
}

type gatewayServer struct {
	workloadID string
	endpoint   string
	process    *exec.Cmd
	port       int
	basePath   string
	bucket     string
	authToken  string
}

func NewDataGatewayExecutor(logger *logrus.Logger) *DataGatewayExecutor {
	return &DataGatewayExecutor{logger: logger, servers: make(map[string]*gatewayServer)}
}

func (e *DataGatewayExecutor) Init(ctx context.Context) error {
	e.logger.Info("DataGatewayExecutor initialized")
	return nil
}

func (e *DataGatewayExecutor) Execute(ctx context.Context, w workload.Workload) error {
	gw, ok := w.(*workload.DataGatewayWorkload)
	if !ok {
		return fmt.Errorf("invalid workload type for DataGatewayExecutor")
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	if _, exists := e.servers[w.GetID()]; exists {
		return fmt.Errorf("gateway already running: %s", w.GetID())
	}

	// 启动独立子进程 ./bin/cnet-gateway-data
	cmd := exec.Command(
		"./bin/cnet-gateway-data",
		"--base-path", gw.BasePath,
		"--bucket", gw.Bucket,
		"--port", fmt.Sprintf("%d", gw.ServicePort),
		"--host", gw.ServiceHost,
	)
	if gw.AuthToken != "" {
		cmd.Args = append(cmd.Args, "--auth-token", gw.AuthToken)
	}
	logFile := fmt.Sprintf("data_gateway_%d.log", gw.ServicePort)
	lf, err := os.Create(logFile)
	if err != nil {
		return fmt.Errorf("create log file: %w", err)
	}
	cmd.Stdout = lf
	cmd.Stderr = lf
	cmd.Dir = "."
	if err := cmd.Start(); err != nil {
		lf.Close()
		return fmt.Errorf("start gateway: %w", err)
	}

	endpoint := fmt.Sprintf("http://%s:%d", gw.ServiceHost, gw.ServicePort)
	if err := e.waitHealthy(endpoint, 30*time.Second); err != nil {
		_ = cmd.Process.Kill()
		lf.Close()
		return fmt.Errorf("gateway not healthy: %w", err)
	}

	gs := &gatewayServer{
		workloadID: w.GetID(),
		endpoint:   endpoint,
		process:    cmd,
		port:       gw.ServicePort,
		basePath:   gw.BasePath,
		bucket:     gw.Bucket,
		authToken:  gw.AuthToken,
	}
	e.servers[w.GetID()] = gs

	gw.Endpoint = endpoint
	gw.ProcessPID = cmd.Process.Pid
	gw.SetStatus(workload.StatusRunning)

	e.logger.WithFields(logrus.Fields{"workload_id": w.GetID(), "endpoint": gw.Endpoint, "pid": gw.ProcessPID}).Info("Data gateway process started")
	return nil
}

func (e *DataGatewayExecutor) Stop(ctx context.Context, w workload.Workload) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	gs, ok := e.servers[w.GetID()]
	if !ok {
		return fmt.Errorf("gateway not found: %s", w.GetID())
	}
	if gs.process != nil && gs.process.Process != nil {
		_ = gs.process.Process.Kill()
	}
	delete(e.servers, w.GetID())
	w.SetStatus(workload.StatusStopped)
	return nil
}

func (e *DataGatewayExecutor) GetLogs(ctx context.Context, w workload.Workload, lines int) ([]string, error) {
	return []string{"Data gateway running"}, nil
}

func (e *DataGatewayExecutor) GetStatus(ctx context.Context, w workload.Workload) (workload.WorkloadStatus, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if _, ok := e.servers[w.GetID()]; !ok {
		return workload.StatusStopped, nil
	}
	return workload.StatusRunning, nil
}

func (e *DataGatewayExecutor) GetEndpoint(workloadID string) (string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if gs, ok := e.servers[workloadID]; ok {
		return gs.endpoint, nil
	}
	return "", fmt.Errorf("gateway not found: %s", workloadID)
}

func (e *DataGatewayExecutor) HealthCheck(ctx context.Context, workloadID string) error {
	e.mu.RLock()
	gs, ok := e.servers[workloadID]
	e.mu.RUnlock()
	if !ok {
		return fmt.Errorf("gateway not found: %s", workloadID)
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, gs.endpoint+"/health", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unhealthy: %d", resp.StatusCode)
	}
	return nil
}

func (e *DataGatewayExecutor) waitHealthy(endpoint string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(endpoint + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for health")
}
