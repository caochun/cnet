package workload

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ContainerWorkload 容器workload
type ContainerWorkload struct {
	BaseWorkload
	Image       string            `json:"image"`
	Command     []string          `json:"command"`
	Args        []string          `json:"args"`
	Env         map[string]string `json:"env"`
	Ports       []PortMapping     `json:"ports"`
	Volumes     []VolumeMount     `json:"volumes"`
	WorkingDir  string            `json:"working_dir"`
	ContainerID string            `json:"container_id,omitempty"`
}

// PortMapping 端口映射
type PortMapping struct {
	HostPort      int    `json:"host_port"`
	ContainerPort int    `json:"container_port"`
	Protocol      string `json:"protocol"` // "tcp" or "udp"
}

// VolumeMount 卷挂载
type VolumeMount struct {
	HostPath      string `json:"host_path"`
	ContainerPath string `json:"container_path"`
	ReadOnly      bool   `json:"read_only"`
}

// NewContainerWorkload 创建容器workload
func NewContainerWorkload(name, image string, req CreateWorkloadRequest) *ContainerWorkload {
	now := time.Now()

	workload := &ContainerWorkload{
		BaseWorkload: BaseWorkload{
			ID:           uuid.New().String(),
			Name:         name,
			Type:         TypeContainer,
			Status:       StatusPending,
			Requirements: req.Requirements,
			CreatedAt:    now,
			UpdatedAt:    now,
			Metadata:     req.Config,
		},
		Image: image,
		Env:   make(map[string]string),
	}

	// 从config中提取容器特定配置
	if req.Config != nil {
		if cmd, ok := req.Config["command"].([]interface{}); ok {
			workload.Command = make([]string, len(cmd))
			for i, v := range cmd {
				workload.Command[i] = fmt.Sprint(v)
			}
		}

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

// Validate 验证容器workload配置
func (w *ContainerWorkload) Validate() error {
	if w.Image == "" {
		return fmt.Errorf("container image cannot be empty")
	}

	if err := w.Requirements.Validate(); err != nil {
		return fmt.Errorf("invalid resource requirements: %w", err)
	}

	// 验证端口映射
	for _, port := range w.Ports {
		if port.HostPort <= 0 || port.HostPort > 65535 {
			return fmt.Errorf("invalid host port: %d", port.HostPort)
		}
		if port.ContainerPort <= 0 || port.ContainerPort > 65535 {
			return fmt.Errorf("invalid container port: %d", port.ContainerPort)
		}
		if port.Protocol != "tcp" && port.Protocol != "udp" {
			return fmt.Errorf("invalid protocol: %s (must be tcp or udp)", port.Protocol)
		}
	}

	return nil
}
