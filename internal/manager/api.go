package manager

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"cnet/internal/register"
	"cnet/internal/workload"

	"github.com/google/uuid"
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
	// 静态文件服务
	staticDir := "./web/static/"
	a.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	// 主页路由
	a.router.HandleFunc("/", a.handleHomePage)

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

// handleHomePage 处理主页
func (a *API) handleHomePage(w http.ResponseWriter, r *http.Request) {
	// 读取HTML模板
	htmlPath := filepath.Join("web", "templates", "index.html")
	http.ServeFile(w, r, htmlPath)
}

// handleSubmitWorkload 提交workload
func (a *API) handleSubmitWorkload(w http.ResponseWriter, r *http.Request) {
	// 检查Content-Type
	contentType := r.Header.Get("Content-Type")

	if contentType == "application/json" {
		// JSON请求
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
	} else if contentType == "multipart/form-data" || (len(contentType) > 0 && contentType[:19] == "multipart/form-data") {
		// 文件上传请求（数据workload）
		a.handleDataWorkloadUpload(w, r)
	} else {
		a.writeError(w, http.StatusBadRequest, "Unsupported content type", nil)
		return
	}
}

// handleDataWorkloadUpload 处理数据workload文件上传（multipart/form-data）
func (a *API) handleDataWorkloadUpload(w http.ResponseWriter, r *http.Request) {
	// 解析multipart表单
	if err := r.ParseMultipartForm(64 << 20); err != nil { // 64MB
		a.writeError(w, http.StatusBadRequest, "Failed to parse multipart form", err)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		name = fmt.Sprintf("data-%s", time.Now().Format("20060102150405"))
	}

	// 解析requirements
	var reqRes register.ResourceRequirements
	if v := r.FormValue("requirements"); v != "" {
		_ = json.Unmarshal([]byte(v), &reqRes)
	}

	// 解析config（除文件内容外）
	var cfg map[string]interface{}
	if v := r.FormValue("config"); v != "" {
		_ = json.Unmarshal([]byte(v), &cfg)
	} else {
		cfg = map[string]interface{}{}
	}

	// 判断上传方式
	uploadMethod := "file"
	if m, ok := cfg["upload_method"].(string); ok && m != "" {
		uploadMethod = m
	}

	dataKey := uuid.New().String()

	// 如果表单里实际包含多文件字段名 "files"，强制按目录上传处理
	if r.MultipartForm != nil {
		if fs, ok := r.MultipartForm.File["files"]; ok && len(fs) > 0 {
			uploadMethod = "directory"
		}
	}

	if uploadMethod == "directory" {
		// 目录上传：字段名 should be "files"
		files := r.MultipartForm.File["files"]
		if len(files) == 0 {
			a.writeError(w, http.StatusBadRequest, "no files provided for directory upload", nil)
			return
		}

		finalBase := filepath.Join("/tmp/cnet_data", dataKey)
		if err := os.MkdirAll(finalBase, 0755); err != nil {
			a.writeError(w, http.StatusInternalServerError, "Failed to create final dir", err)
			return
		}

		var totalSize int64
		for _, fh := range files {
			rel := fh.Filename // 前端应传相对路径
			dstPath := filepath.Join(finalBase, rel)
			if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
				a.writeError(w, http.StatusInternalServerError, "Failed to create subdir", err)
				return
			}
			src, err := fh.Open()
			if err != nil {
				a.writeError(w, http.StatusInternalServerError, "Failed to open uploaded file", err)
				return
			}
			out, err := os.Create(dstPath)
			if err != nil {
				src.Close()
				a.writeError(w, http.StatusInternalServerError, "Failed to create file", err)
				return
			}
			n, err := io.Copy(out, src)
			src.Close()
			out.Close()
			if err != nil {
				a.writeError(w, http.StatusInternalServerError, "Failed to save uploaded file", err)
				return
			}
			totalSize += n
		}

		// 构造目录型workload
		cfg["upload_method"] = "directory"
		cfg["directory_path"] = finalBase
		cfg["file_count"] = len(files)
		cfg["total_size"] = totalSize
		cfg["data_key"] = dataKey
		if _, exists := cfg["data_type"]; !exists {
			cfg["data_type"] = "dataset"
		}

		req := &workload.CreateWorkloadRequest{
			Name:         name,
			Type:         workload.TypeData,
			Requirements: reqRes,
			Config:       cfg,
		}
		wl, err := a.manager.SubmitWorkload(r.Context(), req)
		if err != nil {
			a.writeError(w, http.StatusInternalServerError, "Failed to submit data workload", err)
			return
		}
		a.writeJSON(w, http.StatusCreated, map[string]interface{}{
			"workload": wl,
			"data_key": dataKey,
			"size":     totalSize,
			"status":   "uploaded",
			"path":     finalBase,
		})
		return
	}

	// 单文件上传路径（保持原逻辑）
	file, header, err := r.FormFile("file")
	if err != nil {
		a.writeError(w, http.StatusBadRequest, "file field is required", err)
		return
	}
	defer file.Close()

	hasher := md5.New()
	tee := io.TeeReader(file, hasher)
	tmpDir := os.TempDir()
	tmpPath := filepath.Join(tmpDir, "cnet_uploads", dataKey)
	if err := os.MkdirAll(tmpPath, 0755); err != nil {
		a.writeError(w, http.StatusInternalServerError, "Failed to create temp dir", err)
		return
	}
	dstPath := filepath.Join(tmpPath, header.Filename)
	out, err := os.Create(dstPath)
	if err != nil {
		a.writeError(w, http.StatusInternalServerError, "Failed to create temp file", err)
		return
	}
	if _, err := io.Copy(out, tee); err != nil {
		out.Close()
		a.writeError(w, http.StatusInternalServerError, "Failed to store upload", err)
		return
	}
	out.Close()
	fileHash := hex.EncodeToString(hasher.Sum(nil))

	cfg["upload_method"] = "path"
	cfg["source_path"] = dstPath
	cfg["file_name"] = header.Filename
	cfg["data_key"] = dataKey
	if _, exists := cfg["data_type"]; !exists {
		cfg["data_type"] = "file"
	}

	req := &workload.CreateWorkloadRequest{
		Name:         name,
		Type:         workload.TypeData,
		Requirements: reqRes,
		Config:       cfg,
	}
	wl, err := a.manager.SubmitWorkload(r.Context(), req)
	if err != nil {
		a.writeError(w, http.StatusInternalServerError, "Failed to submit data workload", err)
		return
	}
	// 最终持久化路径（与执行器保持一致）
	finalPath := filepath.Join("/tmp/cnet_data", dataKey, header.Filename)

	a.writeJSON(w, http.StatusCreated, map[string]interface{}{
		"workload": wl,
		"data_key": dataKey,
		"size":     header.Size,
		"hash":     fileHash,
		"status":   "uploaded",
		"path":     finalPath,
	})
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
	a.writeJSON(w, http.StatusOK, map[string]interface{}{
		"resources": resources,
	})
}

// handleGetResourceStats 获取资源统计
func (a *API) handleGetResourceStats(w http.ResponseWriter, r *http.Request) {
	stats := a.manager.GetResourceStats()
	a.writeJSON(w, http.StatusOK, stats)
}

// handleListNodes 列出所有节点
func (a *API) handleListNodes(w http.ResponseWriter, r *http.Request) {
	nodes := a.register.GetAllNodes()
	localNode := a.register.GetLocalResources()

	// 获取父节点
	parent := a.register.GetParentNode()

	// 所有子节点和peer节点（排除本地节点）
	var peers []*register.NodeResources
	for _, node := range nodes {
		if node != nil && node.NodeID != localNode.NodeID {
			peers = append(peers, node)
		}
	}

	a.writeJSON(w, http.StatusOK, map[string]interface{}{
		"parent": parent,
		"peers":  peers,
		"total":  len(peers),
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

	// 返回父节点信息给子节点
	localNode := a.register.GetLocalResources()
	a.writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Child node registered successfully",
		"parent_node": map[string]interface{}{
			"node_id": localNode.NodeID,
			"address": localNode.Address,
		},
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
