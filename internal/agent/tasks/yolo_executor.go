package tasks

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"cnet/internal/logger"
)

// YOLOExecutor executes YOLO inference tasks
type YOLOExecutor struct {
	logger *logger.Logger
}

// YOLOConfig represents YOLO model configuration
type YOLOConfig struct {
	ModelPath      string                 `json:"model_path"`
	ModelType      string                 `json:"model_type"`     // yolov5, yolov8, etc.
	Confidence     float64                `json:"confidence"`     // confidence threshold
	IOUThreshold   float64                `json:"iou_threshold"`  // IoU threshold for NMS
	ImageSize      int                    `json:"image_size"`     // input image size
	MaxDetections  int                    `json:"max_detections"` // maximum number of detections
	Classes        []string               `json:"classes"`        // class names
	Preprocessing  map[string]interface{} `json:"preprocessing"`
	Postprocessing map[string]interface{} `json:"postprocessing"`
}

// YOLODetection represents a single detection result
type YOLODetection struct {
	ClassID     int     `json:"class_id"`
	ClassName   string  `json:"class_name"`
	Confidence  float64 `json:"confidence"`
	BoundingBox struct {
		X      float64 `json:"x"`      // center x
		Y      float64 `json:"y"`      // center y
		Width  float64 `json:"width"`  // width
		Height float64 `json:"height"` // height
	} `json:"bounding_box"`
}

// YOLOInferenceRequest represents a YOLO inference request
type YOLOInferenceRequest struct {
	ImagePath string                 `json:"image_path,omitempty"`
	ImageData string                 `json:"image_data,omitempty"` // base64 encoded image
	ImageURL  string                 `json:"image_url,omitempty"`
	Config    YOLOConfig             `json:"config,omitempty"`
	Options   map[string]interface{} `json:"options,omitempty"`
}

// YOLOInferenceResponse represents a YOLO inference response
type YOLOInferenceResponse struct {
	Detections     []YOLODetection        `json:"detections"`
	ImageInfo      map[string]interface{} `json:"image_info"`
	ProcessingTime float64                `json:"processing_time"`
	ModelInfo      map[string]interface{} `json:"model_info"`
	Timestamp      time.Time              `json:"timestamp"`
}

// NewYOLOExecutor creates a new YOLOExecutor
func NewYOLOExecutor(logger *logger.Logger) *YOLOExecutor {
	return &YOLOExecutor{
		logger: logger,
	}
}

// Execute executes a YOLO inference task
func (e *YOLOExecutor) Execute(ctx context.Context, task *Task) error {
	// Create YOLO inference server script
	scriptPath := filepath.Join(os.TempDir(), "cnet", "yolo_servers", task.ID+".py")
	if err := e.createYOLOInferenceServerScript(scriptPath, task); err != nil {
		return fmt.Errorf("failed to create YOLO inference server script: %w", err)
	}

	// Start the YOLO inference server
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
		return fmt.Errorf("failed to start YOLO inference server: %w", err)
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

// Stop stops a YOLO inference task
func (e *YOLOExecutor) Stop(task *Task) error {
	if task.Process != nil {
		return task.Process.Kill()
	}
	return nil
}

// GetLogs retrieves logs for a YOLO inference task
func (e *YOLOExecutor) GetLogs(task *Task, lines int) ([]string, error) {
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

// createYOLOInferenceServerScript creates a Python script for YOLO inference server
func (e *YOLOExecutor) createYOLOInferenceServerScript(outputPath string, task *Task) error {
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

	// Create the YOLO inference server script
	script := fmt.Sprintf(`#!/usr/bin/env python3
"""
CNET Agent YOLO Inference Server
Generated for task: %s
"""

import sys
import os
import json
import time
import signal
import threading
import base64
import cv2
import numpy as np
from http.server import HTTPServer, BaseHTTPRequestHandler
from urllib.parse import urlparse, parse_qs
import logging
from typing import List, Dict, Any, Optional
import requests
from PIL import Image
import io

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class YOLOInferenceHandler(BaseHTTPRequestHandler):
    def __init__(self, *args, **kwargs):
        self.model = None
        self.model_path = "%s"
        self.script_path = "%s"
        self.classes = []
        self.model_type = "yolov8"
        self.confidence_threshold = 0.5
        self.iou_threshold = 0.45
        self.image_size = 640
        self.max_detections = 100
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
                "model_type": self.model_type,
                "timestamp": time.time()
            }
            self.wfile.write(json.dumps(response).encode())
        elif self.path == '/model_info':
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            info = {
                "model_path": self.model_path,
                "model_type": self.model_type,
                "classes": self.classes,
                "confidence_threshold": self.confidence_threshold,
                "iou_threshold": self.iou_threshold,
                "image_size": self.image_size,
                "max_detections": self.max_detections
            }
            self.wfile.write(json.dumps(info).encode())
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
                
                # Perform YOLO inference
                result = self.perform_yolo_inference(request_data)
                
                self.send_response(200)
                self.send_header('Content-type', 'application/json')
                self.end_headers()
                self.wfile.write(json.dumps(result).encode())
                
            except Exception as e:
                logger.error("YOLO inference error: %%s", str(e))
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
    
    def perform_yolo_inference(self, request_data):
        """Perform YOLO inference"""
        if self.model is None:
            raise Exception("YOLO model not loaded")
        
        start_time = time.time()
        
        # Extract input data
        image_path = request_data.get('image_path')
        image_data = request_data.get('image_data')
        image_url = request_data.get('image_url')
        
        # Load image
        image = self.load_image(image_path, image_data, image_url)
        if image is None:
            raise Exception("Failed to load image")
        
        # Get image info
        image_info = {
            "width": image.shape[1],
            "height": image.shape[0],
            "channels": image.shape[2] if len(image.shape) > 2 else 1
        }
        
        # Perform YOLO inference
        detections = self.run_yolo_inference(image, request_data)
        
        processing_time = time.time() - start_time
        
        response = {
            "detections": detections,
            "image_info": image_info,
            "processing_time": processing_time,
            "model_info": {
                "model_type": self.model_type,
                "confidence_threshold": self.confidence_threshold,
                "iou_threshold": self.iou_threshold
            },
            "timestamp": time.time()
        }
        
        return response
    
    def load_image(self, image_path=None, image_data=None, image_url=None):
        """Load image from various sources"""
        try:
            if image_path:
                # Load from file path
                image = cv2.imread(image_path)
                if image is None:
                    raise Exception(f"Failed to load image from path: {image_path}")
                return cv2.cvtColor(image, cv2.COLOR_BGR2RGB)
            
            elif image_data:
                # Load from base64 data
                image_bytes = base64.b64decode(image_data)
                image = Image.open(io.BytesIO(image_bytes))
                return cv2.cvtColor(np.array(image), cv2.COLOR_RGB2BGR)
            
            elif image_url:
                # Load from URL
                response = requests.get(image_url)
                response.raise_for_status()
                image = Image.open(io.BytesIO(response.content))
                return cv2.cvtColor(np.array(image), cv2.COLOR_RGB2BGR)
            
            else:
                raise Exception("No image source provided")
                
        except Exception as e:
            logger.error("Failed to load image: %%s", str(e))
            return None
    
    def run_yolo_inference(self, image, request_data):
        """Run YOLO inference on the image using Ultralytics YOLO"""
        try:
            # Update parameters from request
            config = request_data.get('config', {})
            if 'confidence' in config:
                self.confidence_threshold = config['confidence']
            if 'iou_threshold' in config:
                self.iou_threshold = config['iou_threshold']
            if 'image_size' in config:
                self.image_size = config['image_size']
            
            # Import Ultralytics YOLO
            try:
                from ultralytics import YOLO
            except ImportError:
                raise Exception("Ultralytics YOLO not installed. Please install with: pip install ultralytics")
            
            # Load YOLO model if not already loaded
            if self.model is None or self.model == "yolo_model_loaded":
                logger.info("Loading YOLO model from: %s", self.model_path)
                self.model = YOLO(self.model_path)
                logger.info("YOLO model loaded successfully")
            
            # Prepare inference parameters
            inference_params = {
                'conf': self.confidence_threshold,
                'iou': self.iou_threshold,
                'imgsz': self.image_size,
                'max_det': self.max_detections,
                'device': 'cpu'  # Can be changed to 'cuda' if GPU is available
            }
            
            # Add custom parameters from request
            if 'device' in config:
                inference_params['device'] = config['device']
            if 'half' in config:
                inference_params['half'] = config['half']
            if 'dnn' in config:
                inference_params['dnn'] = config['dnn']
            if 'vid_stride' in config:
                inference_params['vid_stride'] = config['vid_stride']
            if 'stream' in config:
                inference_params['stream'] = config['stream']
            
            # Run YOLO inference
            logger.info("Running YOLO inference with parameters: %s", inference_params)
            results = self.model(image, **inference_params)
            
            # Process results
            detections = []
            if results and len(results) > 0:
                result = results[0]  # Get first result
                
                # Extract detection data
                if result.boxes is not None:
                    boxes = result.boxes
                    
                    for i in range(len(boxes)):
                        # Get box coordinates (xyxy format)
                        box = boxes.xyxy[i].cpu().numpy()
                        x1, y1, x2, y2 = box
                        
                        # Get confidence score
                        confidence = float(boxes.conf[i].cpu().numpy())
                        
                        # Get class ID and name
                        class_id = int(boxes.cls[i].cpu().numpy())
                        class_name = self.classes[class_id] if class_id < len(self.classes) else f"class_{class_id}"
                        
                        # Convert to center format
                        center_x = (x1 + x2) / 2
                        center_y = (y1 + y2) / 2
                        width = x2 - x1
                        height = y2 - y1
                        
                        detection = {
                            "class_id": class_id,
                            "class_name": class_name,
                            "confidence": confidence,
                            "bounding_box": {
                                "x": float(center_x),
                                "y": float(center_y),
                                "width": float(width),
                                "height": float(height)
                            }
                        }
                        
                        # Only add if confidence meets threshold
                        if confidence >= self.confidence_threshold:
                            detections.append(detection)
                
                # Sort by confidence (highest first)
                detections.sort(key=lambda x: x['confidence'], reverse=True)
                
                # Limit to max_detections
                detections = detections[:self.max_detections]
            
            logger.info("YOLO inference completed. Found %%d detections", len(detections))
            return detections
            
        except Exception as e:
            logger.error("YOLO inference failed: %%s", str(e))
            raise e
    
    def load_model(self):
        """Load the YOLO model"""
        try:
            logger.info("Loading YOLO model from: %%s", self.model_path)
            
            # Import Ultralytics YOLO
            try:
                from ultralytics import YOLO
            except ImportError:
                raise Exception("Ultralytics YOLO not installed. Please install with: pip install ultralytics")
            
            # Load YOLO model
            self.model = YOLO(self.model_path)
            
            # Get class names from the model
            if hasattr(self.model, 'names') and self.model.names:
                self.classes = list(self.model.names.values())
                logger.info("Loaded %%d classes from model", len(self.classes))
            else:
                # Fallback to COCO classes if model doesn't have names
                self.classes = [
                    'person', 'bicycle', 'car', 'motorcycle', 'airplane', 'bus', 'train', 'truck',
                    'boat', 'traffic light', 'fire hydrant', 'stop sign', 'parking meter', 'bench',
                    'bird', 'cat', 'dog', 'horse', 'sheep', 'cow', 'elephant', 'bear', 'zebra',
                    'giraffe', 'backpack', 'umbrella', 'handbag', 'tie', 'suitcase', 'frisbee',
                    'skis', 'snowboard', 'sports ball', 'kite', 'baseball bat', 'baseball glove',
                    'skateboard', 'surfboard', 'tennis racket', 'bottle', 'wine glass', 'cup',
                    'fork', 'knife', 'spoon', 'bowl', 'banana', 'apple', 'sandwich', 'orange',
                    'broccoli', 'carrot', 'hot dog', 'pizza', 'donut', 'cake', 'chair', 'couch',
                    'potted plant', 'bed', 'dining table', 'toilet', 'tv', 'laptop', 'mouse',
                    'remote', 'keyboard', 'cell phone', 'microwave', 'oven', 'toaster', 'sink',
                    'refrigerator', 'book', 'clock', 'vase', 'scissors', 'teddy bear', 'hair drier',
                    'toothbrush'
                ]
                logger.info("Using default COCO classes (%%d classes)", len(self.classes))
            
            # Test the model with a dummy inference to ensure it's working
            try:
                import numpy as np
                dummy_image = np.zeros((640, 640, 3), dtype=np.uint8)
                _ = self.model(dummy_image, verbose=False)
                logger.info("YOLO model test inference successful")
            except Exception as test_error:
                logger.warning("Model test inference failed: %%s", str(test_error))
            
            logger.info("YOLO model loaded successfully with %%d classes", len(self.classes))
            
        except Exception as e:
            logger.error("Failed to load YOLO model: %%s", str(e))
            raise e

def signal_handler(signum, frame):
    """Handle shutdown signals"""
    logger.info("Received shutdown signal, stopping YOLO server...")
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
    
    port = int(os.environ.get('YOLO_SERVER_PORT', str(find_free_port())))
    
    # Create server
    server = HTTPServer(('localhost', port), YOLOInferenceHandler)
    
    # Global variables for model paths
    MODEL_PATH = "%s"
    SCRIPT_PATH = "%s"
    
    # Load model in a separate thread
    def load_model_thread():
        # Create a temporary handler to load the model
        class TempHandler(YOLOInferenceHandler):
            def __init__(self):
                self.model = None
                self.model_path = MODEL_PATH
                self.script_path = SCRIPT_PATH
                self.classes = []
                self.model_type = "yolov8"
                self.confidence_threshold = 0.5
                self.iou_threshold = 0.45
                self.image_size = 640
                self.max_detections = 100
        
        handler = TempHandler()
        handler.load_model()
        # Update the server's handler class with the loaded model
        original_handler = server.RequestHandlerClass
        def create_handler(*args, **kwargs):
            h = original_handler(*args, **kwargs)
            h.model = handler.model
            h.model_path = handler.model_path
            h.script_path = handler.script_path
            h.classes = handler.classes
            h.model_type = handler.model_type
            h.confidence_threshold = handler.confidence_threshold
            h.iou_threshold = handler.iou_threshold
            h.image_size = handler.image_size
            h.max_detections = handler.max_detections
            return h
        server.RequestHandlerClass = create_handler
    
    # Start model loading
    model_thread = threading.Thread(target=load_model_thread)
    model_thread.daemon = True
    model_thread.start()
    
    logger.info("Starting YOLO inference server on port %%d", port)
    
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        logger.info("YOLO server stopped by user")
    except Exception as e:
        logger.error("YOLO server error: %%s", str(e))
        sys.exit(1)

if __name__ == "__main__":
    main()
`, task.ID, modelPath, scriptPath, modelPath, scriptPath, modelPath, scriptPath)

	// Write script to file
	if err := os.WriteFile(outputPath, []byte(script), 0755); err != nil {
		return fmt.Errorf("failed to write script file: %w", err)
	}

	return nil
}

// PerformYOLOInference performs YOLO inference on an image
func (e *YOLOExecutor) PerformYOLOInference(ctx context.Context, task *Task, request *YOLOInferenceRequest) (*YOLOInferenceResponse, error) {
	// This would typically make an HTTP request to the YOLO inference server
	// For now, return a placeholder response
	return &YOLOInferenceResponse{
		Detections: []YOLODetection{
			{
				ClassID:    0,
				ClassName:  "person",
				Confidence: 0.95,
				BoundingBox: struct {
					X      float64 `json:"x"`
					Y      float64 `json:"y"`
					Width  float64 `json:"width"`
					Height float64 `json:"height"`
				}{
					X:      100.0,
					Y:      150.0,
					Width:  80.0,
					Height: 200.0,
				},
			},
		},
		ImageInfo: map[string]interface{}{
			"width":  640,
			"height": 480,
		},
		ProcessingTime: 0.1,
		ModelInfo: map[string]interface{}{
			"model_type": "yolov8",
		},
		Timestamp: time.Now(),
	}, nil
}
