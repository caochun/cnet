package tasks

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"cnet/internal/logger"
)

// MLExecutor executes ML inference tasks
type MLExecutor struct {
	logger *logger.Logger
}

// NewMLExecutor creates a new MLExecutor
func NewMLExecutor(logger *logger.Logger) *MLExecutor {
	return &MLExecutor{
		logger: logger,
	}
}

// Execute executes an ML inference task
func (e *MLExecutor) Execute(ctx context.Context, task *Task) error {
	// Create ML inference server script
	scriptPath := filepath.Join(os.TempDir(), "cnet", "ml_servers", task.ID+".py")
	if err := e.createInferenceServerScript(scriptPath, task); err != nil {
		return fmt.Errorf("failed to create inference server script: %w", err)
	}

	// Start the inference server
	cmd := exec.CommandContext(ctx, "python3", scriptPath)

	// Set working directory
	if task.WorkingDir != "" {
		cmd.Dir = task.WorkingDir
	}

	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range task.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Create log file
	logFile, err := os.Create(task.LogFile)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer logFile.Close()

	// Set up output
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Start process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ML inference server: %w", err)
	}

	task.Process = cmd.Process

	// Wait for completion
	err = cmd.Wait()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode := exitError.ExitCode()
			task.ExitCode = &exitCode
		}
		return err
	}

	exitCode := 0
	task.ExitCode = &exitCode
	return nil
}

// Stop stops an ML inference task
func (e *MLExecutor) Stop(task *Task) error {
	if task.Process != nil {
		return task.Process.Kill()
	}
	return nil
}

// GetLogs retrieves logs for an ML inference task
func (e *MLExecutor) GetLogs(task *Task, lines int) ([]string, error) {
	file, err := os.Open(task.LogFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// Read all lines
	var allLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	// Return last N lines
	start := 0
	if len(allLines) > lines {
		start = len(allLines) - lines
	}

	return allLines[start:], nil
}

// createInferenceServerScript creates a Python script for ML inference server
func (e *MLExecutor) createInferenceServerScript(outputPath string, task *Task) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Extract model path and script from task args
	var modelPath, scriptPath string
	if len(task.Args) >= 2 {
		scriptPath = task.Args[0]
		modelPath = task.Args[1]
	}

	// Create the inference server script
	script := fmt.Sprintf(`#!/usr/bin/env python3
"""
CNET Agent ML Inference Server
Generated for task: %s
"""

import sys
import os
import json
import time
import signal
import threading
from http.server import HTTPServer, BaseHTTPRequestHandler
from urllib.parse import urlparse, parse_qs
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class MLInferenceHandler(BaseHTTPRequestHandler):
    def __init__(self, *args, **kwargs):
        self.model = None
        self.model_path = "%s"
        self.script_path = "%s"
        super().__init__(*args, **kwargs)
    
    def do_GET(self):
        """Handle health check requests"""
        if self.path == '/health':
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            response = {
                "status": "healthy",
                "model_loaded": self.model is not None,
                "timestamp": time.time()
            }
            self.wfile.write(json.dumps(response).encode())
        else:
            self.send_response(404)
            self.end_headers()
    
    def do_POST(self):
        """Handle inference requests"""
        if self.path == '/predict':
            try:
                content_length = int(self.headers['Content-Length'])
                post_data = self.rfile.read(content_length)
                request_data = json.loads(post_data.decode('utf-8'))
                
                # Perform inference
                result = self.perform_inference(request_data)
                
                self.send_response(200)
                self.send_header('Content-type', 'application/json')
                self.end_headers()
                self.wfile.write(json.dumps(result).encode())
                
            except Exception as e:
                logger.error("Inference error: %%s", str(e))
                self.send_response(500)
                self.send_header('Content-type', 'application/json')
                self.end_headers()
                error_response = {
                    "error": str(e),
                    "status": "error"
                }
                self.wfile.write(json.dumps(error_response).encode())
        else:
            self.send_response(404)
            self.end_headers()
    
    def perform_inference(self, request_data):
        """Perform ML inference"""
        if self.model is None:
            raise Exception("Model not loaded")
        
        # Extract input data
        input_data = request_data.get('input_data')
        if input_data is None:
            raise Exception("No input_data provided")
        
        # TODO: Implement actual inference logic
        # This is a placeholder implementation
        prediction = {
            "prediction": f"predicted_value_for_{input_data}",
            "confidence": 0.95,
            "model_type": "placeholder",
            "timestamp": time.time()
        }
        
        return prediction
    
    def load_model(self):
        """Load the ML model"""
        try:
            # TODO: Implement actual model loading
            # This would load the model from self.model_path
            logger.info("Loading model from: %%s", self.model_path)
            
            # For now, just mark as loaded
            self.model = "loaded"
            logger.info("Model loaded successfully")
            
        except Exception as e:
            logger.error("Failed to load model: %%s", str(e))
            raise e

def signal_handler(signum, frame):
    """Handle shutdown signals"""
    logger.info("Received shutdown signal, stopping server...")
    sys.exit(0)

def main():
    # Set up signal handlers
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)
    
    # Get port from environment or use default
    import socket
    def find_free_port():
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
            s.bind(('', 0))
            s.listen(1)
            port = s.getsockname()[1]
        return port
    
    port = int(os.environ.get('ML_SERVER_PORT', str(find_free_port())))
    
    # Create server
    server = HTTPServer(('localhost', port), MLInferenceHandler)
    
    # Global variables for model paths
    MODEL_PATH = "%s"
    SCRIPT_PATH = "%s"
    
    # Load model in a separate thread
    def load_model_thread():
        # Create a temporary handler to load the model
        class TempHandler(MLInferenceHandler):
            def __init__(self):
                self.model = None
                self.model_path = MODEL_PATH
                self.script_path = SCRIPT_PATH
        
        handler = TempHandler()
        handler.load_model()
        # Update the server's handler class with the loaded model
        original_handler = server.RequestHandlerClass
        def create_handler(*args, **kwargs):
            h = original_handler(*args, **kwargs)
            h.model = handler.model
            h.model_path = handler.model_path
            h.script_path = handler.script_path
            return h
        server.RequestHandlerClass = create_handler
    
    # Start model loading
    model_thread = threading.Thread(target=load_model_thread)
    model_thread.daemon = True
    model_thread.start()
    
    logger.info("Starting ML inference server on port %%d", port)
    
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        logger.info("Server stopped by user")
    except Exception as e:
        logger.error("Server error: %%s", str(e))
        sys.exit(1)

if __name__ == "__main__":
    main()
`, task.ID, modelPath, scriptPath, modelPath, scriptPath)

	// Write script to file
	if err := os.WriteFile(outputPath, []byte(script), 0755); err != nil {
		return fmt.Errorf("failed to write script file: %w", err)
	}

	return nil
}
