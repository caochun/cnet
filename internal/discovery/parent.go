package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"cnet/internal/register"

	"github.com/sirupsen/logrus"
)

// ParentConnector 父节点连接器
type ParentConnector struct {
	logger     *logrus.Logger
	register   *register.Register
	parentAddr string
	nodeID     string
	nodeAddr   string // 本节点的地址
	ctx        context.Context
	cancel     context.CancelFunc
	httpClient *http.Client
}

// NewParentConnector 创建父节点连接器
func NewParentConnector(logger *logrus.Logger, reg *register.Register, parentAddr, nodeID, nodeAddr string) *ParentConnector {
	ctx, cancel := context.WithCancel(context.Background())

	return &ParentConnector{
		logger:     logger,
		register:   reg,
		parentAddr: parentAddr,
		nodeID:     nodeID,
		nodeAddr:   nodeAddr,
		ctx:        ctx,
		cancel:     cancel,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Start 启动父节点连接
func (c *ParentConnector) Start() error {
	// 注册到父节点
	if err := c.registerToParent(); err != nil {
		return fmt.Errorf("failed to register to parent: %w", err)
	}

	// 启动心跳
	go c.heartbeatLoop()

	c.logger.WithField("parent_addr", c.parentAddr).Info("Parent connector started")

	return nil
}

// Stop 停止父节点连接
func (c *ParentConnector) Stop() error {
	c.cancel()

	// 从父节点注销
	if err := c.unregisterFromParent(); err != nil {
		c.logger.WithError(err).Warn("Failed to unregister from parent")
	}

	c.logger.Info("Parent connector stopped")
	return nil
}

// registerToParent 注册到父节点
func (c *ParentConnector) registerToParent() error {
	localRes := c.register.GetLocalResources()

	// 设置本节点的地址，让父节点知道如何委托任务过来
	localRes.Address = c.nodeAddr

	// 构造注册请求
	reqBody := map[string]interface{}{
		"node_id":   c.nodeID,
		"resources": localRes,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("http://%s/api/register", c.parentAddr)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to send register request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("register request failed with status: %d", resp.StatusCode)
	}

	// 解析父节点返回的信息
	var respData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err == nil {
		// 如果响应中包含父节点信息，将其存储到Register中
		if parentInfo, ok := respData["parent_node"].(map[string]interface{}); ok {
			parentNodeID, _ := parentInfo["node_id"].(string)
			if parentNodeID != "" {
				// 构建父节点资源信息
				parentNode := &register.NodeResources{
					NodeID:   parentNodeID,
					NodeType: "parent",
					Address:  c.parentAddr,
					Status:   "active",
					Metadata: make(map[string]string),
				}
				c.register.SetParentNode(parentNode)
			}
		}
	}

	c.logger.Info("Successfully registered to parent node")

	return nil
}

// unregisterFromParent 从父节点注销
func (c *ParentConnector) unregisterFromParent() error {
	reqBody := map[string]interface{}{
		"node_id": c.nodeID,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("http://%s/api/unregister", c.parentAddr)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to send unregister request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unregister request failed with status: %d", resp.StatusCode)
	}

	c.logger.Info("Successfully unregistered from parent node")

	return nil
}

// heartbeatLoop 心跳循环
func (c *ParentConnector) heartbeatLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if err := c.sendHeartbeat(); err != nil {
				c.logger.WithError(err).Warn("Failed to send heartbeat to parent")
			}
		}
	}
}

// sendHeartbeat 发送心跳
func (c *ParentConnector) sendHeartbeat() error {
	localRes := c.register.GetLocalResources()
	localRes.Address = c.nodeAddr

	reqBody := map[string]interface{}{
		"node_id":   c.nodeID,
		"resources": localRes,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("http://%s/api/heartbeat", c.parentAddr)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("heartbeat failed with status: %d", resp.StatusCode)
	}

	c.logger.Debug("Heartbeat sent to parent node")

	return nil
}
