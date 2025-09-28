#!/usr/bin/env python3
"""
简单的YOLO推理测试脚本
"""

import sys
import os
import json
import time
import requests
import numpy as np
from ultralytics import YOLO

def test_yolo_basic():
    """测试基本的YOLO推理功能"""
    print("=== 测试基本YOLO推理功能 ===")
    
    try:
        # 加载模型
        print("1. 加载YOLO模型...")
        model = YOLO('yolo11n.pt')
        print("✓ 模型加载成功")
        
        # 测试推理
        print("2. 测试推理...")
        dummy_image = np.zeros((640, 640, 3), dtype=np.uint8)
        results = model(dummy_image, verbose=False)
        print(f"✓ 推理成功，检测到 {len(results[0].boxes) if results[0].boxes is not None else 0} 个目标")
        
        # 测试真实图像推理
        print("3. 测试真实图像推理...")
        # 创建一个简单的测试图像（白色背景上的黑色矩形）
        test_image = np.ones((640, 640, 3), dtype=np.uint8) * 255
        test_image[200:400, 200:400] = [0, 0, 0]  # 黑色矩形
        
        results = model(test_image, verbose=False)
        print(f"✓ 真实图像推理成功，检测到 {len(results[0].boxes) if results[0].boxes is not None else 0} 个目标")
        
        # 显示检测结果
        if results[0].boxes is not None and len(results[0].boxes) > 0:
            print("检测结果:")
            for i, box in enumerate(results[0].boxes):
                conf = float(box.conf[0].cpu().numpy())
                cls = int(box.cls[0].cpu().numpy())
                print(f"  目标 {i+1}: 类别={cls}, 置信度={conf:.3f}")
        
        return True
        
    except Exception as e:
        print(f"✗ 测试失败: {e}")
        return False

def test_yolo_server():
    """测试YOLO推理服务器"""
    print("\n=== 测试YOLO推理服务器 ===")
    
    # 启动服务器
    print("1. 启动YOLO推理服务器...")
    import subprocess
    import threading
    
    # 设置环境变量
    env = os.environ.copy()
    env['MODEL_PATH'] = 'yolo11n.pt'
    
    # 启动服务器进程
    server_process = subprocess.Popen(
        [sys.executable, 'examples/yolo_inference_ultralytics.py'],
        env=env,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE
    )
    
    # 等待服务器启动
    print("等待服务器启动...")
    time.sleep(5)
    
    try:
        # 测试健康检查
        print("2. 测试健康检查...")
        # 找到服务器端口（从服务器输出中解析）
        # 这里我们假设端口是动态分配的，需要从日志中获取
        # 为了简化，我们直接测试一个常见的端口范围
        
        for port in range(9000, 9010):
            try:
                response = requests.get(f'http://localhost:{port}/health', timeout=2)
                if response.status_code == 200:
                    print(f"✓ 服务器在端口 {port} 上运行")
                    server_port = port
                    break
            except:
                continue
        else:
            print("✗ 无法找到运行中的服务器")
            return False
        
        # 测试推理请求
        print("3. 测试推理请求...")
        inference_data = {
            "image_url": "https://ultralytics.com/images/bus.jpg",
            "config": {
                "confidence": 0.5,
                "iou_threshold": 0.45,
                "image_size": 640,
                "device": "cpu"
            }
        }
        
        response = requests.post(
            f'http://localhost:{server_port}/predict',
            json=inference_data,
            timeout=30
        )
        
        if response.status_code == 200:
            result = response.json()
            print("✓ 推理请求成功")
            print(f"检测到 {len(result.get('detections', []))} 个目标")
            print(f"处理时间: {result.get('processing_time', 0):.3f} 秒")
            
            # 显示检测结果
            for i, detection in enumerate(result.get('detections', [])):
                print(f"  目标 {i+1}: {detection.get('class_name', 'unknown')} "
                      f"(置信度: {detection.get('confidence', 0):.3f})")
        else:
            print(f"✗ 推理请求失败: {response.status_code}")
            print(response.text)
            return False
        
        return True
        
    finally:
        # 清理服务器进程
        print("4. 清理服务器进程...")
        server_process.terminate()
        server_process.wait()
        print("✓ 服务器已停止")

def main():
    """主测试函数"""
    print("开始YOLO推理功能测试...")
    print("=" * 50)
    
    # 测试基本功能
    basic_success = test_yolo_basic()
    
    # 测试服务器功能
    server_success = test_yolo_server()
    
    # 总结
    print("\n" + "=" * 50)
    print("测试结果总结:")
    print(f"基本YOLO推理: {'✓ 通过' if basic_success else '✗ 失败'}")
    print(f"YOLO推理服务器: {'✓ 通过' if server_success else '✗ 失败'}")
    
    if basic_success and server_success:
        print("\n🎉 所有测试通过！YOLO推理功能工作正常。")
        return 0
    else:
        print("\n❌ 部分测试失败，请检查错误信息。")
        return 1

if __name__ == "__main__":
    sys.exit(main())
