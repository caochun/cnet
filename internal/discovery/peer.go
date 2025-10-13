package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"cnet/internal/register"

	"github.com/sirupsen/logrus"
)

// PeerDiscovery peer节点发现
type PeerDiscovery struct {
	logger     *logrus.Logger
	register   *register.Register
	peerAddrs  []string
	nodeID     string
	ctx        context.Context
	cancel     context.CancelFunc
	httpClient *http.Client
	mu         sync.RWMutex
}

// NewPeerDiscovery 创建peer发现服务
func NewPeerDiscovery(logger *logrus.Logger, reg *register.Register, peerAddrs []string, nodeID string) *PeerDiscovery {
	ctx, cancel := context.WithCancel(context.Background())

	return &PeerDiscovery{
		logger:    logger,
		register:  reg,
		peerAddrs: peerAddrs,
		nodeID:    nodeID,
		ctx:       ctx,
		cancel:    cancel,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Start 启动peer发现
func (d *PeerDiscovery) Start() error {
	// 注册到所有peer节点
	for _, peerAddr := range d.peerAddrs {
		if err := d.registerToPeer(peerAddr); err != nil {
			d.logger.WithError(err).WithField("peer_addr", peerAddr).Warn("Failed to register to peer")
		}
	}

	// 启动peer信息同步
	go d.syncLoop()

	d.logger.WithField("peer_count", len(d.peerAddrs)).Info("Peer discovery started")

	return nil
}

// Stop 停止peer发现
func (d *PeerDiscovery) Stop() error {
	d.cancel()

	// 从所有peer节点注销
	for _, peerAddr := range d.peerAddrs {
		if err := d.unregisterFromPeer(peerAddr); err != nil {
			d.logger.WithError(err).WithField("peer_addr", peerAddr).Warn("Failed to unregister from peer")
		}
	}

	d.logger.Info("Peer discovery stopped")
	return nil
}

// registerToPeer 注册到peer节点
func (d *PeerDiscovery) registerToPeer(peerAddr string) error {
	localRes := d.register.GetLocalResources()

	reqBody := map[string]interface{}{
		"node_id":   d.nodeID,
		"resources": localRes,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("http://%s/api/peer/register", peerAddr)
	resp, err := d.httpClient.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to send register request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("register request failed with status: %d", resp.StatusCode)
	}

	d.logger.WithField("peer_addr", peerAddr).Info("Successfully registered to peer")

	return nil
}

// unregisterFromPeer 从peer节点注销
func (d *PeerDiscovery) unregisterFromPeer(peerAddr string) error {
	reqBody := map[string]interface{}{
		"node_id": d.nodeID,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("http://%s/api/peer/unregister", peerAddr)
	resp, err := d.httpClient.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to send unregister request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unregister request failed with status: %d", resp.StatusCode)
	}

	d.logger.WithField("peer_addr", peerAddr).Info("Successfully unregistered from peer")

	return nil
}

// syncLoop 同步循环
func (d *PeerDiscovery) syncLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			d.syncWithPeers()
		}
	}
}

// syncWithPeers 与peer节点同步
func (d *PeerDiscovery) syncWithPeers() {
	for _, peerAddr := range d.peerAddrs {
		if err := d.syncWithPeer(peerAddr); err != nil {
			d.logger.WithError(err).WithField("peer_addr", peerAddr).Debug("Failed to sync with peer")
		}
	}
}

// syncWithPeer 与单个peer节点同步
func (d *PeerDiscovery) syncWithPeer(peerAddr string) error {
	url := fmt.Sprintf("http://%s/api/resources", peerAddr)
	resp, err := d.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch peer resources: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch resources failed with status: %d", resp.StatusCode)
	}

	var resources register.NodeResources
	if err := json.NewDecoder(resp.Body).Decode(&resources); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// 更新peer节点资源信息
	resources.Address = peerAddr
	if err := d.register.UpdateNodeResources(resources.NodeID, &resources); err != nil {
		// 如果节点不存在，注册新节点
		if err := d.register.RegisterPeerNode(resources.NodeID, &resources); err != nil {
			return fmt.Errorf("failed to register peer node: %w", err)
		}
	}

	return nil
}

// AddPeer 添加新的peer节点
func (d *PeerDiscovery) AddPeer(peerAddr string) error {
	d.mu.Lock()
	d.peerAddrs = append(d.peerAddrs, peerAddr)
	d.mu.Unlock()

	// 注册到新的peer
	if err := d.registerToPeer(peerAddr); err != nil {
		return fmt.Errorf("failed to register to new peer: %w", err)
	}

	d.logger.WithField("peer_addr", peerAddr).Info("New peer added")

	return nil
}
