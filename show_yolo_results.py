#!/usr/bin/env python3
"""
YOLO推理结果展示脚本
"""

import requests
import json
import time
from datetime import datetime

def show_yolo_results():
    """展示YOLO推理结果"""
    base_url = "http://localhost:8080"
    
    print("🔍 YOLO推理结果展示")
    print("=" * 60)
    
    # 1. 检查代理状态
    print("1. 检查CNET代理状态...")
    try:
        response = requests.get(f"{base_url}/api/health")
        if response.status_code == 200:
            health_data = response.json()
            print(f"   ✓ 代理状态: {health_data['status']}")
            print(f"   ✓ 时间戳: {health_data['timestamp']}")
        else:
            print("   ✗ 代理未运行")
            return
    except Exception as e:
        print(f"   ✗ 无法连接到代理: {e}")
        return
    
    # 2. 创建YOLO任务
    print("\n2. 创建YOLO推理任务...")
    task_data = {
        "name": "demo-yolo-inference",
        "model_path": "yolo11n.pt",
        "script_path": "/Users/chun/Develop/cnet/examples/yolo_inference_ultralytics.py",
        "config": {
            "model_type": "yolov8",
            "confidence": 0.5,
            "iou_threshold": 0.45,
            "image_size": 640,
            "max_detections": 100,
            "device": "cpu"
        },
        "resources": {
            "cpu_limit": 2.0,
            "memory_limit": 4096,
            "disk_limit": 1024
        }
    }
    
    try:
        response = requests.post(f"{base_url}/api/yolo/tasks", json=task_data)
        if response.status_code == 200:
            task_info = response.json()
            task_id = task_info['id']
            print(f"   ✓ 任务创建成功")
            print(f"   ✓ 任务ID: {task_id}")
            print(f"   ✓ 任务状态: {task_info['status']}")
        else:
            print(f"   ✗ 任务创建失败: {response.status_code}")
            return
    except Exception as e:
        print(f"   ✗ 任务创建失败: {e}")
        return
    
    # 3. 等待任务启动
    print("\n3. 等待YOLO模型加载...")
    time.sleep(5)
    
    # 4. 执行推理
    print("\n4. 执行YOLO推理...")
    inference_data = {
        "image_url": "https://ultralytics.com/images/bus.jpg",
        "config": {
            "confidence": 0.5,
            "iou_threshold": 0.45,
            "image_size": 640,
            "device": "cpu"
        }
    }
    
    try:
        response = requests.post(f"{base_url}/api/yolo/tasks/{task_id}/predict", json=inference_data)
        if response.status_code == 200:
            results = response.json()
            print("   ✓ 推理执行成功")
            
            # 展示推理结果
            print("\n📊 推理结果详情:")
            print("-" * 40)
            
            # 图像信息
            image_info = results.get('image_info', {})
            print(f"🖼️  图像信息:")
            print(f"   - 宽度: {image_info.get('width', 'N/A')} 像素")
            print(f"   - 高度: {image_info.get('height', 'N/A')} 像素")
            
            # 处理时间
            processing_time = results.get('processing_time', 0)
            print(f"⏱️  处理时间: {processing_time:.3f} 秒")
            
            # 模型信息
            model_info = results.get('model_info', {})
            print(f"🤖 模型信息:")
            print(f"   - 模型类型: {model_info.get('model_type', 'N/A')}")
            
            # 检测结果
            detections = results.get('detections', [])
            print(f"\n🎯 检测结果 (共 {len(detections)} 个目标):")
            print("-" * 40)
            
            if detections:
                for i, detection in enumerate(detections, 1):
                    class_name = detection.get('class_name', 'unknown')
                    confidence = detection.get('confidence', 0)
                    class_id = detection.get('class_id', -1)
                    
                    bbox = detection.get('bounding_box', {})
                    x = bbox.get('x', 0)
                    y = bbox.get('y', 0)
                    width = bbox.get('width', 0)
                    height = bbox.get('height', 0)
                    
                    print(f"   目标 {i}:")
                    print(f"     🏷️  类别: {class_name} (ID: {class_id})")
                    print(f"     📊 置信度: {confidence:.3f} ({confidence*100:.1f}%)")
                    print(f"     📦 边界框: 中心({x:.0f}, {y:.0f}) 尺寸({width:.0f}×{height:.0f})")
                    print(f"     📍 位置: 左上角({x-width/2:.0f}, {y-height/2:.0f}) 右下角({x+width/2:.0f}, {y+height/2:.0f})")
                    print()
            else:
                print("   ❌ 未检测到任何目标")
            
            # 时间戳
            timestamp = results.get('timestamp', '')
            if timestamp:
                dt = datetime.fromisoformat(timestamp.replace('Z', '+00:00'))
                print(f"🕐 推理时间: {dt.strftime('%Y-%m-%d %H:%M:%S')}")
            
            print("\n" + "=" * 60)
            print("🎉 YOLO推理结果展示完成！")
            
        else:
            print(f"   ✗ 推理执行失败: {response.status_code}")
            print(f"   错误信息: {response.text}")
    except Exception as e:
        print(f"   ✗ 推理执行失败: {e}")

if __name__ == "__main__":
    show_yolo_results()
