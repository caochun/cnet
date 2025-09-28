package api

import (
	"encoding/json"
	"net/http"

	"cnet/internal/agent/discovery"
)

// handleListNodes handles node listing requests
func (s *Server) handleListNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := s.discovery.ListNodes()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to list nodes", err)
		return
	}

	s.writeJSON(w, http.StatusOK, nodes)
}

// handleRegister handles node registration requests
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	// Validate content type
	if r.Header.Get("Content-Type") != "application/json" {
		s.writeError(w, http.StatusBadRequest, "Content-Type must be application/json", nil)
		return
	}

	var req discovery.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Errorf("Failed to decode register request: %v", err)
		s.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if err := s.discovery.RegisterNode(&req); err != nil {
		s.logger.Errorf("Failed to register node %s: %v", req.Node.ID, err)
		s.writeError(w, http.StatusInternalServerError, "Failed to register node", err)
		return
	}

	s.logger.Infof("Successfully registered node: %s", req.Node.ID)
	s.writeJSON(w, http.StatusOK, map[string]string{"message": "Node registered"})
}

// handleDeregister handles node deregistration requests
func (s *Server) handleDeregister(w http.ResponseWriter, r *http.Request) {
	var req discovery.DeregisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if err := s.discovery.DeregisterNode(&req); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to deregister node", err)
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{"message": "Node deregistered"})
}

// handleAssignHierarchyID handles hierarchy ID assignment requests
func (s *Server) handleAssignHierarchyID(w http.ResponseWriter, r *http.Request) {
	var req discovery.HierarchyAssignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	response, err := s.discovery.AssignHierarchyID(&req)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to assign hierarchy ID", err)
		return
	}

	s.writeJSON(w, http.StatusOK, response)
}

// handleResolveHierarchyID handles hierarchy ID resolution requests
func (s *Server) handleResolveHierarchyID(w http.ResponseWriter, r *http.Request) {
	var req discovery.ResolveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	response, err := s.discovery.ResolveHierarchyID(&req)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "Hierarchy ID not found", err)
		return
	}

	s.writeJSON(w, http.StatusOK, response)
}

// handleListNodesByHierarchy handles hierarchical node listing requests
func (s *Server) handleListNodesByHierarchy(w http.ResponseWriter, r *http.Request) {
	hierarchy, err := s.discovery.ListNodesByHierarchy()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to list nodes by hierarchy", err)
		return
	}

	s.writeJSON(w, http.StatusOK, hierarchy)
}
