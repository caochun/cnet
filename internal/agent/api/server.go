package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"cnet/internal/agent/discovery"
	"cnet/internal/agent/resources"
	"cnet/internal/agent/tasks"
	"cnet/internal/config"
	"cnet/internal/logger"

	"github.com/gorilla/mux"
)

// Server represents the API server
type Server struct {
	config    *config.Config
	logger    *logger.Logger
	discovery *discovery.Service
	resources *resources.Service
	tasks     *tasks.Service
	router    *mux.Router
}

// New creates a new API server
func New(cfg *config.Config, log *logger.Logger, disc *discovery.Service, res *resources.Service, tsk *tasks.Service) *Server {
	server := &Server{
		config:    cfg,
		logger:    log,
		discovery: disc,
		resources: res,
		tasks:     tsk,
		router:    mux.NewRouter(),
	}

	server.setupRoutes()
	return server
}

// Handler returns the HTTP handler
func (s *Server) Handler() http.Handler {
	return s.router
}

// setupRoutes sets up the API routes
func (s *Server) setupRoutes() {
	// Web UI routes
	s.router.HandleFunc("/", s.handleWebUI).Methods("GET")
	s.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static/"))))

	// API routes
	api := s.router.PathPrefix("/api").Subrouter()

	// Health check
	api.HandleFunc("/health", s.handleHealth).Methods("GET")

	// Node information
	api.HandleFunc("/node", s.handleNodeInfo).Methods("GET")
	api.HandleFunc("/node/hierarchy", s.handleNodeHierarchy).Methods("GET")

	// Resource information
	api.HandleFunc("/resources", s.handleResources).Methods("GET")
	api.HandleFunc("/resources/usage", s.handleResourceUsage).Methods("GET")

	// Task management
	api.HandleFunc("/tasks", s.handleListTasks).Methods("GET")
	api.HandleFunc("/tasks", s.handleCreateTask).Methods("POST")
	api.HandleFunc("/tasks/{id}", s.handleGetTask).Methods("GET")
	api.HandleFunc("/tasks/{id}", s.handleStopTask).Methods("DELETE")
	api.HandleFunc("/tasks/{id}/logs", s.handleTaskLogs).Methods("GET")

	// Discovery
	api.HandleFunc("/discovery/nodes", s.handleListNodes).Methods("GET")
	api.HandleFunc("/discovery/register", s.handleRegister).Methods("POST")
	api.HandleFunc("/discovery/deregister", s.handleDeregister).Methods("POST")

	// Hierarchy management
	api.HandleFunc("/discovery/hierarchy/assign", s.handleAssignHierarchyID).Methods("POST")
	api.HandleFunc("/discovery/hierarchy/resolve", s.handleResolveHierarchyID).Methods("POST")
	api.HandleFunc("/discovery/hierarchy/nodes", s.handleListNodesByHierarchy).Methods("GET")

	// Legacy API routes for backward compatibility
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
	s.router.HandleFunc("/node", s.handleNodeInfo).Methods("GET")
	s.router.HandleFunc("/resources", s.handleResources).Methods("GET")
	s.router.HandleFunc("/resources/usage", s.handleResourceUsage).Methods("GET")
	s.router.HandleFunc("/tasks", s.handleListTasks).Methods("GET")
	s.router.HandleFunc("/tasks", s.handleCreateTask).Methods("POST")
	s.router.HandleFunc("/tasks/{id}", s.handleGetTask).Methods("GET")
	s.router.HandleFunc("/tasks/{id}", s.handleStopTask).Methods("DELETE")
	s.router.HandleFunc("/tasks/{id}/logs", s.handleTaskLogs).Methods("GET")
	s.router.HandleFunc("/discovery/nodes", s.handleListNodes).Methods("GET")
	s.router.HandleFunc("/discovery/register", s.handleRegister).Methods("POST")
	s.router.HandleFunc("/discovery/deregister", s.handleDeregister).Methods("POST")
}

// handleWebUI serves the web UI
func (s *Server) handleWebUI(w http.ResponseWriter, r *http.Request) {
	// Serve the main HTML file
	http.ServeFile(w, r, "web/templates/index.html")
}

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

// handleResources handles resource information requests
func (s *Server) handleResources(w http.ResponseWriter, r *http.Request) {
	resources, err := s.resources.GetResources()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to get resources", err)
		return
	}

	s.writeJSON(w, http.StatusOK, resources)
}

// handleResourceUsage handles resource usage requests
func (s *Server) handleResourceUsage(w http.ResponseWriter, r *http.Request) {
	usage, err := s.resources.GetUsage()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to get resource usage", err)
		return
	}

	s.writeJSON(w, http.StatusOK, usage)
}

// handleListTasks handles task listing requests
func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.tasks.ListTasks()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to list tasks", err)
		return
	}

	s.writeJSON(w, http.StatusOK, tasks)
}

// handleCreateTask handles task creation requests
func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req tasks.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	task, err := s.tasks.CreateTask(&req)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to create task", err)
		return
	}

	s.writeJSON(w, http.StatusCreated, task)
}

// handleGetTask handles task retrieval requests
func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	task, err := s.tasks.GetTask(taskID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "Task not found", err)
		return
	}

	s.writeJSON(w, http.StatusOK, task)
}

// handleStopTask handles task stopping requests
func (s *Server) handleStopTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	if err := s.tasks.StopTask(taskID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to stop task", err)
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{"message": "Task stopped"})
}

// handleTaskLogs handles task log requests
func (s *Server) handleTaskLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	// Parse query parameters
	lines := 100
	if linesStr := r.URL.Query().Get("lines"); linesStr != "" {
		if l, err := strconv.Atoi(linesStr); err == nil && l > 0 {
			lines = l
		}
	}

	logs, err := s.tasks.GetTaskLogs(taskID, lines)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "Task logs not found", err)
		return
	}

	s.writeJSON(w, http.StatusOK, logs)
}

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

// writeJSON writes a JSON response
func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.Errorf("Failed to encode JSON response: %v", err)
	}
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

// writeError writes an error response
func (s *Server) writeError(w http.ResponseWriter, status int, message string, err error) {
	response := map[string]interface{}{
		"error":     message,
		"status":    status,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	if err != nil {
		response["details"] = err.Error()
	}

	s.writeJSON(w, status, response)
}
