package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the agent configuration
type Config struct {
	Agent     AgentConfig     `yaml:"agent"`
	Logging   LoggingConfig   `yaml:"logging"`
	Discovery DiscoveryConfig `yaml:"discovery"`
	Resources ResourcesConfig `yaml:"resources"`
	Tasks     TasksConfig     `yaml:"tasks"`
}

// AgentConfig contains agent-specific configuration
type AgentConfig struct {
	Address    string        `yaml:"address"`
	Port       int           `yaml:"port"`
	NodeID     string        `yaml:"node_id"`
	NodeName   string        `yaml:"node_name"`
	Region     string        `yaml:"region"`
	Datacenter string        `yaml:"datacenter"`
	Heartbeat  time.Duration `yaml:"heartbeat"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// DiscoveryConfig contains service discovery configuration
type DiscoveryConfig struct {
	Enabled bool          `yaml:"enabled"`
	Servers []string      `yaml:"servers"`
	Timeout time.Duration `yaml:"timeout"`
	Retry   int           `yaml:"retry"`
}

// ResourcesConfig contains resource monitoring configuration
type ResourcesConfig struct {
	CPU      bool          `yaml:"cpu"`
	Memory   bool          `yaml:"memory"`
	Disk     bool          `yaml:"disk"`
	Network  bool          `yaml:"network"`
	Interval time.Duration `yaml:"interval"`
}

// TasksConfig contains task execution configuration
type TasksConfig struct {
	MaxConcurrent int           `yaml:"max_concurrent"`
	Timeout       time.Duration `yaml:"timeout"`
	Cleanup       bool          `yaml:"cleanup"`
}

// Load loads configuration from file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if config.Agent.Address == "" {
		config.Agent.Address = "0.0.0.0"
	}
	if config.Agent.Port == 0 {
		config.Agent.Port = 8080
	}
	if config.Agent.NodeID == "" {
		config.Agent.NodeID = generateNodeID()
	}
	if config.Agent.NodeName == "" {
		config.Agent.NodeName = getHostname()
	}
	if config.Agent.Region == "" {
		config.Agent.Region = "default"
	}
	if config.Agent.Datacenter == "" {
		config.Agent.Datacenter = "dc1"
	}
	if config.Agent.Heartbeat == 0 {
		config.Agent.Heartbeat = 30 * time.Second
	}

	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}
	if config.Logging.Format == "" {
		config.Logging.Format = "json"
	}

	if config.Discovery.Timeout == 0 {
		config.Discovery.Timeout = 5 * time.Second
	}
	if config.Discovery.Retry == 0 {
		config.Discovery.Retry = 3
	}

	if config.Resources.Interval == 0 {
		config.Resources.Interval = 10 * time.Second
	}

	if config.Tasks.MaxConcurrent == 0 {
		config.Tasks.MaxConcurrent = 10
	}
	if config.Tasks.Timeout == 0 {
		config.Tasks.Timeout = 5 * time.Minute
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate agent configuration
	if c.Agent.Port <= 0 || c.Agent.Port > 65535 {
		return fmt.Errorf("invalid port: %d (must be between 1 and 65535)", c.Agent.Port)
	}

	if c.Agent.NodeID == "" {
		return fmt.Errorf("node_id cannot be empty")
	}

	if c.Agent.NodeName == "" {
		return fmt.Errorf("node_name cannot be empty")
	}

	if c.Agent.Heartbeat <= 0 {
		return fmt.Errorf("heartbeat must be positive")
	}

	// Validate logging configuration
	validLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("invalid logging level: %s (must be debug, info, warn, or error)", c.Logging.Level)
	}

	validFormats := map[string]bool{
		"json": true, "text": true,
	}
	if !validFormats[c.Logging.Format] {
		return fmt.Errorf("invalid logging format: %s (must be json or text)", c.Logging.Format)
	}

	// Validate discovery configuration
	if c.Discovery.Enabled {
		// Allow empty servers list for root discovery servers
		// Only require servers if this is not a discovery server node
		if len(c.Discovery.Servers) == 0 && c.Agent.NodeID != "discovery-server" {
			return fmt.Errorf("discovery servers cannot be empty when discovery is enabled")
		}

		if c.Discovery.Timeout <= 0 {
			return fmt.Errorf("discovery timeout must be positive")
		}

		if c.Discovery.Retry < 0 {
			return fmt.Errorf("discovery retry count cannot be negative")
		}
	}

	// Validate resources configuration
	if c.Resources.Interval <= 0 {
		return fmt.Errorf("resources interval must be positive")
	}

	// Validate tasks configuration
	if c.Tasks.MaxConcurrent <= 0 {
		return fmt.Errorf("max_concurrent must be positive")
	}

	if c.Tasks.Timeout <= 0 {
		return fmt.Errorf("tasks timeout must be positive")
	}

	return nil
}

func generateNodeID() string {
	hostname, _ := os.Hostname()
	return fmt.Sprintf("%s-%d", hostname, time.Now().Unix())
}

func getHostname() string {
	hostname, _ := os.Hostname()
	return hostname
}
