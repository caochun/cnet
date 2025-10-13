package scheduler

import (
	"cnet/internal/register"
)

// ScheduleStrategy 调度策略接口
type ScheduleStrategy interface {
	// MakeDecision 根据资源情况做出调度决策
	MakeDecision(reg *register.Register, req register.ResourceRequirements) *ScheduleDecision
}

// DefaultScheduleStrategy 默认调度策略
// 策略：
// 1. 优先本地资源
// 2. 本地资源不足时，选择下级节点
// 3. 下级节点也不足时，选择同级peer节点
type DefaultScheduleStrategy struct{}

// MakeDecision 实现调度决策
func (s *DefaultScheduleStrategy) MakeDecision(reg *register.Register, req register.ResourceRequirements) *ScheduleDecision {
	// 1. 检查本地资源
	localRes := reg.GetLocalResources()
	if localRes.Available.CanSatisfy(req) {
		return &ScheduleDecision{
			Action:   "local",
			NodeID:   localRes.NodeID,
			NodeAddr: localRes.Address,
			Reason:   "Local resources available",
		}
	}

	// 2. 检查下级节点
	childNodes := reg.GetChildNodes()
	for _, node := range childNodes {
		if node.Status == "active" && node.Available.CanSatisfy(req) {
			return &ScheduleDecision{
				Action:   "delegate_child",
				NodeID:   node.NodeID,
				NodeAddr: node.Address,
				Reason:   "Delegating to child node with sufficient resources",
			}
		}
	}

	// 3. 检查同级peer节点
	peerNodes := reg.GetPeerNodes()
	for _, node := range peerNodes {
		if node.Status == "active" && node.Available.CanSatisfy(req) {
			return &ScheduleDecision{
				Action:   "delegate_peer",
				NodeID:   node.NodeID,
				NodeAddr: node.Address,
				Reason:   "Delegating to peer node with sufficient resources",
			}
		}
	}

	// 4. 无可用资源
	return &ScheduleDecision{
		Action: "reject",
		Reason: "Insufficient resources in cluster",
	}
}

// BestFitStrategy 最佳适配策略
// 选择资源最接近需求的节点（避免资源浪费）
type BestFitStrategy struct{}

// MakeDecision 实现最佳适配调度
func (s *BestFitStrategy) MakeDecision(reg *register.Register, req register.ResourceRequirements) *ScheduleDecision {
	type candidate struct {
		decision *ScheduleDecision
		score    float64 // 资源匹配度分数，越小越好
	}

	var best *candidate

	// 计算资源匹配分数
	calcScore := func(available register.Resources) float64 {
		// 使用CPU和内存的剩余比例作为分数
		cpuRatio := (available.CPU - req.CPU) / available.CPU
		memRatio := float64(available.Memory-req.Memory) / float64(available.Memory)
		return cpuRatio + memRatio
	}

	// 检查本地资源
	localRes := reg.GetLocalResources()
	if localRes.Available.CanSatisfy(req) {
		score := calcScore(localRes.Available)
		best = &candidate{
			decision: &ScheduleDecision{
				Action:   "local",
				NodeID:   localRes.NodeID,
				NodeAddr: localRes.Address,
				Reason:   "Best fit: local resources",
			},
			score: score,
		}
	}

	// 检查下级节点
	childNodes := reg.GetChildNodes()
	for _, node := range childNodes {
		if node.Status == "active" && node.Available.CanSatisfy(req) {
			score := calcScore(node.Available)
			if best == nil || score < best.score {
				best = &candidate{
					decision: &ScheduleDecision{
						Action:   "delegate_child",
						NodeID:   node.NodeID,
						NodeAddr: node.Address,
						Reason:   "Best fit: child node",
					},
					score: score,
				}
			}
		}
	}

	// 检查同级peer节点
	peerNodes := reg.GetPeerNodes()
	for _, node := range peerNodes {
		if node.Status == "active" && node.Available.CanSatisfy(req) {
			score := calcScore(node.Available)
			if best == nil || score < best.score {
				best = &candidate{
					decision: &ScheduleDecision{
						Action:   "delegate_peer",
						NodeID:   node.NodeID,
						NodeAddr: node.Address,
						Reason:   "Best fit: peer node",
					},
					score: score,
				}
			}
		}
	}

	if best != nil {
		return best.decision
	}

	return &ScheduleDecision{
		Action: "reject",
		Reason: "Insufficient resources in cluster",
	}
}
