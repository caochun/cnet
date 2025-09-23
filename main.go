package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cnet/internal/agent"
	"cnet/internal/config"
	"cnet/internal/logger"
)

func main() {
	var configPath = flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger := logger.New(cfg.Logging.Level)

	// Create agent
	agent, err := agent.New(cfg, logger)
	if err != nil {
		logger.Fatalf("Failed to create agent: %v", err)
	}

	// Start agent
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start agent services
	if err := agent.Start(ctx); err != nil {
		logger.Fatalf("Failed to start agent: %v", err)
	}

	logger.Infof("CNET Agent started on %s:%d", cfg.Agent.Address, cfg.Agent.Port)

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Agent.Address, cfg.Agent.Port),
		Handler: agent.Handler(),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	logger.Info("Shutting down agent...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Errorf("Server shutdown error: %v", err)
	}

	if err := agent.Stop(); err != nil {
		logger.Errorf("Agent stop error: %v", err)
	}

	logger.Info("Agent stopped")
}
