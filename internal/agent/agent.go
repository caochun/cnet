package agent

import (
	"context"
	"fmt"

	"cnet/internal/discovery"
	"cnet/internal/executor"
	"cnet/internal/manager"
	"cnet/internal/register"
	"cnet/internal/scheduler"
	"cnet/internal/workload"

	"github.com/sirupsen/logrus"
)

// AgentConfig Agent配置
type AgentConfig struct {
	NodeID    string
	Address   string
	Port      int
	Resources register.Resources

	// 父节点配置
	ParentEnabled bool
	ParentAddr    string

	// Peer节点配置
	PeerEnabled bool
	PeerAddrs   []string
}

// Agent CNET Agent
type Agent struct {
	config *AgentConfig
	logger *logrus.Logger
	ctx    context.Context
	cancel context.CancelFunc

	// 核心组件
	register    *register.Register
	scheduler   *scheduler.Scheduler
	manager     *manager.Manager
	execFactory *executor.ExecutorFactory

	// 节点发现
	parentConn *discovery.ParentConnector
	peerDisc   *discovery.PeerDiscovery
}

// NewAgent 创建新的Agent
func NewAgent(config *AgentConfig, logger *logrus.Logger) (*Agent, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// 创建Register
	reg := register.NewRegister(config.NodeID, config.Resources, logger)

	// 创建Executor工厂
	execFactory := executor.NewExecutorFactory()
	execFactory.Register(workload.TypeProcess, executor.NewProcessExecutor(logger))
	execFactory.Register(workload.TypeContainer, executor.NewContainerExecutor(logger))
	execFactory.Register(workload.TypeMLModel, executor.NewMLModelExecutor(logger))
	execFactory.Register(workload.TypeVision, executor.NewVisionExecutor(logger))

	// 创建Scheduler
	sched := scheduler.NewScheduler(logger, reg, execFactory)

	// 创建Manager
	mgr := manager.NewManager(logger, sched, reg)

	agent := &Agent{
		config:      config,
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
		register:    reg,
		scheduler:   sched,
		manager:     mgr,
		execFactory: execFactory,
	}

	// 创建父节点连接器（如果启用）
	if config.ParentEnabled && config.ParentAddr != "" {
		// 构造本节点地址 (address:port)
		nodeAddr := fmt.Sprintf("%s:%d", config.Address, config.Port)
		// 如果address是0.0.0.0或空，使用localhost
		if config.Address == "0.0.0.0" || config.Address == "" {
			nodeAddr = fmt.Sprintf("localhost:%d", config.Port)
		}
		agent.parentConn = discovery.NewParentConnector(logger, reg, config.ParentAddr, config.NodeID, nodeAddr)
	}

	// 创建Peer发现（如果启用）
	if config.PeerEnabled && len(config.PeerAddrs) > 0 {
		agent.peerDisc = discovery.NewPeerDiscovery(logger, reg, config.PeerAddrs, config.NodeID)
	}

	return agent, nil
}

// Start 启动Agent
func (a *Agent) Start() error {
	a.logger.Info("Starting CNET Agent...")

	// 初始化所有Executor
	if err := a.execFactory.InitAll(a.ctx); err != nil {
		return fmt.Errorf("failed to initialize executors: %w", err)
	}
	a.logger.Info("All executors initialized")

	// 启动Register
	if err := a.register.Start(); err != nil {
		return fmt.Errorf("failed to start register: %w", err)
	}
	a.logger.Info("Register started")

	// 启动父节点连接（如果启用）
	if a.parentConn != nil {
		if err := a.parentConn.Start(); err != nil {
			a.logger.WithError(err).Warn("Failed to start parent connector")
			// 不返回错误，允许agent独立运行
		} else {
			a.logger.Info("Parent connector started")
		}
	}

	// 启动Peer发现（如果启用）
	if a.peerDisc != nil {
		if err := a.peerDisc.Start(); err != nil {
			a.logger.WithError(err).Warn("Failed to start peer discovery")
			// 不返回错误，允许agent独立运行
		} else {
			a.logger.Info("Peer discovery started")
		}
	}

	a.logger.WithFields(logrus.Fields{
		"node_id": a.config.NodeID,
		"address": fmt.Sprintf("%s:%d", a.config.Address, a.config.Port),
	}).Info("CNET Agent started successfully")

	return nil
}

// Stop 停止Agent
func (a *Agent) Stop() error {
	a.logger.Info("Stopping CNET Agent...")

	// 停止Peer发现
	if a.peerDisc != nil {
		if err := a.peerDisc.Stop(); err != nil {
			a.logger.WithError(err).Warn("Failed to stop peer discovery")
		}
	}

	// 停止父节点连接
	if a.parentConn != nil {
		if err := a.parentConn.Stop(); err != nil {
			a.logger.WithError(err).Warn("Failed to stop parent connector")
		}
	}

	// 停止Register
	if err := a.register.Stop(); err != nil {
		a.logger.WithError(err).Warn("Failed to stop register")
	}

	a.cancel()
	a.logger.Info("CNET Agent stopped")

	return nil
}

// GetManager 获取Manager
func (a *Agent) GetManager() *manager.Manager {
	return a.manager
}

// GetRegister 获取Register
func (a *Agent) GetRegister() *register.Register {
	return a.register
}

// GetScheduler 获取Scheduler
func (a *Agent) GetScheduler() *scheduler.Scheduler {
	return a.scheduler
}
