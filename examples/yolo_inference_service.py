#!/usr/bin/env python3
"""
YOLO Inference Service for CNET Agent
Containerized Python service for better isolation and management
"""

import asyncio
import json
import logging
import os
import time
import uuid
from typing import List, Dict, Any
from fastapi import FastAPI, HTTPException, UploadFile, File
from fastapi.responses import FileResponse
from pydantic import BaseModel
import uvicorn
from ultralytics import YOLO
import cv2
import numpy as np
from PIL import Image, ImageDraw, ImageFont
import io
import base64

# 配置日志
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# FastAPI应用
app = FastAPI(title="YOLO Inference Service", version="1.0.0")

# 全局变量
model = None
model_path = None
classes = []

class InferenceRequest(BaseModel):
    image_data: str  # base64 encoded image
    confidence: float = 0.5
    iou_threshold: float = 0.45
    image_size: int = 640
    max_detections: int = 100

class Detection(BaseModel):
    class_id: int
    class_name: str
    confidence: float
    x: float
    y: float
    width: float
    height: float

class InferenceResponse(BaseModel):
    success: bool
    detections: List[Detection]
    result_image_path: str
    processing_time: float
    message: str = ""

@app.on_event("startup")
async def startup_event():
    """启动时加载模型"""
    global model, model_path, classes
    
    model_path = os.getenv("YOLO_MODEL_PATH", "yolo11n.pt")
    logger.info(f"Loading YOLO model from: {model_path}")
    
    try:
        model = YOLO(model_path)
        
        # 获取类别名称
        if hasattr(model, 'names') and model.names:
            classes = list(model.names.values())
        else:
            classes = ['person', 'bicycle', 'car', 'motorcycle', 'airplane']
        
        logger.info(f"Model loaded successfully with {len(classes)} classes")
        
        # 测试推理
        dummy_image = np.zeros((640, 640, 3), dtype=np.uint8)
        _ = model(dummy_image, verbose=False)
        logger.info("Model test inference successful")
        
    except Exception as e:
        logger.error(f"Failed to load model: {e}")
        raise

@app.get("/health")
async def health_check():
    """健康检查"""
    return {
        "status": "healthy",
        "model_loaded": model is not None,
        "model_path": model_path,
        "classes_count": len(classes)
    }

@app.post("/predict", response_model=InferenceResponse)
async def predict(request: InferenceRequest):
    """执行YOLO推理"""
    try:
        if model is None:
            raise HTTPException(status_code=500, detail="Model not loaded")
        
        start_time = time.time()
        
        # 解码图像
        image = decode_base64_image(request.image_data)
        
        # 执行推理
        results = model(
            image,
            conf=request.confidence,
            iou=request.iou_threshold,
            imgsz=request.image_size,
            max_det=request.max_detections
        )
        
        # 处理结果
        detections = []
        if results and len(results) > 0:
            result = results[0]
            if result.boxes is not None:
                for i in range(len(result.boxes)):
                    box = result.boxes.xyxy[i].cpu().numpy()
                    confidence = float(result.boxes.conf[i].cpu().numpy())
                    class_id = int(result.boxes.cls[i].cpu().numpy())
                    
                    # 转换为中心点格式
                    x1, y1, x2, y2 = box
                    center_x = (x1 + x2) / 2
                    center_y = (y1 + y2) / 2
                    width = x2 - x1
                    height = y2 - y1
                    
                    detection = Detection(
                        class_id=class_id,
                        class_name=classes[class_id] if class_id < len(classes) else f"class_{class_id}",
                        confidence=confidence,
                        x=float(center_x),
                        y=float(center_y),
                        width=float(width),
                        height=float(height)
                    )
                    detections.append(detection)
        
        # 绘制并保存结果图像
        result_image_path = draw_and_save_result(image, detections)
        
        processing_time = time.time() - start_time
        
        return InferenceResponse(
            success=True,
            detections=detections,
            result_image_path=result_image_path,
            processing_time=processing_time
        )
        
    except Exception as e:
        logger.error(f"Inference failed: {e}")
        return InferenceResponse(
            success=False,
            detections=[],
            result_image_path="",
            processing_time=0,
            message=str(e)
        )

@app.get("/result/{filename}")
async def get_result_image(filename: str):
    """获取结果图像"""
    result_path = f"/tmp/cnet/yolo_results/{filename}"
    if os.path.exists(result_path):
        return FileResponse(result_path)
    else:
        raise HTTPException(status_code=404, detail="Result image not found")

def decode_base64_image(image_data: str) -> np.ndarray:
    """解码base64图像数据"""
    try:
        image_bytes = base64.b64decode(image_data)
        image = Image.open(io.BytesIO(image_bytes))
        return cv2.cvtColor(np.array(image), cv2.COLOR_RGB2BGR)
    except Exception as e:
        logger.error(f"Failed to decode image: {e}")
        raise

def draw_and_save_result(image: np.ndarray, detections: List[Detection]) -> str:
    """绘制边界框并保存结果"""
    try:
        # 创建输出目录
        output_dir = "/tmp/cnet/yolo_results"
        os.makedirs(output_dir, exist_ok=True)
        
        # 生成文件名
        timestamp = int(time.time())
        unique_id = str(uuid.uuid4())[:8]
        filename = f"yolo_result_{timestamp}_{unique_id}.jpg"
        result_path = os.path.join(output_dir, filename)
        
        # 转换为PIL图像
        if isinstance(image, np.ndarray):
            image = cv2.cvtColor(image, cv2.COLOR_BGR2RGB)
            pil_image = Image.fromarray(image)
        else:
            pil_image = image.copy()
        
        draw = ImageDraw.Draw(pil_image)
        
        # 定义颜色
        colors = [
            (255, 0, 0),    # Red
            (0, 255, 0),    # Green
            (0, 0, 255),    # Blue
            (255, 255, 0),  # Yellow
            (255, 0, 255),  # Magenta
        ]
        
        # 绘制边界框
        for i, detection in enumerate(detections):
            color = colors[i % len(colors)]
            
            # 计算边界框坐标
            x1 = detection.x - detection.width / 2
            y1 = detection.y - detection.height / 2
            x2 = detection.x + detection.width / 2
            y2 = detection.y + detection.height / 2
            
            # 绘制边界框
            draw.rectangle([x1, y1, x2, y2], outline=color, width=3)
            
            # 绘制标签
            label = f"{detection.class_name}: {detection.confidence:.2f}"
            draw.text((x1, y1 - 20), label, fill=color)
        
        # 保存图像
        pil_image.save(result_path, "JPEG", quality=95)
        logger.info(f"Result image saved to: {result_path}")
        
        return result_path
        
    except Exception as e:
        logger.error(f"Failed to save result image: {e}")
        return ""

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8080)
