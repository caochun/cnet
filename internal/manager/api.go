package manager

import (
	"encoding/json"
	"net/http"
	"strconv"

	"cnet/internal/register"
	"cnet/internal/workload"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// API Manager的HTTP API
type API struct {
	manager  *Manager
	register *register.Register
	logger   *logrus.Logger
	router   *mux.Router
}

// NewAPI 创建API
func NewAPI(mgr *Manager, reg *register.Register, logger *logrus.Logger) *API {
	api := &API{
		manager:  mgr,
		register: reg,
		logger:   logger,
		router:   mux.NewRouter(),
	}

	api.setupRoutes()
	return api
}

// GetRouter 获取路由器
func (a *API) GetRouter() *mux.Router {
	return a.router
}

// setupRoutes 设置路由
func (a *API) setupRoutes() {
	// API路由
	apiRouter := a.router.PathPrefix("/api").Subrouter()

	// Workload管理
	apiRouter.HandleFunc("/workloads", a.handleListWorkloads).Methods("GET")
	apiRouter.HandleFunc("/workloads", a.handleSubmitWorkload).Methods("POST")
	apiRouter.HandleFunc("/workloads/{id}", a.handleGetWorkload).Methods("GET")
	apiRouter.HandleFunc("/workloads/{id}", a.handleDeleteWorkload).Methods("DELETE")
	apiRouter.HandleFunc("/workloads/{id}/stop", a.handleStopWorkload).Methods("POST")
	apiRouter.HandleFunc("/workloads/{id}/logs", a.handleGetLogs).Methods("GET")

	// 资源信息
	apiRouter.HandleFunc("/resources", a.handleGetResources).Methods("GET")
	apiRouter.HandleFunc("/resources/stats", a.handleGetResourceStats).Methods("GET")

	// 节点管理
	apiRouter.HandleFunc("/nodes", a.handleListNodes).Methods("GET")
	apiRouter.HandleFunc("/nodes/local", a.handleGetLocalNode).Methods("GET")

	// 子节点注册（父节点功能）
	apiRouter.HandleFunc("/register", a.handleRegisterChild).Methods("POST")
	apiRouter.HandleFunc("/unregister", a.handleUnregisterChild).Methods("POST")
	apiRouter.HandleFunc("/heartbeat", a.handleHeartbeat).Methods("POST")

	// Peer注册
	apiRouter.HandleFunc("/peer/register", a.handleRegisterPeer).Methods("POST")
	apiRouter.HandleFunc("/peer/unregister", a.handleUnregisterPeer).Methods("POST")

	// 健康检查
	apiRouter.HandleFunc("/health", a.handleHealth).Methods("GET")
}

// handleSubmitWorkload 提交workload
func (a *API) handleSubmitWorkload(w http.ResponseWriter, r *http.Request) {
	var req workload.CreateWorkloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	wl, err := a.manager.SubmitWorkload(r.Context(), &req)
	if err != nil {
		a.writeError(w, http.StatusInternalServerError, "Failed to submit workload", err)
		return
	}

	a.writeJSON(w, http.StatusCreated, wl)
}

// handleListWorkloads 列出所有workload
func (a *API) handleListWorkloads(w http.ResponseWriter, r *http.Request) {
	workloads := a.manager.ListWorkloads()
	a.writeJSON(w, http.StatusOK, map[string]interface{}{
		"workloads": workloads,
		"count":     len(workloads),
	})
}

// handleGetWorkload 获取单个workload
func (a *API) handleGetWorkload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workloadID := vars["id"]

	wl, err := a.manager.GetWorkload(workloadID)
	if err != nil {
		a.writeError(w, http.StatusNotFound, "Workload not found", err)
		return
	}

	a.writeJSON(w, http.StatusOK, wl)
}

// handleDeleteWorkload 删除workload
func (a *API) handleDeleteWorkload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workloadID := vars["id"]

	if err := a.manager.DeleteWorkload(r.Context(), workloadID); err != nil {
		a.writeError(w, http.StatusInternalServerError, "Failed to delete workload", err)
		return
	}

	a.writeJSON(w, http.StatusOK, map[string]string{
		"message": "Workload deleted successfully",
	})
}

// handleStopWorkload 停止workload
func (a *API) handleStopWorkload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workloadID := vars["id"]

	if err := a.manager.StopWorkload(r.Context(), workloadID); err != nil {
		a.writeError(w, http.StatusInternalServerError, "Failed to stop workload", err)
		return
	}

	a.writeJSON(w, http.StatusOK, map[string]string{
		"message": "Workload stopped successfully",
	})
}

// handleGetLogs 获取workload日志
func (a *API) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workloadID := vars["id"]

	// 获取行数参数
	lines := 100
	if linesStr := r.URL.Query().Get("lines"); linesStr != "" {
		if l, err := strconv.Atoi(linesStr); err == nil {
			lines = l
		}
	}

	logs, err := a.manager.GetWorkloadLogs(r.Context(), workloadID, lines)
	if err != nil {
		a.writeError(w, http.StatusInternalServerError, "Failed to get logs", err)
		return
	}

	a.writeJSON(w, http.StatusOK, map[string]interface{}{
		"logs":  logs,
		"count": len(logs),
	})
}

// handleGetResources 获取本地资源信息
func (a *API) handleGetResources(w http.ResponseWriter, r *http.Request) {
	resources := a.register.GetLocalResources()
	a.writeJSON(w, http.StatusOK, resources)
}

// handleGetResourceStats 获取资源统计
func (a *API) handleGetResourceStats(w http.ResponseWriter, r *http.Request) {
	stats := a.manager.GetResourceStats()
	a.writeJSON(w, http.StatusOK, stats)
}

// handleListNodes 列出所有节点
func (a *API) handleListNodes(w http.ResponseWriter, r *http.Request) {
	nodes := a.register.GetAllNodes()
	a.writeJSON(w, http.StatusOK, map[string]interface{}{
		"nodes": nodes,
		"count": len(nodes),
	})
}

// handleGetLocalNode 获取本地节点信息
func (a *API) handleGetLocalNode(w http.ResponseWriter, r *http.Request) {
	node := a.register.GetLocalResources()
	a.writeJSON(w, http.StatusOK, node)
}

// handleRegisterChild 注册子节点
func (a *API) handleRegisterChild(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NodeID    string                  `json:"node_id"`
		Resources *register.NodeResources `json:"resources"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if err := a.register.RegisterChildNode(req.NodeID, req.Resources); err != nil {
		a.writeError(w, http.StatusInternalServerError, "Failed to register child node", err)
		return
	}

	a.writeJSON(w, http.StatusOK, map[string]string{
		"message": "Child node registered successfully",
	})
}

// handleUnregisterChild 注销子节点
func (a *API) handleUnregisterChild(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NodeID string `json:"node_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if err := a.register.UnregisterNode(req.NodeID); err != nil {
		a.writeError(w, http.StatusInternalServerError, "Failed to unregister node", err)
		return
	}

	a.writeJSON(w, http.StatusOK, map[string]string{
		"message": "Node unregistered successfully",
	})
}

// handleHeartbeat 处理心跳
func (a *API) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NodeID    string                  `json:"node_id"`
		Resources *register.NodeResources `json:"resources"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if err := a.register.UpdateNodeResources(req.NodeID, req.Resources); err != nil {
		a.writeError(w, http.StatusInternalServerError, "Failed to update node resources", err)
		return
	}

	a.writeJSON(w, http.StatusOK, map[string]string{
		"message": "Heartbeat received",
	})
}

// handleRegisterPeer 注册peer节点
func (a *API) handleRegisterPeer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NodeID    string                  `json:"node_id"`
		Resources *register.NodeResources `json:"resources"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if err := a.register.RegisterPeerNode(req.NodeID, req.Resources); err != nil {
		a.writeError(w, http.StatusInternalServerError, "Failed to register peer node", err)
		return
	}

	a.writeJSON(w, http.StatusOK, map[string]string{
		"message": "Peer node registered successfully",
	})
}

// handleUnregisterPeer 注销peer节点
func (a *API) handleUnregisterPeer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NodeID string `json:"node_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if err := a.register.UnregisterNode(req.NodeID); err != nil {
		a.writeError(w, http.StatusInternalServerError, "Failed to unregister peer", err)
		return
	}

	a.writeJSON(w, http.StatusOK, map[string]string{
		"message": "Peer unregistered successfully",
	})
}

// handleHealth 健康检查
func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	a.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "healthy",
		"message": "CNET Agent is running",
	})
}

// 辅助方法

func (a *API) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (a *API) writeError(w http.ResponseWriter, status int, message string, err error) {
	a.logger.WithError(err).Error(message)
	a.writeJSON(w, status, map[string]string{
		"error":   message,
		"details": err.Error(),
	})
}
