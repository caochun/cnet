package register

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ResourceChangeCallback 资源变化回调函数
type ResourceChangeCallback func()

// Register 资源注册器
type Register struct {
	nodeID string
	logger *logrus.Logger
	mu     sync.RWMutex

	// 本地资源
	localResources *NodeResources

	// 上级父节点资源
	parentNode *NodeResources

	// 下级节点资源（树状结构）
	childNodes map[string]*NodeResources

	// 同级peer节点资源
	peerNodes map[string]*NodeResources

	// 资源分配记录
	allocations map[string]*Allocation

	// 资源变化回调
	resourceChangeCallback ResourceChangeCallback

	ctx    context.Context
	cancel context.CancelFunc
}

// Allocation 资源分配记录
type Allocation struct {
	ID          string               `json:"id"`
	WorkloadID  string               `json:"workload_id"`
	NodeID      string               `json:"node_id"`
	Resources   ResourceRequirements `json:"resources"`
	AllocatedAt time.Time            `json:"allocated_at"`
	Status      string               `json:"status"` // "allocated", "released"
}

// NewRegister 创建新的资源注册器
func NewRegister(nodeID string, totalResources Resources, logger *logrus.Logger) *Register {
	ctx, cancel := context.WithCancel(context.Background())

	return &Register{
		nodeID: nodeID,
		logger: logger,
		localResources: &NodeResources{
			NodeID:      nodeID,
			NodeType:    "local",
			Total:       totalResources,
			Available:   totalResources.Clone(),
			Used:        Resources{},
			LastUpdated: time.Now(),
			Status:      "active",
			Metadata:    make(map[string]string),
		},
		childNodes:  make(map[string]*NodeResources),
		peerNodes:   make(map[string]*NodeResources),
		allocations: make(map[string]*Allocation),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start 启动资源注册器
func (r *Register) Start() error {
	r.logger.Info("Register started")

	// 启动资源监控协程
	go r.monitorResources()

	return nil
}

// Stop 停止资源注册器
func (r *Register) Stop() error {
	r.cancel()
	r.logger.Info("Register stopped")
	return nil
}

// GetLocalResources 获取本地资源信息
func (r *Register) GetLocalResources() *NodeResources {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 返回副本
	res := *r.localResources
	return &res
}

// RegisterChildNode 注册下级节点
func (r *Register) RegisterChildNode(nodeID string, resources *NodeResources) error {
	if nodeID == "" {
		return fmt.Errorf("node ID cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	resources.NodeType = "child"
	resources.LastUpdated = time.Now()
	r.childNodes[nodeID] = resources

	r.logger.WithFields(logrus.Fields{
		"child_node_id": nodeID,
		"resources":     resources.Total.String(),
	}).Info("Child node registered")

	return nil
}

// RegisterPeerNode 注册同级peer节点
func (r *Register) RegisterPeerNode(nodeID string, resources *NodeResources) error {
	if nodeID == "" {
		return fmt.Errorf("node ID cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	resources.NodeType = "peer"
	resources.LastUpdated = time.Now()
	r.peerNodes[nodeID] = resources

	r.logger.WithFields(logrus.Fields{
		"peer_node_id": nodeID,
		"resources":    resources.Total.String(),
	}).Info("Peer node registered")

	return nil
}

// UpdateNodeResources 更新节点资源信息
func (r *Register) UpdateNodeResources(nodeID string, resources *NodeResources) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 尝试在child nodes中查找
	if _, exists := r.childNodes[nodeID]; exists {
		resources.NodeType = "child"
		resources.LastUpdated = time.Now()
		r.childNodes[nodeID] = resources
		r.logger.WithField("node_id", nodeID).Debug("Child node resources updated")
		return nil
	}

	// 尝试在peer nodes中查找
	if _, exists := r.peerNodes[nodeID]; exists {
		resources.NodeType = "peer"
		resources.LastUpdated = time.Now()
		r.peerNodes[nodeID] = resources
		r.logger.WithField("node_id", nodeID).Debug("Peer node resources updated")
		return nil
	}

	return fmt.Errorf("node not found: %s", nodeID)
}

// UnregisterNode 注销节点
func (r *Register) UnregisterNode(nodeID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 从child nodes中删除
	if _, exists := r.childNodes[nodeID]; exists {
		delete(r.childNodes, nodeID)
		r.logger.WithField("node_id", nodeID).Info("Child node unregistered")
		return nil
	}

	// 从peer nodes中删除
	if _, exists := r.peerNodes[nodeID]; exists {
		delete(r.peerNodes, nodeID)
		r.logger.WithField("node_id", nodeID).Info("Peer node unregistered")
		return nil
	}

	return fmt.Errorf("node not found: %s", nodeID)
}

// SetParentNode 设置父节点信息
func (r *Register) SetParentNode(parentResources *NodeResources) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.parentNode = parentResources
	r.logger.WithField("parent_node_id", parentResources.NodeID).Info("Parent node set")
}

// GetParentNode 获取父节点信息
func (r *Register) GetParentNode() *NodeResources {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.parentNode
}

// SetResourceChangeCallback 设置资源变化回调
func (r *Register) SetResourceChangeCallback(callback ResourceChangeCallback) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resourceChangeCallback = callback
}

// triggerResourceChange 触发资源变化回调
func (r *Register) triggerResourceChange() {
	if r.resourceChangeCallback != nil {
		// 异步触发，避免阻塞
		go r.resourceChangeCallback()
	}
}

// AllocateResources 分配资源
func (r *Register) AllocateResources(workloadID string, req ResourceRequirements) (*Allocation, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid resource requirements: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查本地资源是否充足
	if !r.localResources.Available.CanSatisfy(req) {
		return nil, fmt.Errorf("insufficient resources: available=%s, required=CPU:%.2f GPU:%d Mem:%dMB",
			r.localResources.Available.String(), req.CPU, req.GPU, req.Memory/(1024*1024))
	}

	// 分配资源
	r.localResources.Available = r.localResources.Available.Sub(req)
	r.localResources.Used = r.localResources.Used.Add(req)

	// 创建分配记录
	allocation := &Allocation{
		ID:          fmt.Sprintf("alloc-%d", time.Now().UnixNano()),
		WorkloadID:  workloadID,
		NodeID:      r.nodeID,
		Resources:   req,
		AllocatedAt: time.Now(),
		Status:      "allocated",
	}

	r.allocations[allocation.ID] = allocation

	r.logger.WithFields(logrus.Fields{
		"workload_id":   workloadID,
		"allocation_id": allocation.ID,
		"resources":     fmt.Sprintf("CPU:%.2f GPU:%d Mem:%dMB", req.CPU, req.GPU, req.Memory/(1024*1024)),
	}).Info("Resources allocated")

	// 触发资源变化回调（通知父节点）
	r.mu.Unlock()
	r.triggerResourceChange()
	r.mu.Lock()

	return allocation, nil
}

// ReleaseResources 释放资源
func (r *Register) ReleaseResources(allocationID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	allocation, exists := r.allocations[allocationID]
	if !exists {
		return fmt.Errorf("allocation not found: %s", allocationID)
	}

	if allocation.Status == "released" {
		return fmt.Errorf("allocation already released: %s", allocationID)
	}

	// 释放资源
	r.localResources.Available = r.localResources.Available.Add(allocation.Resources)
	r.localResources.Used = r.localResources.Used.Sub(allocation.Resources)
	allocation.Status = "released"

	r.logger.WithFields(logrus.Fields{
		"allocation_id": allocationID,
		"workload_id":   allocation.WorkloadID,
	}).Info("Resources released")

	// 触发资源变化回调（通知父节点）
	r.mu.Unlock()
	r.triggerResourceChange()
	r.mu.Lock()

	return nil
}

// GetAllNodes 获取所有节点资源信息（包括本地、child、peer）
func (r *Register) GetAllNodes() []*NodeResources {
	r.mu.RLock()
	defer r.mu.RUnlock()

	nodes := make([]*NodeResources, 0, 1+len(r.childNodes)+len(r.peerNodes))

	// 添加本地资源
	localCopy := *r.localResources
	nodes = append(nodes, &localCopy)

	// 添加child节点
	for _, node := range r.childNodes {
		nodeCopy := *node
		nodes = append(nodes, &nodeCopy)
	}

	// 添加peer节点
	for _, node := range r.peerNodes {
		nodeCopy := *node
		nodes = append(nodes, &nodeCopy)
	}

	return nodes
}

// GetChildNodes 获取所有下级节点
func (r *Register) GetChildNodes() []*NodeResources {
	r.mu.RLock()
	defer r.mu.RUnlock()

	nodes := make([]*NodeResources, 0, len(r.childNodes))
	for _, node := range r.childNodes {
		nodeCopy := *node
		nodes = append(nodes, &nodeCopy)
	}

	return nodes
}

// GetPeerNodes 获取所有同级节点
func (r *Register) GetPeerNodes() []*NodeResources {
	r.mu.RLock()
	defer r.mu.RUnlock()

	nodes := make([]*NodeResources, 0, len(r.peerNodes))
	for _, node := range r.peerNodes {
		nodeCopy := *node
		nodes = append(nodes, &nodeCopy)
	}

	return nodes
}

// monitorResources 监控资源状态
func (r *Register) monitorResources() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.checkStaleNodes()
		}
	}
}

// checkStaleNodes 检查过期节点
func (r *Register) checkStaleNodes() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	timeout := 90 * time.Second // 90秒没更新视为unreachable

	// 检查child节点
	for nodeID, node := range r.childNodes {
		if now.Sub(node.LastUpdated) > timeout && node.Status != "unreachable" {
			node.Status = "unreachable"
			r.logger.WithField("node_id", nodeID).Warn("Child node marked as unreachable")
		}
	}

	// 检查peer节点
	for nodeID, node := range r.peerNodes {
		if now.Sub(node.LastUpdated) > timeout && node.Status != "unreachable" {
			node.Status = "unreachable"
			r.logger.WithField("node_id", nodeID).Warn("Peer node marked as unreachable")
		}
	}
}
