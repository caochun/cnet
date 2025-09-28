#!/bin/bash

# YOLO Demo Script for CNET Agent
# This script demonstrates how to use the YOLO inference engine

set -e

# Configuration
AGENT_HOST="localhost"
AGENT_PORT="8080"
BASE_URL="http://${AGENT_HOST}:${AGENT_PORT}"

echo "=== CNET YOLO Demo ==="
echo "Agent URL: ${BASE_URL}"
echo

# Function to make HTTP requests
make_request() {
    local method=$1
    local url=$2
    local data=$3
    
    if [ -n "$data" ]; then
        curl -s -X "$method" \
            -H "Content-Type: application/json" \
            -d "$data" \
            "$url"
    else
        curl -s -X "$method" "$url"
    fi
}

# Function to check if agent is running
check_agent() {
    echo "Checking if agent is running..."
    if make_request "GET" "${BASE_URL}/api/health" | grep -q "healthy"; then
        echo "✓ Agent is running"
        return 0
    else
        echo "✗ Agent is not running. Please start the agent first."
        return 1
    fi
}

# Function to create a YOLO task
create_yolo_task() {
    echo "Creating YOLO task..."
    
    local task_data='{
        "name": "yolo-detection-task",
        "model_path": "yolo11n.pt",
        "script_path": "/Users/chun/Develop/cnet/examples/yolo_inference_ultralytics.py",
        "config": {
            "model_type": "yolov8",
            "confidence": 0.5,
            "iou_threshold": 0.45,
            "image_size": 640,
            "max_detections": 100,
            "device": "cpu",
            "classes": ["person", "car", "bicycle", "dog", "cat"]
        },
        "resources": {
            "cpu_limit": 2.0,
            "memory_limit": 4096,
            "disk_limit": 1024
        },
        "env": {
            "CUDA_VISIBLE_DEVICES": "0",
            "PYTHONPATH": "/Users/chun/Develop/cnet/examples"
        }
    }'
    
    local response=$(make_request "POST" "${BASE_URL}/api/yolo/tasks" "$task_data")
    echo "Response: $response"
    
    # Extract task ID from response
    local task_id=$(echo "$response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    echo "Task ID: $task_id"
    echo "$task_id"
}

# Function to list YOLO tasks
list_yolo_tasks() {
    echo "Listing YOLO tasks..."
    make_request "GET" "${BASE_URL}/api/yolo/tasks"
    echo
}

# Function to get task info
get_task_info() {
    local task_id=$1
    echo "Getting task info for: $task_id"
    make_request "GET" "${BASE_URL}/api/yolo/tasks/$task_id"
    echo
}

# Function to get task health
get_task_health() {
    local task_id=$1
    echo "Checking task health for: $task_id"
    make_request "GET" "${BASE_URL}/api/yolo/tasks/$task_id/health"
    echo
}

# Function to get model info
get_model_info() {
    local task_id=$1
    echo "Getting model info for: $task_id"
    make_request "GET" "${BASE_URL}/api/yolo/tasks/$task_id/model"
    echo
}

# Function to perform YOLO prediction
perform_prediction() {
    local task_id=$1
    echo "Performing YOLO prediction for task: $task_id"
    
    local prediction_data='{
        "image_url": "https://ultralytics.com/images/bus.jpg",
        "config": {
            "confidence": 0.6,
            "iou_threshold": 0.4,
            "image_size": 640,
            "device": "cpu"
        },
        "options": {
            "save_results": true,
            "output_format": "json"
        }
    }'
    
    make_request "POST" "${BASE_URL}/api/yolo/tasks/$task_id/predict" "$prediction_data"
    echo
}

# Function to get task logs
get_task_logs() {
    local task_id=$1
    echo "Getting task logs for: $task_id"
    make_request "GET" "${BASE_URL}/api/yolo/tasks/$task_id/logs"
    echo
}

# Function to stop task
stop_task() {
    local task_id=$1
    echo "Stopping task: $task_id"
    make_request "DELETE" "${BASE_URL}/api/yolo/tasks/$task_id"
    echo
}

# Main demo flow
main() {
    echo "Starting YOLO demo..."
    echo
    
    # Check if agent is running
    if ! check_agent; then
        exit 1
    fi
    
    echo
    
    # List existing YOLO tasks
    echo "=== Step 1: List existing YOLO tasks ==="
    list_yolo_tasks
    
    # Create a new YOLO task
    echo "=== Step 2: Create YOLO task ==="
    task_id=$(create_yolo_task)
    
    if [ -z "$task_id" ]; then
        echo "Failed to create YOLO task"
        exit 1
    fi
    
    echo "Created task with ID: $task_id"
    echo
    
    # Wait a moment for task to start
    echo "Waiting for task to start..."
    sleep 2
    
    # Get task info
    echo "=== Step 3: Get task information ==="
    get_task_info "$task_id"
    
    # Check task health
    echo "=== Step 4: Check task health ==="
    get_task_health "$task_id"
    
    # Get model info
    echo "=== Step 5: Get model information ==="
    get_model_info "$task_id"
    
    # Perform prediction
    echo "=== Step 6: Perform YOLO prediction ==="
    perform_prediction "$task_id"
    
    # Get task logs
    echo "=== Step 7: Get task logs ==="
    get_task_logs "$task_id"
    
    # List tasks again
    echo "=== Step 8: List YOLO tasks again ==="
    list_yolo_tasks
    
    # Stop the task
    echo "=== Step 9: Stop YOLO task ==="
    stop_task "$task_id"
    
    echo
    echo "=== YOLO Demo Complete ==="
    echo "The demo has successfully demonstrated:"
    echo "- Creating YOLO tasks"
    echo "- Managing YOLO tasks"
    echo "- Performing YOLO inference"
    echo "- Getting task information and logs"
    echo "- Stopping tasks"
}

# Run the demo
main "$@"
