package workload

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ProcessWorkload 进程workload
type ProcessWorkload struct {
	BaseWorkload
	Command    string            `json:"command"`
	Args       []string          `json:"args"`
	Env        map[string]string `json:"env"`
	WorkingDir string            `json:"working_dir"`
	PID        int               `json:"pid,omitempty"`
	ExitCode   *int              `json:"exit_code,omitempty"`
}

// NewProcessWorkload 创建进程workload
func NewProcessWorkload(name, command string, req CreateWorkloadRequest) *ProcessWorkload {
	now := time.Now()

	workload := &ProcessWorkload{
		BaseWorkload: BaseWorkload{
			ID:           uuid.New().String(),
			Name:         name,
			Type:         TypeProcess,
			Status:       StatusPending,
			Requirements: req.Requirements,
			CreatedAt:    now,
			UpdatedAt:    now,
			Metadata:     req.Config,
		},
		Command: command,
		Env:     make(map[string]string),
	}

	// 从config中提取进程特定配置
	if req.Config != nil {
		if args, ok := req.Config["args"].([]interface{}); ok {
			workload.Args = make([]string, len(args))
			for i, v := range args {
				workload.Args[i] = fmt.Sprint(v)
			}
		}

		if env, ok := req.Config["env"].(map[string]interface{}); ok {
			for k, v := range env {
				workload.Env[k] = fmt.Sprint(v)
			}
		}

		if workdir, ok := req.Config["working_dir"].(string); ok {
			workload.WorkingDir = workdir
		}
	}

	return workload
}

// Validate 验证进程workload配置
func (w *ProcessWorkload) Validate() error {
	if w.Command == "" {
		return fmt.Errorf("command cannot be empty")
	}

	if err := w.Requirements.Validate(); err != nil {
		return fmt.Errorf("invalid resource requirements: %w", err)
	}

	return nil
}
