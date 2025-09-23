package resources

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cnet/internal/config"
	"cnet/internal/logger"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// Service represents the resources monitoring service
type Service struct {
	config     *config.Config
	logger     *logger.Logger
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.RWMutex
	resources  *ResourceInfo
	usage      *UsageInfo
	lastUpdate time.Time
}

// ResourceInfo represents the node's resource information
type ResourceInfo struct {
	CPU     CPUInfo     `json:"cpu"`
	Memory  MemoryInfo  `json:"memory"`
	Disk    DiskInfo    `json:"disk"`
	Network NetworkInfo `json:"network"`
}

// CPUInfo represents CPU information
type CPUInfo struct {
	Count     int     `json:"count"`
	ModelName string  `json:"model_name"`
	Mhz       float64 `json:"mhz"`
}

// MemoryInfo represents memory information
type MemoryInfo struct {
	Total     uint64 `json:"total"`
	Available uint64 `json:"available"`
	Used      uint64 `json:"used"`
	Free      uint64 `json:"free"`
}

// DiskInfo represents disk information
type DiskInfo struct {
	Total uint64 `json:"total"`
	Used  uint64 `json:"used"`
	Free  uint64 `json:"free"`
}

// NetworkInfo represents network information
type NetworkInfo struct {
	Interfaces []InterfaceInfo `json:"interfaces"`
}

// InterfaceInfo represents network interface information
type InterfaceInfo struct {
	Name         string   `json:"name"`
	MTU          int      `json:"mtu"`
	HardwareAddr string   `json:"hardware_addr"`
	Flags        []string `json:"flags"`
}

// UsageInfo represents current resource usage
type UsageInfo struct {
	CPU       CPUUsage     `json:"cpu"`
	Memory    MemoryUsage  `json:"memory"`
	Disk      DiskUsage    `json:"disk"`
	Network   NetworkUsage `json:"network"`
	Timestamp time.Time    `json:"timestamp"`
}

// CPUUsage represents CPU usage
type CPUUsage struct {
	Percent float64   `json:"percent"`
	LoadAvg []float64 `json:"load_avg"`
}

// MemoryUsage represents memory usage
type MemoryUsage struct {
	Percent   float64 `json:"percent"`
	Used      uint64  `json:"used"`
	Available uint64  `json:"available"`
}

// DiskUsage represents disk usage
type DiskUsage struct {
	Percent float64 `json:"percent"`
	Used    uint64  `json:"used"`
	Free    uint64  `json:"free"`
}

// NetworkUsage represents network usage
type NetworkUsage struct {
	BytesSent   uint64 `json:"bytes_sent"`
	BytesRecv   uint64 `json:"bytes_recv"`
	PacketsSent uint64 `json:"packets_sent"`
	PacketsRecv uint64 `json:"packets_recv"`
}

// New creates a new resources service
func New(cfg *config.Config, log *logger.Logger) (*Service, error) {
	ctx, cancel := context.WithCancel(context.Background())

	service := &Service{
		config: cfg,
		logger: log,
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize resource information
	if err := service.updateResourceInfo(); err != nil {
		return nil, fmt.Errorf("failed to initialize resource info: %w", err)
	}

	return service, nil
}

// Start starts the resources monitoring service
func (s *Service) Start(ctx context.Context) error {
	go s.monitorLoop()
	return nil
}

// Stop stops the resources monitoring service
func (s *Service) Stop() error {
	s.cancel()
	return nil
}

// GetResources returns the node's resource information
func (s *Service) GetResources() (*ResourceInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.resources == nil {
		return nil, fmt.Errorf("resources not initialized")
	}

	return s.resources, nil
}

// GetUsage returns the current resource usage
func (s *Service) GetUsage() (*UsageInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.usage == nil {
		return nil, fmt.Errorf("usage not available")
	}

	return s.usage, nil
}

// monitorLoop runs the resource monitoring loop
func (s *Service) monitorLoop() {
	ticker := time.NewTicker(s.config.Resources.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			if err := s.updateUsage(); err != nil {
				s.logger.Errorf("Failed to update usage: %v", err)
			}
		}
	}
}

// updateResourceInfo updates the static resource information
func (s *Service) updateResourceInfo() error {
	// Get CPU info
	cpuInfo, err := cpu.Info()
	if err != nil {
		return fmt.Errorf("failed to get CPU info: %w", err)
	}

	// Get memory info
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return fmt.Errorf("failed to get memory info: %w", err)
	}

	// Get disk info
	diskInfo, err := disk.Usage("/")
	if err != nil {
		return fmt.Errorf("failed to get disk info: %w", err)
	}

	// Get network interfaces
	netInterfaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("failed to get network interfaces: %w", err)
	}

	interfaces := make([]InterfaceInfo, len(netInterfaces))
	for i, iface := range netInterfaces {
		interfaces[i] = InterfaceInfo{
			Name:         iface.Name,
			MTU:          iface.MTU,
			HardwareAddr: iface.HardwareAddr,
			Flags:        iface.Flags,
		}
	}

	s.mu.Lock()
	s.resources = &ResourceInfo{
		CPU: CPUInfo{
			Count:     len(cpuInfo),
			ModelName: cpuInfo[0].ModelName,
			Mhz:       cpuInfo[0].Mhz,
		},
		Memory: MemoryInfo{
			Total:     memInfo.Total,
			Available: memInfo.Available,
			Used:      memInfo.Used,
			Free:      memInfo.Free,
		},
		Disk: DiskInfo{
			Total: diskInfo.Total,
			Used:  diskInfo.Used,
			Free:  diskInfo.Free,
		},
		Network: NetworkInfo{
			Interfaces: interfaces,
		},
	}
	s.mu.Unlock()

	return nil
}

// updateUsage updates the current resource usage
func (s *Service) updateUsage() error {
	// Get CPU usage
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err != nil {
		return fmt.Errorf("failed to get CPU usage: %w", err)
	}

	// Get load average (simplified for now)
	loadAvg := struct {
		Load1  float64
		Load5  float64
		Load15 float64
	}{
		Load1:  0.0,
		Load5:  0.0,
		Load15: 0.0,
	}

	// Get memory usage
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return fmt.Errorf("failed to get memory usage: %w", err)
	}

	// Get disk usage
	diskInfo, err := disk.Usage("/")
	if err != nil {
		return fmt.Errorf("failed to get disk usage: %w", err)
	}

	// Get network usage
	netIO, err := net.IOCounters(false)
	if err != nil {
		return fmt.Errorf("failed to get network usage: %w", err)
	}

	var networkUsage NetworkUsage
	if len(netIO) > 0 {
		networkUsage = NetworkUsage{
			BytesSent:   netIO[0].BytesSent,
			BytesRecv:   netIO[0].BytesRecv,
			PacketsSent: netIO[0].PacketsSent,
			PacketsRecv: netIO[0].PacketsRecv,
		}
	}

	s.mu.Lock()
	s.usage = &UsageInfo{
		CPU: CPUUsage{
			Percent: cpuPercent[0],
			LoadAvg: []float64{loadAvg.Load1, loadAvg.Load5, loadAvg.Load15},
		},
		Memory: MemoryUsage{
			Percent:   memInfo.UsedPercent,
			Used:      memInfo.Used,
			Available: memInfo.Available,
		},
		Disk: DiskUsage{
			Percent: diskInfo.UsedPercent,
			Used:    diskInfo.Used,
			Free:    diskInfo.Free,
		},
		Network:   networkUsage,
		Timestamp: time.Now(),
	}
	s.lastUpdate = time.Now()
	s.mu.Unlock()

	return nil
}
