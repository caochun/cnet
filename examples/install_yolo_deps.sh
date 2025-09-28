#!/bin/bash

# YOLO Dependencies Installation Script
# This script installs the necessary dependencies for YOLO inference using Ultralytics YOLO

set -e

echo "=== Installing YOLO Dependencies ==="
echo "Based on Ultralytics YOLO documentation: https://docs.ultralytics.com/zh/modes/predict/"
echo

# Check if Python is available
if ! command -v python3 &> /dev/null; then
    echo "Error: Python 3 is not installed. Please install Python 3 first."
    exit 1
fi

# Check if pip is available
if ! command -v pip3 &> /dev/null; then
    echo "Error: pip3 is not installed. Please install pip3 first."
    exit 1
fi

echo "Python version:"
python3 --version
echo

echo "Pip version:"
pip3 --version
echo

# Install core dependencies
echo "Installing core YOLO dependencies..."
pip3 install ultralytics>=8.0.0

echo "Installing PyTorch..."
pip3 install torch torchvision torchaudio

echo "Installing computer vision libraries..."
pip3 install opencv-python>=4.5.0
pip3 install Pillow>=8.0.0
pip3 install numpy>=1.19.0

echo "Installing HTTP and networking libraries..."
pip3 install requests>=2.25.0

# Optional: Install ONNX for better performance
echo "Installing optional ONNX dependencies for better performance..."
pip3 install onnx>=1.8.0
pip3 install onnxruntime>=1.8.0

# Install development dependencies
echo "Installing development dependencies..."
pip3 install pytest>=6.0.0
pip3 install pytest-cov>=2.10.0

echo
echo "=== Installation Complete ==="
echo "All YOLO dependencies have been installed successfully!"
echo

# Test installation
echo "Testing YOLO installation..."
python3 -c "
try:
    from ultralytics import YOLO
    print('✓ Ultralytics YOLO imported successfully')
    
    # Test loading a model
    model = YOLO('yolo11n.pt')
    print('✓ YOLO model loaded successfully')
    
    # Test inference on a dummy image
    import numpy as np
    dummy_image = np.zeros((640, 640, 3), dtype=np.uint8)
    results = model(dummy_image, verbose=False)
    print('✓ YOLO inference test successful')
    
    print('All tests passed! YOLO is ready to use.')
    
except ImportError as e:
    print(f'✗ Import error: {e}')
    exit(1)
except Exception as e:
    print(f'✗ Test error: {e}')
    exit(1)
"

echo
echo "=== YOLO Setup Complete ==="
echo "You can now use YOLO inference with CNET Agent!"
echo
echo "To test the installation, run:"
echo "  ./examples/yolo_demo.sh"
echo
echo "To run the YOLO inference server directly:"
echo "  python3 examples/yolo_inference_ultralytics.py"
