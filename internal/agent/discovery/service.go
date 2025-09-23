package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"cnet/internal/config"
	"cnet/internal/logger"
)

// Service represents the discovery service
type Service struct {
	config *config.Config
	logger *logger.Logger
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
	nodes  map[string]*Node
	client *http.Client
}

// Node represents a discovered node
type Node struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Address    string            `json:"address"`
	Port       int               `json:"port"`
	Region     string            `json:"region"`
	Datacenter string            `json:"datacenter"`
	Status     string            `json:"status"`
	LastSeen   time.Time         `json:"last_seen"`
	Metadata   map[string]string `json:"metadata"`
}

// RegisterRequest represents a node registration request
type RegisterRequest struct {
	Node Node `json:"node"`
}

// DeregisterRequest represents a node deregistration request
type DeregisterRequest struct {
	NodeID string `json:"node_id"`
}

// New creates a new discovery service
func New(cfg *config.Config, log *logger.Logger) (*Service, error) {
	ctx, cancel := context.WithCancel(context.Background())

	service := &Service{
		config: cfg,
		logger: log,
		ctx:    ctx,
		cancel: cancel,
		nodes:  make(map[string]*Node),
		client: &http.Client{
			Timeout: cfg.Discovery.Timeout,
		},
	}

	return service, nil
}

// Start starts the discovery service
func (s *Service) Start(ctx context.Context) error {
	if !s.config.Discovery.Enabled {
		s.logger.Info("Discovery service disabled")
		return nil
	}

	// Register this node with discovery servers
	if err := s.registerSelf(); err != nil {
		s.logger.Errorf("Failed to register self: %v", err)
	}

	// Start discovery loop
	go s.discoveryLoop()

	return nil
}

// Stop stops the discovery service
func (s *Service) Stop() error {
	if !s.config.Discovery.Enabled {
		return nil
	}

	// Deregister this node
	if err := s.deregisterSelf(); err != nil {
		s.logger.Errorf("Failed to deregister self: %v", err)
	}

	s.cancel()
	return nil
}

// RegisterNode registers a node with the discovery service
func (s *Service) RegisterNode(req *RegisterRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	node := &req.Node
	node.LastSeen = time.Now()
	node.Status = "active"

	s.nodes[node.ID] = node
	s.logger.Infof("Registered node: %s (%s)", node.Name, node.ID)

	return nil
}

// DeregisterNode deregisters a node from the discovery service
func (s *Service) DeregisterNode(req *DeregisterRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if node, exists := s.nodes[req.NodeID]; exists {
		node.Status = "inactive"
		s.logger.Infof("Deregistered node: %s (%s)", node.Name, node.ID)
	}

	return nil
}

// ListNodes returns all discovered nodes
func (s *Service) ListNodes() ([]*Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nodes := make([]*Node, 0, len(s.nodes))
	for _, node := range s.nodes {
		nodes = append(nodes, node)
	}

	return nodes, nil
}

// GetNode retrieves a specific node
func (s *Service) GetNode(nodeID string) (*Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	node, exists := s.nodes[nodeID]
	if !exists {
		return nil, fmt.Errorf("node not found")
	}

	return node, nil
}

// registerSelf registers this node with discovery servers
func (s *Service) registerSelf() error {
	node := Node{
		ID:         s.config.Agent.NodeID,
		Name:       s.config.Agent.NodeName,
		Address:    s.config.Agent.Address,
		Port:       s.config.Agent.Port,
		Region:     s.config.Agent.Region,
		Datacenter: s.config.Agent.Datacenter,
		Status:     "active",
		LastSeen:   time.Now(),
		Metadata: map[string]string{
			"version": "1.0.0",
		},
	}

	req := RegisterRequest{Node: node}

	// Try to register with each discovery server
	for _, server := range s.config.Discovery.Servers {
		if err := s.registerWithServer(server, &req); err != nil {
			s.logger.Errorf("Failed to register with server %s: %v", server, err)
			continue
		}
		s.logger.Infof("Successfully registered with server: %s", server)
		return nil
	}

	return fmt.Errorf("failed to register with any discovery server")
}

// deregisterSelf deregisters this node from discovery servers
func (s *Service) deregisterSelf() error {
	req := DeregisterRequest{NodeID: s.config.Agent.NodeID}

	// Try to deregister from each discovery server
	for _, server := range s.config.Discovery.Servers {
		if err := s.deregisterFromServer(server, &req); err != nil {
			s.logger.Errorf("Failed to deregister from server %s: %v", server, err)
			continue
		}
		s.logger.Infof("Successfully deregistered from server: %s", server)
		return nil
	}

	return fmt.Errorf("failed to deregister from any discovery server")
}

// registerWithServer registers with a specific discovery server
func (s *Service) registerWithServer(server string, req *RegisterRequest) error {
	url := fmt.Sprintf("http://%s/discovery/register", server)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := s.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	return nil
}

// deregisterFromServer deregisters from a specific discovery server
func (s *Service) deregisterFromServer(server string, req *DeregisterRequest) error {
	url := fmt.Sprintf("http://%s/discovery/deregister", server)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := s.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	return nil
}

// discoveryLoop runs the discovery loop
func (s *Service) discoveryLoop() {
	ticker := time.NewTicker(s.config.Agent.Heartbeat)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			// Update last seen for this node
			s.mu.Lock()
			if node, exists := s.nodes[s.config.Agent.NodeID]; exists {
				node.LastSeen = time.Now()
			}
			s.mu.Unlock()

			// Clean up stale nodes
			s.cleanupStaleNodes()
		}
	}
}

// cleanupStaleNodes removes nodes that haven't been seen recently
func (s *Service) cleanupStaleNodes() {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-s.config.Agent.Heartbeat * 3)

	for id, node := range s.nodes {
		if node.LastSeen.Before(cutoff) {
			node.Status = "stale"
			s.logger.Infof("Marked node as stale: %s (%s)", node.Name, id)
		}
	}
}
