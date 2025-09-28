package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cnet/internal/config"
	httpclient "cnet/internal/http"
	"cnet/internal/logger"
)

const (
	// RootDiscoveryServerID is the node ID for the root discovery server
	RootDiscoveryServerID = "discovery-server"
	// RootHierarchyID is the hierarchy ID for the root discovery server
	RootHierarchyID = "34.23.1"
	// RootHierarchyLevel is the hierarchy level for the root discovery server
	RootHierarchyLevel = 3
)

// Service represents the discovery service
type Service struct {
	config *config.Config
	logger *logger.Logger
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
	nodes  map[string]*Node
	client *httpclient.Client
}

// Node represents a discovered node
type Node struct {
	ID          string            `json:"id"`           // 原始节点ID
	HierarchyID string            `json:"hierarchy_id"` // 层次化标识 (如 34.23.1.8)
	Name        string            `json:"name"`
	Address     string            `json:"address"`
	Port        int               `json:"port"`
	Region      string            `json:"region"`
	Datacenter  string            `json:"datacenter"`
	Status      string            `json:"status"`
	LastSeen    time.Time         `json:"last_seen"`
	ParentID    string            `json:"parent_id"` // 父节点ID
	Level       int               `json:"level"`     // 层次级别
	Metadata    map[string]string `json:"metadata"`
}

// RegisterRequest represents a node registration request
type RegisterRequest struct {
	Node Node `json:"node"`
}

// DeregisterRequest represents a node deregistration request
type DeregisterRequest struct {
	NodeID string `json:"node_id"`
}

// HierarchyAssignRequest represents a hierarchy ID assignment request
type HierarchyAssignRequest struct {
	NodeID string `json:"node_id"`
}

// HierarchyAssignResponse represents a hierarchy ID assignment response
type HierarchyAssignResponse struct {
	HierarchyID string `json:"hierarchy_id"`
	Level       int    `json:"level"`
	ParentID    string `json:"parent_id"`
}

// ResolveRequest represents a hierarchy ID resolution request
type ResolveRequest struct {
	HierarchyID string `json:"hierarchy_id"`
}

// ResolveResponse represents a hierarchy ID resolution response
type ResolveResponse struct {
	NodeID   string `json:"node_id"`
	Address  string `json:"address"`
	Port     int    `json:"port"`
	Status   string `json:"status"`
	LastSeen string `json:"last_seen"`
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
		client: httpclient.NewClient(cfg.Discovery.Timeout),
	}

	return service, nil
}

// Start starts the discovery service
func (s *Service) Start(ctx context.Context) error {
	// Always start discovery server functionality
	s.logger.Info("Discovery service started")

	// Initialize current node in local nodes map
	s.initializeCurrentNode()

	// If discovery is enabled, register this node with discovery servers
	if s.config.Discovery.Enabled {
		if err := s.registerSelf(); err != nil {
			s.logger.Errorf("Failed to register self: %v", err)
		}
	}

	// Start discovery loop
	go s.discoveryLoop()

	return nil
}

// Stop stops the discovery service
func (s *Service) Stop() error {
	// If discovery is enabled, deregister this node
	if s.config.Discovery.Enabled {
		if err := s.deregisterSelf(); err != nil {
			s.logger.Errorf("Failed to deregister self: %v", err)
		}
	}

	s.cancel()
	return nil
}

// RegisterNode registers a node with the discovery service
func (s *Service) RegisterNode(req *RegisterRequest) error {
	// Validate input
	if req == nil {
		return fmt.Errorf("register request cannot be nil")
	}

	node := &req.Node
	if node.ID == "" {
		return fmt.Errorf("node ID cannot be empty")
	}
	if node.Name == "" {
		return fmt.Errorf("node name cannot be empty")
	}
	if node.Address == "" {
		return fmt.Errorf("node address cannot be empty")
	}
	if node.Port <= 0 || node.Port > 65535 {
		return fmt.Errorf("node port must be between 1 and 65535, got %d", node.Port)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if node already exists
	if existingNode, exists := s.nodes[node.ID]; exists {
		s.logger.Warnf("Node %s already exists, updating registration", node.ID)
		// Update existing node
		existingNode.Name = node.Name
		existingNode.Address = node.Address
		existingNode.Port = node.Port
		existingNode.Region = node.Region
		existingNode.Datacenter = node.Datacenter
		existingNode.LastSeen = time.Now()
		existingNode.Status = "active"
		if node.Metadata != nil {
			existingNode.Metadata = node.Metadata
		}
		s.logger.Infof("Updated existing node: %s (%s)", existingNode.Name, existingNode.ID)
		return nil
	}

	node.LastSeen = time.Now()
	node.Status = "active"

	// 如果节点没有层次化标识，自动分配一个
	if node.HierarchyID == "" {
		hierarchyID, level, parentID := s.assignHierarchyIDUnsafe(node.ID)
		node.HierarchyID = hierarchyID
		node.Level = level
		node.ParentID = parentID
	}

	s.nodes[node.ID] = node
	s.logger.Infof("Registered node: %s (%s) with hierarchy ID: %s", node.Name, node.ID, node.HierarchyID)

	return nil
}

// DeregisterNode deregisters a node from the discovery service
func (s *Service) DeregisterNode(req *DeregisterRequest) error {
	// Validate input
	if req == nil {
		return fmt.Errorf("deregister request cannot be nil")
	}
	if req.NodeID == "" {
		return fmt.Errorf("node ID cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if node, exists := s.nodes[req.NodeID]; exists {
		node.Status = "inactive"
		s.logger.Infof("Deregistered node: %s (%s)", node.Name, node.ID)
	} else {
		s.logger.Warnf("Attempted to deregister non-existent node: %s", req.NodeID)
		return fmt.Errorf("node not found: %s", req.NodeID)
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

		// After successful registration, get the assigned hierarchy info
		if err := s.syncHierarchyInfoFromServer(server); err != nil {
			s.logger.Warnf("Failed to sync hierarchy info from server %s: %v", server, err)
		}
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
	return s.client.PostJSON(url, req, nil)
}

// deregisterFromServer deregisters from a specific discovery server
func (s *Service) deregisterFromServer(server string, req *DeregisterRequest) error {
	url := fmt.Sprintf("http://%s/discovery/deregister", server)
	return s.client.PostJSON(url, req, nil)
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
			s.performHeartbeat()
		}
	}
}

// performHeartbeat performs heartbeat operations
func (s *Service) performHeartbeat() {
	now := time.Now()
	cutoff := now.Add(-s.config.Agent.Heartbeat * 3)

	s.mu.Lock()
	defer s.mu.Unlock()

	// Update last seen for this node
	if node, exists := s.nodes[s.config.Agent.NodeID]; exists {
		node.LastSeen = now
	}

	// Clean up stale nodes
	staleCount := 0
	for id, node := range s.nodes {
		if node.LastSeen.Before(cutoff) && node.Status != "stale" {
			node.Status = "stale"
			staleCount++
			s.logger.Infof("Marked node as stale: %s (%s)", node.Name, id)
		}
	}

	if staleCount > 0 {
		s.logger.Infof("Marked %d nodes as stale", staleCount)
	}
}

// assignHierarchyID assigns a hierarchical ID to a node
func (s *Service) assignHierarchyID(nodeID string) (string, int, string) {
	// 获取当前节点的层次化标识
	currentNodeID := s.config.Agent.NodeID
	currentHierarchyID := s.getCurrentNodeHierarchyID()

	// 计算下一级标识
	nextID := s.getNextChildID(currentHierarchyID)
	level := s.calculateLevel(nextID)

	s.logger.Infof("Assigning hierarchy ID: %s -> %s (level %d, parent: %s)", nodeID, nextID, level, currentNodeID)
	return nextID, level, currentNodeID
}

// assignHierarchyIDUnsafe assigns a hierarchical ID to a node without locking
// This should only be called when the caller already holds the lock
func (s *Service) assignHierarchyIDUnsafe(nodeID string) (string, int, string) {
	// 获取当前节点的层次化标识
	currentNodeID := s.config.Agent.NodeID
	currentHierarchyID := s.getCurrentNodeHierarchyIDUnsafe()

	// 计算下一级标识
	nextID := s.getNextChildIDUnsafe(currentHierarchyID)
	level := s.calculateLevel(nextID)

	s.logger.Infof("Assigning hierarchy ID: %s -> %s (level %d, parent: %s)", nodeID, nextID, level, currentNodeID)
	return nextID, level, currentNodeID
}

// getCurrentNodeHierarchyID returns the current node's hierarchy ID
func (s *Service) getCurrentNodeHierarchyID() string {
	// 如果当前节点没有层次化标识，使用节点ID作为根标识
	if s.config.Agent.NodeID == RootDiscoveryServerID {
		return RootHierarchyID // 根节点标识
	}

	// 从本地存储的节点信息中获取层次化标识
	s.mu.RLock()
	defer s.mu.RUnlock()

	if node, exists := s.nodes[s.config.Agent.NodeID]; exists {
		// 如果节点已经有层次化标识，使用它
		if node.HierarchyID != "" && node.HierarchyID != s.config.Agent.NodeID {
			return node.HierarchyID
		}
	}

	// 如果本地没有找到或层次化标识无效，返回节点ID作为fallback
	return s.config.Agent.NodeID
}

// getCurrentNodeHierarchyIDUnsafe returns the current node's hierarchy ID without locking
// This should only be called when the caller already holds the lock
func (s *Service) getCurrentNodeHierarchyIDUnsafe() string {
	// 如果当前节点没有层次化标识，使用节点ID作为根标识
	if s.config.Agent.NodeID == RootDiscoveryServerID {
		return RootHierarchyID // 根节点标识
	}

	// 从本地存储的节点信息中获取层次化标识（不需要锁，调用者已持有锁）
	if node, exists := s.nodes[s.config.Agent.NodeID]; exists {
		// 如果节点已经有层次化标识，使用它
		if node.HierarchyID != "" && node.HierarchyID != s.config.Agent.NodeID {
			return node.HierarchyID
		}
	}

	// 如果本地没有找到或层次化标识无效，返回节点ID作为fallback
	return s.config.Agent.NodeID
}

// getNextChildID generates the next child hierarchy ID
func (s *Service) getNextChildID(parentHierarchyID string) string {
	// 计算当前父节点下已有的子节点数量
	childCount := s.countChildNodes(parentHierarchyID)

	// 生成下一级标识，格式：父标识.子序号
	return fmt.Sprintf("%s.%d", parentHierarchyID, childCount+1)
}

// getNextChildIDUnsafe generates the next child hierarchy ID without locking
// This should only be called when the caller already holds the lock
func (s *Service) getNextChildIDUnsafe(parentHierarchyID string) string {
	// 计算当前父节点下已有的子节点数量
	childCount := s.countChildNodesUnsafe(parentHierarchyID)

	// 生成下一级标识，格式：父标识.子序号
	return fmt.Sprintf("%s.%d", parentHierarchyID, childCount+1)
}

// countChildNodes counts the number of child nodes under a parent hierarchy ID
func (s *Service) countChildNodes(parentHierarchyID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, node := range s.nodes {
		// 检查节点是否是当前节点的直接子节点
		if node.ParentID == s.config.Agent.NodeID &&
			len(node.HierarchyID) > len(parentHierarchyID) &&
			node.HierarchyID[:len(parentHierarchyID)] == parentHierarchyID {
			count++
		}
	}
	s.logger.Debugf("Counted %d child nodes for parent hierarchy ID: %s", count, parentHierarchyID)
	return count
}

// countChildNodesUnsafe counts the number of child nodes under a parent hierarchy ID without locking
// This should only be called when the caller already holds the lock
func (s *Service) countChildNodesUnsafe(parentHierarchyID string) int {
	count := 0
	for _, node := range s.nodes {
		// 检查节点是否是当前节点的直接子节点
		if node.ParentID == s.config.Agent.NodeID &&
			len(node.HierarchyID) > len(parentHierarchyID) &&
			node.HierarchyID[:len(parentHierarchyID)] == parentHierarchyID {
			count++
		}
	}
	s.logger.Debugf("Counted %d child nodes for parent hierarchy ID: %s", count, parentHierarchyID)
	return count
}

// calculateLevel calculates the hierarchy level from a hierarchy ID
func (s *Service) calculateLevel(hierarchyID string) int {
	// 计算层次级别：通过点号分隔符的数量
	// 使用strings.Count更高效
	level := 1
	for i := 0; i < len(hierarchyID); i++ {
		if hierarchyID[i] == '.' {
			level++
		}
	}
	return level
}

// AssignHierarchyID assigns a hierarchy ID to a specific node
func (s *Service) AssignHierarchyID(req *HierarchyAssignRequest) (*HierarchyAssignResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	node, exists := s.nodes[req.NodeID]
	if !exists {
		return nil, fmt.Errorf("node not found: %s", req.NodeID)
	}

	// 分配新的层次化标识
	hierarchyID, level, parentID := s.assignHierarchyIDUnsafe(req.NodeID)
	node.HierarchyID = hierarchyID
	node.Level = level
	node.ParentID = parentID

	s.logger.Infof("Assigned hierarchy ID %s to node %s", hierarchyID, req.NodeID)

	return &HierarchyAssignResponse{
		HierarchyID: hierarchyID,
		Level:       level,
		ParentID:    parentID,
	}, nil
}

// ResolveHierarchyID resolves a hierarchy ID to node information
func (s *Service) ResolveHierarchyID(req *ResolveRequest) (*ResolveResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 查找具有指定层次化标识的节点
	for _, node := range s.nodes {
		if node.HierarchyID == req.HierarchyID {
			return &ResolveResponse{
				NodeID:   node.ID,
				Address:  node.Address,
				Port:     node.Port,
				Status:   node.Status,
				LastSeen: node.LastSeen.Format(time.RFC3339),
			}, nil
		}
	}

	return nil, fmt.Errorf("hierarchy ID not found: %s", req.HierarchyID)
}

// syncHierarchyInfoFromServer syncs hierarchy information from a discovery server
func (s *Service) syncHierarchyInfoFromServer(server string) error {
	url := fmt.Sprintf("http://%s/discovery/nodes", server)

	var nodes []Node
	if err := s.client.GetJSON(url, &nodes); err != nil {
		return fmt.Errorf("failed to get nodes from server: %w", err)
	}

	// Find current node and update local hierarchy info
	currentNodeID := s.config.Agent.NodeID
	for _, node := range nodes {
		if node.ID == currentNodeID {
			// Update the existing node with hierarchy info from server
			s.mu.Lock()
			if existingNode, exists := s.nodes[node.ID]; exists {
				existingNode.HierarchyID = node.HierarchyID
				existingNode.Level = node.Level
				existingNode.ParentID = node.ParentID
				existingNode.Status = node.Status
				existingNode.LastSeen = node.LastSeen
			} else {
				// If node doesn't exist locally, add it
				s.nodes[node.ID] = &node
			}
			s.mu.Unlock()

			s.logger.Infof("Synced hierarchy info: %s -> %s (level %d, parent: %s)", node.ID, node.HierarchyID, node.Level, node.ParentID)
			return nil
		}
	}

	return fmt.Errorf("current node not found in server response")
}

// ListNodesByHierarchy returns nodes organized by hierarchy
func (s *Service) ListNodesByHierarchy() (map[string][]*Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	hierarchy := make(map[string][]*Node)

	for _, node := range s.nodes {
		parentID := node.ParentID
		if parentID == "" {
			parentID = "root"
		}
		hierarchy[parentID] = append(hierarchy[parentID], node)
	}

	return hierarchy, nil
}

// initializeCurrentNode initializes the current node in the local nodes map
func (s *Service) initializeCurrentNode() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create current node entry
	currentNode := &Node{
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

	// Set hierarchy information based on node type
	if s.config.Agent.NodeID == RootDiscoveryServerID {
		// Root discovery server
		currentNode.HierarchyID = RootHierarchyID
		currentNode.Level = RootHierarchyLevel
		currentNode.ParentID = ""
	} else {
		// Worker node - will be updated after registration
		currentNode.HierarchyID = s.config.Agent.NodeID
		currentNode.Level = 1
		currentNode.ParentID = ""
	}

	s.nodes[s.config.Agent.NodeID] = currentNode
	s.logger.Infof("Initialized current node: %s with hierarchy ID: %s", currentNode.Name, currentNode.HierarchyID)
}
