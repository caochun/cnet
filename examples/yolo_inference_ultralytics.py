#!/usr/bin/env python3
"""
YOLO Inference Script for CNET Agent using Ultralytics YOLO
This script demonstrates how to implement a YOLO inference server using Ultralytics YOLO
Based on: https://docs.ultralytics.com/zh/modes/predict/
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
from PIL import Image, ImageDraw, ImageFont
import io
import uuid

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
        """Perform YOLO inference using Ultralytics YOLO"""
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
        inference_result = self.run_yolo_inference(image, request_data)
        
        processing_time = time.time() - start_time
        
        # Extract detections and result image path
        detections = inference_result.get("detections", [])
        result_image_path = inference_result.get("result_image_path")
        
        response = {
            "detections": detections,
            "image_info": image_info,
            "processing_time": processing_time,
            "result_image_path": result_image_path,
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
            
            logger.info("YOLO inference completed. Found %d detections", len(detections))
            
            # Draw bounding boxes on image and save result
            result_image_path = self.draw_boxes_and_save(image, detections, request_data)
            
            return {
                "detections": detections,
                "result_image_path": result_image_path
            }
            
        except Exception as e:
            logger.error("YOLO inference failed: %s", str(e))
            raise e
    
    def draw_boxes_and_save(self, image, detections, request_data):
        """Draw bounding boxes on image and save the result"""
        try:
            # Create output directory if it doesn't exist
            output_dir = "/tmp/cnet/yolo_results"
            os.makedirs(output_dir, exist_ok=True)
            
            # Generate unique filename
            timestamp = int(time.time())
            unique_id = str(uuid.uuid4())[:8]
            filename = f"yolo_result_{timestamp}_{unique_id}.jpg"
            result_path = os.path.join(output_dir, filename)
            
            # Convert image to PIL if it's numpy array
            if isinstance(image, np.ndarray):
                if len(image.shape) == 3 and image.shape[2] == 3:
                    # Convert BGR to RGB if needed
                    image = cv2.cvtColor(image, cv2.COLOR_BGR2RGB)
                pil_image = Image.fromarray(image)
            else:
                pil_image = image.copy()
            
            # Create a copy for drawing
            draw_image = pil_image.copy()
            draw = ImageDraw.Draw(draw_image)
            
            # Try to load a font, fallback to default if not available
            try:
                font = ImageFont.truetype("/System/Library/Fonts/Arial.ttf", 20)
            except:
                try:
                    font = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", 20)
                except:
                    font = ImageFont.load_default()
            
            # Define colors for different classes
            colors = [
                (255, 0, 0),    # Red
                (0, 255, 0),    # Green
                (0, 0, 255),    # Blue
                (255, 255, 0),  # Yellow
                (255, 0, 255),  # Magenta
                (0, 255, 255),  # Cyan
                (255, 128, 0),  # Orange
                (128, 0, 255),  # Purple
                (255, 192, 203), # Pink
                (0, 128, 0),   # Dark Green
            ]
            
            # Draw bounding boxes and labels
            for i, detection in enumerate(detections):
                bbox = detection['bounding_box']
                class_name = detection['class_name']
                confidence = detection['confidence']
                
                # Get box coordinates (convert from center format to corner format)
                center_x = bbox['x']
                center_y = bbox['y']
                width = bbox['width']
                height = bbox['height']
                
                x1 = center_x - width / 2
                y1 = center_y - height / 2
                x2 = center_x + width / 2
                y2 = center_y + height / 2
                
                # Choose color based on class
                color = colors[i % len(colors)]
                
                # Draw bounding box
                draw.rectangle([x1, y1, x2, y2], outline=color, width=3)
                
                # Prepare label text
                label = f"{class_name}: {confidence:.2f}"
                
                # Get text size
                bbox_text = draw.textbbox((0, 0), label, font=font)
                text_width = bbox_text[2] - bbox_text[0]
                text_height = bbox_text[3] - bbox_text[1]
                
                # Draw label background
                label_y = max(y1 - text_height - 5, 0)
                draw.rectangle([x1, label_y, x1 + text_width + 10, y1], fill=color)
                
                # Draw label text
                draw.text((x1 + 5, label_y), label, fill=(255, 255, 255), font=font)
            
            # Add timestamp and detection count to image
            timestamp_text = f"Detection Time: {time.strftime('%Y-%m-%d %H:%M:%S')}"
            count_text = f"Objects Detected: {len(detections)}"
            
            # Get image dimensions
            img_width, img_height = draw_image.size
            
            # Draw info text at bottom of image
            info_y = img_height - 60
            draw.rectangle([0, info_y, img_width, img_height], fill=(0, 0, 0, 128))
            draw.text((10, info_y + 5), timestamp_text, fill=(255, 255, 255), font=font)
            draw.text((10, info_y + 30), count_text, fill=(255, 255, 255), font=font)
            
            # Save the result image
            draw_image.save(result_path, "JPEG", quality=95)
            logger.info("Result image saved to: %s", result_path)
            
            return result_path
            
        except Exception as e:
            logger.error("Failed to draw boxes and save image: %s", str(e))
            return None
    
    def load_model(self):
        """Load the YOLO model using Ultralytics YOLO"""
        try:
            logger.info("Loading YOLO model from: %s", self.model_path)
            
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
                logger.info("Loaded %d classes from model", len(self.classes))
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
                logger.info("Using default COCO classes (%d classes)", len(self.classes))
            
            # Test the model with a dummy inference to ensure it's working
            try:
                import numpy as np
                dummy_image = np.zeros((640, 640, 3), dtype=np.uint8)
                _ = self.model(dummy_image, verbose=False)
                logger.info("YOLO model test inference successful")
            except Exception as test_error:
                logger.warning("Model test inference failed: %s", str(test_error))
            
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
