package scheduler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"cnet/internal/register"
	"cnet/internal/workload"
)

// DelegateWorkloadRequest 委托workload请求
type DelegateWorkloadRequest struct {
	Name         string                        `json:"name"`
	Type         workload.WorkloadType         `json:"type"`
	Requirements register.ResourceRequirements `json:"requirements"`
	Config       map[string]interface{}        `json:"config"`
}

// httpClient HTTP客户端
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// delegateWorkload 委托workload到远程节点
func (s *Scheduler) delegateWorkload(ctx context.Context, w workload.Workload, nodeAddr string) error {
	metadata := w.GetMetadata()
	
	// 获取workload名称
	name := fmt.Sprintf("delegated-%s", w.GetID())
	if nameVal, ok := metadata["name"]; ok && nameVal != nil {
		if nameStr, ok := nameVal.(string); ok && nameStr != "" {
			name = nameStr
		}
	}
	
	// 构造委托请求
	req := DelegateWorkloadRequest{
		Name:         name,
		Type:         w.GetType(),
		Requirements: w.GetResourceRequirements(),
		Config:       metadata,
	}
	
	// 序列化请求
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// 发送到目标节点
	url := fmt.Sprintf("http://%s/api/workloads", nodeAddr)
	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to send request to %s: %w", nodeAddr, err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("delegation failed with status: %d", resp.StatusCode)
	}
	
	// 解析响应
	var remoteWorkload map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&remoteWorkload); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	
	remoteID, idOk := remoteWorkload["id"].(string)
	if !idOk {
		return fmt.Errorf("invalid response: missing workload id")
	}
	
	s.logger.WithFields(map[string]interface{}{
		"local_workload_id":  w.GetID(),
		"remote_workload_id": remoteID,
		"target_node":        nodeAddr,
	}).Info("Workload delegated successfully")
	
	// 更新本地workload状态
	w.SetStatus(workload.StatusRunning)
	
	// 记录委托信息到metadata
	metadata["delegated"] = true
	metadata["delegated_to"] = nodeAddr
	metadata["remote_workload_id"] = remoteID
	
	return nil
}

// DelegateToChild 委托给下级节点（实现版本）
func (s *Scheduler) DelegateToChild(ctx context.Context, w workload.Workload, nodeID string) error {
	// 从register中获取子节点信息
	childNodes := s.register.GetChildNodes()
	
	var targetNode *register.NodeResources
	for _, node := range childNodes {
		if node.NodeID == nodeID {
			targetNode = node
			break
		}
	}
	
	if targetNode == nil {
		return fmt.Errorf("child node not found: %s", nodeID)
	}
	
	if targetNode.Status != "active" {
		return fmt.Errorf("child node is not active: %s (status: %s)", nodeID, targetNode.Status)
	}
	
	s.logger.WithFields(map[string]interface{}{
		"workload_id": w.GetID(),
		"target_node": nodeID,
		"node_addr":   targetNode.Address,
	}).Info("Delegating workload to child node")
	
	// 执行委托
	return s.delegateWorkload(ctx, w, targetNode.Address)
}

// DelegateToPeer 委托给同级节点（实现版本）
func (s *Scheduler) DelegateToPeer(ctx context.Context, w workload.Workload, nodeID string) error {
	// 从register中获取peer节点信息
	peerNodes := s.register.GetPeerNodes()
	
	var targetNode *register.NodeResources
	for _, node := range peerNodes {
		if node.NodeID == nodeID {
			targetNode = node
			break
		}
	}
	
	if targetNode == nil {
		return fmt.Errorf("peer node not found: %s", nodeID)
	}
	
	if targetNode.Status != "active" {
		return fmt.Errorf("peer node is not active: %s (status: %s)", nodeID, targetNode.Status)
	}
	
	s.logger.WithFields(map[string]interface{}{
		"workload_id": w.GetID(),
		"target_node": nodeID,
		"node_addr":   targetNode.Address,
	}).Info("Delegating workload to peer node")
	
	// 执行委托
	return s.delegateWorkload(ctx, w, targetNode.Address)
}

