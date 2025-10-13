package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cnet/internal/agent"
	"cnet/internal/manager"
	"cnet/internal/register"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// Config 配置结构
type Config struct {
	Agent struct {
		NodeID  string `yaml:"node_id"`
		Address string `yaml:"address"`
		Port    int    `yaml:"port"`
	} `yaml:"agent"`

	Resources struct {
		CPU     float64 `yaml:"cpu"`
		GPU     int     `yaml:"gpu"`
		Memory  int64   `yaml:"memory"`
		Storage int64   `yaml:"storage"`
	} `yaml:"resources"`

	Parent struct {
		Enabled bool   `yaml:"enabled"`
		Address string `yaml:"address"`
	} `yaml:"parent"`

	Peers struct {
		Enabled   bool     `yaml:"enabled"`
		Addresses []string `yaml:"addresses"`
	} `yaml:"peers"`

	Logging struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
	} `yaml:"logging"`
}

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// 加载配置
	config, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 创建logger
	logger := createLogger(config.Logging.Level, config.Logging.Format)

	// 创建Agent配置
	agentConfig := &agent.AgentConfig{
		NodeID:  config.Agent.NodeID,
		Address: config.Agent.Address,
		Port:    config.Agent.Port,
		Resources: register.Resources{
			CPU:     config.Resources.CPU,
			GPU:     config.Resources.GPU,
			Memory:  config.Resources.Memory,
			Storage: config.Resources.Storage,
		},
		ParentEnabled: config.Parent.Enabled,
		ParentAddr:    config.Parent.Address,
		PeerEnabled:   config.Peers.Enabled,
		PeerAddrs:     config.Peers.Addresses,
	}

	// 创建Agent
	ag, err := agent.NewAgent(agentConfig, logger)
	if err != nil {
		logger.Fatalf("Failed to create agent: %v", err)
	}

	// 启动Agent
	if err := ag.Start(); err != nil {
		logger.Fatalf("Failed to start agent: %v", err)
	}

	// 创建HTTP API
	api := manager.NewAPI(ag.GetManager(), ag.GetRegister(), logger)

	// 启动HTTP服务器
	addr := fmt.Sprintf("%s:%d", config.Agent.Address, config.Agent.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: api.GetRouter(),
	}

	// 启动服务器
	go func() {
		logger.Infof("CNET Agent HTTP server listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("HTTP server error: %v", err)
		}
	}()

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down agent...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Errorf("Server shutdown error: %v", err)
	}

	if err := ag.Stop(); err != nil {
		logger.Errorf("Agent stop error: %v", err)
	}

	logger.Info("Agent stopped gracefully")
}

// loadConfig 加载配置文件
func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 设置默认值
	if config.Agent.Address == "" {
		config.Agent.Address = "0.0.0.0"
	}
	if config.Agent.Port == 0 {
		config.Agent.Port = 8080
	}
	if config.Agent.NodeID == "" {
		hostname, _ := os.Hostname()
		config.Agent.NodeID = fmt.Sprintf("%s-%d", hostname, time.Now().Unix())
	}
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}
	if config.Logging.Format == "" {
		config.Logging.Format = "json"
	}

	return &config, nil
}

// createLogger 创建logger
func createLogger(level, format string) *logrus.Logger {
	logger := logrus.New()

	// 设置日志级别
	switch level {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	// 设置日志格式
	if format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
	}

	return logger
}
