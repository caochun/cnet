package api

import (
	"net/http"

	"cnet/internal/agent/discovery"
	"cnet/internal/agent/ml"
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
	ml        *ml.Service
	router    *mux.Router

	// Task handlers for different task types
	taskHandlers map[string]TaskHandler
}

// New creates a new API server
func New(cfg *config.Config, log *logger.Logger, disc *discovery.Service, res *resources.Service, tsk *tasks.Service, mlService *ml.Service) *Server {
	server := &Server{
		config:       cfg,
		logger:       log,
		discovery:    disc,
		resources:    res,
		tasks:        tsk,
		ml:           mlService,
		router:       mux.NewRouter(),
		taskHandlers: make(map[string]TaskHandler),
	}

	// Initialize task handlers
	server.initTaskHandlers()
	server.setupRoutes()
	return server
}

// Handler returns the HTTP handler
func (s *Server) Handler() http.Handler {
	return s.router
}

// initTaskHandlers initializes task handlers for different task types
func (s *Server) initTaskHandlers() {
	s.taskHandlers["process"] = NewProcessTaskHandler(s)
	s.taskHandlers["container"] = NewContainerTaskHandler(s)
	s.taskHandlers["vm"] = NewVMTaskHandler(s)
	s.taskHandlers["ml"] = NewMLTaskHandler(s)
	s.taskHandlers["yolo"] = NewYOLOTaskHandler(s)
}

// setupRoutes sets up the API routes
func (s *Server) setupRoutes() {
	// Web UI routes
	s.router.HandleFunc("/", s.handleWebUI).Methods("GET")
	s.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static/"))))

	// API routes
	api := s.router.PathPrefix("/api").Subrouter()

	// Health check and node info
	api.HandleFunc("/health", s.handleHealth).Methods("GET")
	api.HandleFunc("/node", s.handleNodeInfo).Methods("GET")
	api.HandleFunc("/node/hierarchy", s.handleNodeHierarchy).Methods("GET")

	// Resource information
	api.HandleFunc("/resources", s.handleResources).Methods("GET")
	api.HandleFunc("/resources/usage", s.handleResourceUsage).Methods("GET")

	// Task management (generic)
	api.HandleFunc("/tasks", s.handleGenericTasks).Methods("GET")
	api.HandleFunc("/tasks", s.handleGenericTasks).Methods("POST")
	api.HandleFunc("/tasks/{id}", s.handleGenericTasks).Methods("GET")
	api.HandleFunc("/tasks/{id}", s.handleGenericTasks).Methods("DELETE")
	api.HandleFunc("/tasks/{id}/logs", s.handleGenericTasks).Methods("GET")
	api.HandleFunc("/tasks/{id}/info", s.handleGenericTasks).Methods("GET")
	api.HandleFunc("/tasks/{id}/health", s.handleGenericTasks).Methods("GET")

	// Process tasks
	api.HandleFunc("/process/tasks", s.handleProcessTasks).Methods("GET")
	api.HandleFunc("/process/tasks", s.handleProcessTasks).Methods("POST")
	api.HandleFunc("/process/tasks/{id}", s.handleProcessTasks).Methods("GET")
	api.HandleFunc("/process/tasks/{id}", s.handleProcessTasks).Methods("DELETE")
	api.HandleFunc("/process/tasks/{id}/logs", s.handleProcessTasks).Methods("GET")
	api.HandleFunc("/process/tasks/{id}/info", s.handleProcessTasks).Methods("GET")
	api.HandleFunc("/process/tasks/{id}/health", s.handleProcessTasks).Methods("GET")

	// Container tasks
	api.HandleFunc("/container/tasks", s.handleContainerTasks).Methods("GET")
	api.HandleFunc("/container/tasks", s.handleContainerTasks).Methods("POST")
	api.HandleFunc("/container/tasks/{id}", s.handleContainerTasks).Methods("GET")
	api.HandleFunc("/container/tasks/{id}", s.handleContainerTasks).Methods("DELETE")
	api.HandleFunc("/container/tasks/{id}/logs", s.handleContainerTasks).Methods("GET")
	api.HandleFunc("/container/tasks/{id}/info", s.handleContainerTasks).Methods("GET")
	api.HandleFunc("/container/tasks/{id}/health", s.handleContainerTasks).Methods("GET")

	// VM tasks
	api.HandleFunc("/vm/tasks", s.handleVMTasks).Methods("GET")
	api.HandleFunc("/vm/tasks", s.handleVMTasks).Methods("POST")
	api.HandleFunc("/vm/tasks/{id}", s.handleVMTasks).Methods("GET")
	api.HandleFunc("/vm/tasks/{id}", s.handleVMTasks).Methods("DELETE")
	api.HandleFunc("/vm/tasks/{id}/logs", s.handleVMTasks).Methods("GET")
	api.HandleFunc("/vm/tasks/{id}/info", s.handleVMTasks).Methods("GET")
	api.HandleFunc("/vm/tasks/{id}/health", s.handleVMTasks).Methods("GET")

	// ML tasks
	api.HandleFunc("/ml/tasks", s.handleMLTasks).Methods("GET")
	api.HandleFunc("/ml/tasks", s.handleMLTasks).Methods("POST")
	api.HandleFunc("/ml/tasks/{id}", s.handleMLTasks).Methods("GET")
	api.HandleFunc("/ml/tasks/{id}", s.handleMLTasks).Methods("DELETE")
	api.HandleFunc("/ml/tasks/{id}/predict", s.handleMLTasks).Methods("POST")
	api.HandleFunc("/ml/tasks/{id}/logs", s.handleMLTasks).Methods("GET")
	api.HandleFunc("/ml/tasks/{id}/info", s.handleMLTasks).Methods("GET")
	api.HandleFunc("/ml/tasks/{id}/health", s.handleMLTasks).Methods("GET")

	// YOLO tasks
	api.HandleFunc("/yolo/tasks", s.handleYOLOTasks).Methods("GET")
	api.HandleFunc("/yolo/tasks", s.handleYOLOTasks).Methods("POST")
	api.HandleFunc("/yolo/tasks/{id}", s.handleYOLOTasks).Methods("GET")
	api.HandleFunc("/yolo/tasks/{id}", s.handleYOLOTasks).Methods("DELETE")
	api.HandleFunc("/yolo/tasks/{id}/predict", s.handleYOLOTasks).Methods("POST")
	api.HandleFunc("/yolo/tasks/{id}/logs", s.handleYOLOTasks).Methods("GET")
	api.HandleFunc("/yolo/tasks/{id}/info", s.handleYOLOTasks).Methods("GET")
	api.HandleFunc("/yolo/tasks/{id}/health", s.handleYOLOTasks).Methods("GET")
	api.HandleFunc("/yolo/tasks/{id}/model", s.handleYOLOTasks).Methods("GET")

	// Legacy ML API routes for backward compatibility
	api.HandleFunc("/ml/models", s.handleMLTasks).Methods("GET")
	api.HandleFunc("/ml/models", s.handleMLTasks).Methods("POST")
	api.HandleFunc("/ml/models/{id}", s.handleMLTasks).Methods("GET")
	api.HandleFunc("/ml/models/{id}", s.handleMLTasks).Methods("DELETE")
	api.HandleFunc("/ml/models/{id}/predict", s.handleMLTasks).Methods("POST")
	api.HandleFunc("/ml/models/{id}/logs", s.handleMLTasks).Methods("GET")
	api.HandleFunc("/ml/models/{id}/info", s.handleMLTasks).Methods("GET")
	api.HandleFunc("/ml/models/{id}/health", s.handleMLTasks).Methods("GET")

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
	s.router.HandleFunc("/tasks", s.handleGenericTasks).Methods("GET")
	s.router.HandleFunc("/tasks", s.handleGenericTasks).Methods("POST")
	s.router.HandleFunc("/tasks/{id}", s.handleGenericTasks).Methods("GET")
	s.router.HandleFunc("/tasks/{id}", s.handleGenericTasks).Methods("DELETE")
	s.router.HandleFunc("/tasks/{id}/logs", s.handleGenericTasks).Methods("GET")
	s.router.HandleFunc("/discovery/nodes", s.handleListNodes).Methods("GET")
	s.router.HandleFunc("/discovery/register", s.handleRegister).Methods("POST")
	s.router.HandleFunc("/discovery/deregister", s.handleDeregister).Methods("POST")
}

// handleWebUI serves the web UI
func (s *Server) handleWebUI(w http.ResponseWriter, r *http.Request) {
	// Serve the main HTML file
	http.ServeFile(w, r, "web/templates/index.html")
}

// Generic task handlers that delegate to appropriate task handlers

func (s *Server) handleGenericTasks(w http.ResponseWriter, r *http.Request) {
	// Use process handler as default for generic tasks
	handler := s.taskHandlers["process"]
	switch r.Method {
	case "GET":
		if r.URL.Path == "/api/tasks" {
			handler.ListTasks(w, r)
		} else {
			handler.GetTask(w, r)
		}
	case "POST":
		handler.CreateTask(w, r)
	case "DELETE":
		handler.StopTask(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
	}
}

func (s *Server) handleProcessTasks(w http.ResponseWriter, r *http.Request) {
	handler := s.taskHandlers["process"]
	switch r.Method {
	case "GET":
		if r.URL.Path == "/api/process/tasks" {
			handler.ListTasks(w, r)
		} else if r.URL.Path[len(r.URL.Path)-5:] == "/info" {
			handler.GetTaskInfo(w, r)
		} else if r.URL.Path[len(r.URL.Path)-6:] == "/health" {
			handler.GetTaskHealth(w, r)
		} else if r.URL.Path[len(r.URL.Path)-5:] == "/logs" {
			handler.GetTaskLogs(w, r)
		} else {
			handler.GetTask(w, r)
		}
	case "POST":
		handler.CreateTask(w, r)
	case "DELETE":
		handler.StopTask(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
	}
}

func (s *Server) handleContainerTasks(w http.ResponseWriter, r *http.Request) {
	handler := s.taskHandlers["container"]
	switch r.Method {
	case "GET":
		if r.URL.Path == "/api/container/tasks" {
			handler.ListTasks(w, r)
		} else if r.URL.Path[len(r.URL.Path)-5:] == "/info" {
			handler.GetTaskInfo(w, r)
		} else if r.URL.Path[len(r.URL.Path)-6:] == "/health" {
			handler.GetTaskHealth(w, r)
		} else if r.URL.Path[len(r.URL.Path)-5:] == "/logs" {
			handler.GetTaskLogs(w, r)
		} else {
			handler.GetTask(w, r)
		}
	case "POST":
		handler.CreateTask(w, r)
	case "DELETE":
		handler.StopTask(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
	}
}

func (s *Server) handleVMTasks(w http.ResponseWriter, r *http.Request) {
	handler := s.taskHandlers["vm"]
	switch r.Method {
	case "GET":
		if r.URL.Path == "/api/vm/tasks" {
			handler.ListTasks(w, r)
		} else if r.URL.Path[len(r.URL.Path)-5:] == "/info" {
			handler.GetTaskInfo(w, r)
		} else if r.URL.Path[len(r.URL.Path)-6:] == "/health" {
			handler.GetTaskHealth(w, r)
		} else if r.URL.Path[len(r.URL.Path)-5:] == "/logs" {
			handler.GetTaskLogs(w, r)
		} else {
			handler.GetTask(w, r)
		}
	case "POST":
		handler.CreateTask(w, r)
	case "DELETE":
		handler.StopTask(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
	}
}

func (s *Server) handleMLTasks(w http.ResponseWriter, r *http.Request) {
	handler := s.taskHandlers["ml"]
	switch r.Method {
	case "GET":
		if r.URL.Path == "/api/ml/tasks" || r.URL.Path == "/api/ml/models" {
			handler.ListTasks(w, r)
		} else if r.URL.Path[len(r.URL.Path)-5:] == "/info" {
			handler.GetTaskInfo(w, r)
		} else if r.URL.Path[len(r.URL.Path)-6:] == "/health" {
			handler.GetTaskHealth(w, r)
		} else if r.URL.Path[len(r.URL.Path)-5:] == "/logs" {
			handler.GetTaskLogs(w, r)
		} else {
			handler.GetTask(w, r)
		}
	case "POST":
		if r.URL.Path[len(r.URL.Path)-8:] == "/predict" {
			// Special case for ML prediction
			if mlHandler, ok := handler.(*MLTaskHandler); ok {
				mlHandler.Predict(w, r)
			} else {
				s.writeError(w, http.StatusInternalServerError, "ML handler not available", nil)
			}
		} else {
			handler.CreateTask(w, r)
		}
	case "DELETE":
		handler.StopTask(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
	}
}

func (s *Server) handleYOLOTasks(w http.ResponseWriter, r *http.Request) {
	handler := s.taskHandlers["yolo"]
	switch r.Method {
	case "GET":
		if r.URL.Path == "/api/yolo/tasks" {
			handler.ListTasks(w, r)
		} else if r.URL.Path[len(r.URL.Path)-5:] == "/info" {
			handler.GetTaskInfo(w, r)
		} else if r.URL.Path[len(r.URL.Path)-6:] == "/health" {
			handler.GetTaskHealth(w, r)
		} else if r.URL.Path[len(r.URL.Path)-5:] == "/logs" {
			handler.GetTaskLogs(w, r)
		} else if r.URL.Path[len(r.URL.Path)-6:] == "/model" {
			// Special case for YOLO model info
			if yoloHandler, ok := handler.(*YOLOTaskHandler); ok {
				yoloHandler.GetModelInfo(w, r)
			} else {
				s.writeError(w, http.StatusInternalServerError, "YOLO handler not available", nil)
			}
		} else {
			handler.GetTask(w, r)
		}
	case "POST":
		if r.URL.Path[len(r.URL.Path)-8:] == "/predict" {
			// Special case for YOLO prediction
			if yoloHandler, ok := handler.(*YOLOTaskHandler); ok {
				yoloHandler.Predict(w, r)
			} else {
				s.writeError(w, http.StatusInternalServerError, "YOLO handler not available", nil)
			}
		} else {
			handler.CreateTask(w, r)
		}
	case "DELETE":
		handler.StopTask(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
	}
}
