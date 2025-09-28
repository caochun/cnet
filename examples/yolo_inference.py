#!/usr/bin/env python3
"""
YOLO Inference Script for CNET Agent
This script demonstrates how to implement a YOLO inference server
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
        self.model_path = ""
        self.script_path = ""
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
                logger.error("YOLO inference error: %s", str(e))
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
            logger.error("Failed to load image: %s", str(e))
            return None
    
    def run_yolo_inference(self, image, request_data):
        """Run YOLO inference on the image"""
        try:
            # Update parameters from request
            config = request_data.get('config', {})
            if 'confidence' in config:
                self.confidence_threshold = config['confidence']
            if 'iou_threshold' in config:
                self.iou_threshold = config['iou_threshold']
            if 'image_size' in config:
                self.image_size = config['image_size']
            
            # TODO: Implement actual YOLO inference
            # This is a placeholder implementation
            # In a real implementation, you would:
            # 1. Preprocess the image (resize, normalize, etc.)
            # 2. Run the YOLO model inference
            # 3. Post-process the results (NMS, confidence filtering, etc.)
            # 4. Convert to the desired output format
            
            # Placeholder detections
            detections = [
                {
                    "class_id": 0,
                    "class_name": "person",
                    "confidence": 0.95,
                    "bounding_box": {
                        "x": 100.0,
                        "y": 150.0,
                        "width": 80.0,
                        "height": 200.0
                    }
                },
                {
                    "class_id": 1,
                    "class_name": "car",
                    "confidence": 0.87,
                    "bounding_box": {
                        "x": 300.0,
                        "y": 200.0,
                        "width": 120.0,
                        "height": 80.0
                    }
                }
            ]
            
            # Filter by confidence threshold
            filtered_detections = []
            for det in detections:
                if det['confidence'] >= self.confidence_threshold:
                    filtered_detections.append(det)
            
            return filtered_detections[:self.max_detections]
            
        except Exception as e:
            logger.error("YOLO inference failed: %s", str(e))
            raise e
    
    def load_model(self):
        """Load the YOLO model"""
        try:
            logger.info("Loading YOLO model from: %s", self.model_path)
            
            # TODO: Implement actual YOLO model loading
            # This would load the YOLO model from self.model_path
            # For different YOLO versions (YOLOv5, YOLOv8, etc.)
            
            # Placeholder model loading
            self.model = "yolo_model_loaded"
            
            # Load class names (COCO classes as default)
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
            
            logger.info("YOLO model loaded successfully with %d classes", len(self.classes))
            
        except Exception as e:
            logger.error("Failed to load YOLO model: %s", str(e))
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
    MODEL_PATH = os.environ.get('MODEL_PATH', '/path/to/yolo/model.pt')
    SCRIPT_PATH = os.environ.get('SCRIPT_PATH', '/path/to/yolo/inference.py')
    
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
    
    logger.info("Starting YOLO inference server on port %d", port)
    
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        logger.info("YOLO server stopped by user")
    except Exception as e:
        logger.error("YOLO server error: %s", str(e))
        sys.exit(1)

if __name__ == "__main__":
    main()
