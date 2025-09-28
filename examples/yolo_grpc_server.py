#!/usr/bin/env python3
"""
YOLO gRPC Server for CNET Agent
This is a more robust approach than HTTP servers
"""

import grpc
from concurrent import futures
import time
import logging
from ultralytics import YOLO
import cv2
import numpy as np
from PIL import Image, ImageDraw, ImageFont
import io
import uuid
import os

# 导入生成的protobuf文件
import yolo_pb2
import yolo_pb2_grpc

logger = logging.getLogger(__name__)

class YOLOInferenceService(yolo_pb2_grpc.YOLOInferenceServicer):
    def __init__(self):
        self.model = None
        self.model_path = None
        self.classes = []
        
    def LoadModel(self, request, context):
        """Load YOLO model"""
        try:
            self.model_path = request.model_path
            self.model = YOLO(self.model_path)
            
            # Get class names
            if hasattr(self.model, 'names') and self.model.names:
                self.classes = list(self.model.names.values())
            else:
                self.classes = ['person', 'bicycle', 'car', 'motorcycle', 'airplane']
            
            return yolo_pb2.LoadModelResponse(
                success=True,
                message=f"Model loaded successfully with {len(self.classes)} classes"
            )
        except Exception as e:
            return yolo_pb2.LoadModelResponse(
                success=False,
                message=f"Failed to load model: {str(e)}"
            )
    
    def Predict(self, request, context):
        """Perform YOLO inference"""
        try:
            if self.model is None:
                return yolo_pb2.PredictResponse(
                    success=False,
                    message="Model not loaded"
                )
            
            # Convert image data
            if request.image_data:
                image = self._decode_image(request.image_data)
            else:
                return yolo_pb2.PredictResponse(
                    success=False,
                    message="No image data provided"
                )
            
            # Run inference
            results = self.model(image, conf=request.confidence, iou=request.iou_threshold)
            
            # Process results
            detections = []
            if results and len(results) > 0:
                result = results[0]
                if result.boxes is not None:
                    for i in range(len(result.boxes)):
                        box = result.boxes.xyxy[i].cpu().numpy()
                        confidence = float(result.boxes.conf[i].cpu().numpy())
                        class_id = int(result.boxes.cls[i].cpu().numpy())
                        
                        detection = yolo_pb2.Detection(
                            class_id=class_id,
                            class_name=self.classes[class_id] if class_id < len(self.classes) else f"class_{class_id}",
                            confidence=confidence,
                            x1=float(box[0]),
                            y1=float(box[1]),
                            x2=float(box[2]),
                            y2=float(box[3])
                        )
                        detections.append(detection)
            
            # Draw and save result image
            result_image_path = self._draw_and_save_result(image, detections)
            
            return yolo_pb2.PredictResponse(
                success=True,
                detections=detections,
                result_image_path=result_image_path,
                processing_time=0.1
            )
            
        except Exception as e:
            return yolo_pb2.PredictResponse(
                success=False,
                message=f"Inference failed: {str(e)}"
            )
    
    def _decode_image(self, image_data):
        """Decode base64 image data"""
        import base64
        image_bytes = base64.b64decode(image_data)
        image = Image.open(io.BytesIO(image_bytes))
        return cv2.cvtColor(np.array(image), cv2.COLOR_RGB2BGR)
    
    def _draw_and_save_result(self, image, detections):
        """Draw bounding boxes and save result"""
        try:
            output_dir = "/tmp/cnet/yolo_results"
            os.makedirs(output_dir, exist_ok=True)
            
            timestamp = int(time.time())
            unique_id = str(uuid.uuid4())[:8]
            filename = f"yolo_result_{timestamp}_{unique_id}.jpg"
            result_path = os.path.join(output_dir, filename)
            
            # Convert to PIL for drawing
            if isinstance(image, np.ndarray):
                image = cv2.cvtColor(image, cv2.COLOR_BGR2RGB)
                pil_image = Image.fromarray(image)
            else:
                pil_image = image.copy()
            
            draw = ImageDraw.Draw(pil_image)
            
            # Draw bounding boxes
            colors = [(255, 0, 0), (0, 255, 0), (0, 0, 255), (255, 255, 0), (255, 0, 255)]
            for i, detection in enumerate(detections):
                color = colors[i % len(colors)]
                draw.rectangle([detection.x1, detection.y1, detection.x2, detection.y2], 
                              outline=color, width=3)
                
                label = f"{detection.class_name}: {detection.confidence:.2f}"
                draw.text((detection.x1, detection.y1 - 20), label, fill=color)
            
            pil_image.save(result_path, "JPEG", quality=95)
            return result_path
            
        except Exception as e:
            logger.error(f"Failed to save result image: {e}")
            return ""

def serve():
    """Start gRPC server"""
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    yolo_pb2_grpc.add_YOLOInferenceServicer_to_server(YOLOInferenceService(), server)
    
    listen_addr = '[::]:50051'
    server.add_insecure_port(listen_addr)
    
    logger.info(f"Starting YOLO gRPC server on {listen_addr}")
    server.start()
    
    try:
        server.wait_for_termination()
    except KeyboardInterrupt:
        server.stop(0)

if __name__ == '__main__':
    logging.basicConfig(level=logging.INFO)
    serve()
