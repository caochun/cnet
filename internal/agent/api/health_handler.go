package api

import (
	"net/http"
	"time"

	"cnet/internal/agent/discovery"
)

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"uptime":    time.Since(time.Now()).String(),
	}

	s.writeJSON(w, http.StatusOK, response)
}

// handleNodeInfo handles node information requests
func (s *Server) handleNodeInfo(w http.ResponseWriter, r *http.Request) {
	nodeInfo := map[string]interface{}{
		"node_id":    s.config.Agent.NodeID,
		"node_name":  s.config.Agent.NodeName,
		"region":     s.config.Agent.Region,
		"datacenter": s.config.Agent.Datacenter,
		"address":    s.config.Agent.Address,
		"port":       s.config.Agent.Port,
		"version":    "1.0.0",
		"started_at": time.Now().Format(time.RFC3339),
	}

	s.writeJSON(w, http.StatusOK, nodeInfo)
}

// handleNodeHierarchy handles node hierarchy information requests
func (s *Server) handleNodeHierarchy(w http.ResponseWriter, r *http.Request) {
	// Get current node's hierarchy information
	currentNodeID := s.config.Agent.NodeID

	// Try to find current node in the discovery service
	nodes, err := s.discovery.ListNodes()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to get nodes", err)
		return
	}

	var currentNode *discovery.Node
	for _, node := range nodes {
		if node.ID == currentNodeID {
			currentNode = node
			break
		}
	}

	// If current node is not found in discovery, create basic info
	if currentNode == nil {
		// For discovery server, use root hierarchy ID
		if s.config.Agent.NodeID == "discovery-server" {
			currentNode = &discovery.Node{
				ID:          currentNodeID,
				Name:        s.config.Agent.NodeName,
				Address:     s.config.Agent.Address,
				Port:        s.config.Agent.Port,
				Region:      s.config.Agent.Region,
				Datacenter:  s.config.Agent.Datacenter,
				HierarchyID: "34.23.1",
				Level:       3,
				ParentID:    "",
				Status:      "active",
				LastSeen:    time.Now(),
			}
		} else {
			// For worker nodes, use fallback info
			// The hierarchy info should have been synced during registration
			currentNode = &discovery.Node{
				ID:          currentNodeID,
				Name:        s.config.Agent.NodeName,
				Address:     s.config.Agent.Address,
				Port:        s.config.Agent.Port,
				Region:      s.config.Agent.Region,
				Datacenter:  s.config.Agent.Datacenter,
				HierarchyID: currentNodeID, // Fallback to node ID
				Level:       1,
				ParentID:    "",
				Status:      "active",
				LastSeen:    time.Now(),
			}
		}
	}

	hierarchyInfo := map[string]interface{}{
		"node_id":      currentNode.ID,
		"hierarchy_id": currentNode.HierarchyID,
		"level":        currentNode.Level,
		"parent_id":    currentNode.ParentID,
		"name":         currentNode.Name,
		"address":      currentNode.Address,
		"port":         currentNode.Port,
		"region":       currentNode.Region,
		"datacenter":   currentNode.Datacenter,
		"status":       currentNode.Status,
		"last_seen":    currentNode.LastSeen.Format(time.RFC3339),
	}

	s.writeJSON(w, http.StatusOK, hierarchyInfo)
}

// getCurrentNodeHierarchyID returns the current node's hierarchy ID
func (s *Server) getCurrentNodeHierarchyID() string {
	// If this is a discovery server, use a root hierarchy ID
	if s.config.Agent.NodeID == "discovery-server" {
		return "34.23.1"
	}

	// For other nodes, try to get from discovery or use node ID
	return s.config.Agent.NodeID
}

// calculateCurrentNodeLevel calculates the current node's hierarchy level
func (s *Server) calculateCurrentNodeLevel() int {
	hierarchyID := s.getCurrentNodeHierarchyID()
	level := 1
	for _, char := range hierarchyID {
		if char == '.' {
			level++
		}
	}
	return level
}
