package agent

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"cnet/internal/agent/api"
	"cnet/internal/agent/discovery"
	"cnet/internal/agent/resources"
	"cnet/internal/agent/tasks"
	"cnet/internal/config"
	"cnet/internal/logger"
)

// Agent represents the CNET agent
type Agent struct {
	config     *config.Config
	logger     *logger.Logger
	discovery  *discovery.Service
	resources  *resources.Service
	tasks      *tasks.Service
	api        *api.Server
	httpServer *http.Server
	mu         sync.RWMutex
	running    bool
}

// New creates a new agent instance
func New(cfg *config.Config, log *logger.Logger) (*Agent, error) {
	// Create discovery service
	discoveryService, err := discovery.New(cfg, log)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery service: %w", err)
	}

	// Create resources service
	resourcesService, err := resources.New(cfg, log)
	if err != nil {
		return nil, fmt.Errorf("failed to create resources service: %w", err)
	}

	// Create tasks service
	tasksService, err := tasks.New(cfg, log)
	if err != nil {
		return nil, fmt.Errorf("failed to create tasks service: %w", err)
	}

	// Create API server
	apiServer := api.New(cfg, log, discoveryService, resourcesService, tasksService)

	agent := &Agent{
		config:    cfg,
		logger:    log,
		discovery: discoveryService,
		resources: resourcesService,
		tasks:     tasksService,
		api:       apiServer,
	}

	return agent, nil
}

// Start starts the agent services
func (a *Agent) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return fmt.Errorf("agent is already running")
	}

	// Start discovery service
	if a.config.Discovery.Enabled {
		if err := a.discovery.Start(ctx); err != nil {
			return fmt.Errorf("failed to start discovery service: %w", err)
		}
		a.logger.Info("Discovery service started")
	}

	// Start resources monitoring
	if err := a.resources.Start(ctx); err != nil {
		return fmt.Errorf("failed to start resources service: %w", err)
	}
	a.logger.Info("Resources service started")

	// Start tasks service
	if err := a.tasks.Start(ctx); err != nil {
		return fmt.Errorf("failed to start tasks service: %w", err)
	}
	a.logger.Info("Tasks service started")

	a.running = true
	return nil
}

// Stop stops the agent services
func (a *Agent) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return nil
	}

	// Stop services in reverse order
	if err := a.tasks.Stop(); err != nil {
		a.logger.Errorf("Failed to stop tasks service: %v", err)
	}

	if err := a.resources.Stop(); err != nil {
		a.logger.Errorf("Failed to stop resources service: %v", err)
	}

	if a.config.Discovery.Enabled {
		if err := a.discovery.Stop(); err != nil {
			a.logger.Errorf("Failed to stop discovery service: %v", err)
		}
	}

	a.running = false
	a.logger.Info("Agent stopped")
	return nil
}

// Handler returns the HTTP handler for the agent
func (a *Agent) Handler() http.Handler {
	return a.api.Handler()
}

// IsRunning returns whether the agent is running
func (a *Agent) IsRunning() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.running
}

// GetNodeInfo returns basic node information
func (a *Agent) GetNodeInfo() map[string]interface{} {
	return map[string]interface{}{
		"node_id":    a.config.Agent.NodeID,
		"node_name":  a.config.Agent.NodeName,
		"region":     a.config.Agent.Region,
		"datacenter": a.config.Agent.Datacenter,
		"address":    a.config.Agent.Address,
		"port":       a.config.Agent.Port,
		"running":    a.IsRunning(),
		"started_at": time.Now().Format(time.RFC3339),
	}
}
