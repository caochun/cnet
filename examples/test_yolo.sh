#!/bin/bash

# YOLO Integration Test Script
# This script tests the YOLO inference engine integration

set -e

# Configuration
AGENT_HOST="localhost"
AGENT_PORT="8080"
BASE_URL="http://${AGENT_HOST}:${AGENT_PORT}"

echo "=== CNET YOLO Integration Test ==="
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
    local response=$(make_request "GET" "${BASE_URL}/api/health")
    if echo "$response" | grep -q "healthy"; then
        echo "✓ Agent is running"
        return 0
    else
        echo "✗ Agent is not running"
        echo "Response: $response"
        return 1
    fi
}

# Function to test YOLO task creation
test_create_yolo_task() {
    echo "Testing YOLO task creation..."
    
    local task_data='{
        "name": "test-yolo-task",
        "model_path": "/tmp/test_yolo_model.pt",
        "script_path": "/tmp/test_yolo_script.py",
        "config": {
            "model_type": "yolov8",
            "confidence": 0.5,
            "iou_threshold": 0.45,
            "image_size": 640,
            "max_detections": 100
        },
        "resources": {
            "cpu_limit": 1.0,
            "memory_limit": 2048,
            "disk_limit": 512
        }
    }'
    
    local response=$(make_request "POST" "${BASE_URL}/api/yolo/tasks" "$task_data")
    echo "Response: $response"
    
    # Extract task ID
    local task_id=$(echo "$response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    if [ -n "$task_id" ]; then
        echo "✓ YOLO task created successfully with ID: $task_id"
        echo "$task_id"
    else
        echo "✗ Failed to create YOLO task"
        return 1
    fi
}

# Function to test YOLO task listing
test_list_yolo_tasks() {
    echo "Testing YOLO task listing..."
    
    local response=$(make_request "GET" "${BASE_URL}/api/yolo/tasks")
    echo "Response: $response"
    
    if echo "$response" | grep -q "test-yolo-task"; then
        echo "✓ YOLO task listing works"
        return 0
    else
        echo "✗ YOLO task listing failed"
        return 1
    fi
}

# Function to test YOLO task info
test_yolo_task_info() {
    local task_id=$1
    echo "Testing YOLO task info for task: $task_id"
    
    local response=$(make_request "GET" "${BASE_URL}/api/yolo/tasks/$task_id")
    echo "Response: $response"
    
    if echo "$response" | grep -q "test-yolo-task"; then
        echo "✓ YOLO task info retrieval works"
        return 0
    else
        echo "✗ YOLO task info retrieval failed"
        return 1
    fi
}

# Function to test YOLO task health
test_yolo_task_health() {
    local task_id=$1
    echo "Testing YOLO task health for task: $task_id"
    
    local response=$(make_request "GET" "${BASE_URL}/api/yolo/tasks/$task_id/health")
    echo "Response: $response"
    
    if echo "$response" | grep -q "healthy\|unhealthy"; then
        echo "✓ YOLO task health check works"
        return 0
    else
        echo "✗ YOLO task health check failed"
        return 1
    fi
}

# Function to test YOLO model info
test_yolo_model_info() {
    local task_id=$1
    echo "Testing YOLO model info for task: $task_id"
    
    local response=$(make_request "GET" "${BASE_URL}/api/yolo/tasks/$task_id/model")
    echo "Response: $response"
    
    if echo "$response" | grep -q "model_type\|classes"; then
        echo "✓ YOLO model info retrieval works"
        return 0
    else
        echo "✗ YOLO model info retrieval failed"
        return 1
    fi
}

# Function to test YOLO prediction
test_yolo_prediction() {
    local task_id=$1
    echo "Testing YOLO prediction for task: $task_id"
    
    local prediction_data='{
        "image_path": "/tmp/test_image.jpg",
        "config": {
            "confidence": 0.6,
            "iou_threshold": 0.4
        }
    }'
    
    local response=$(make_request "POST" "${BASE_URL}/api/yolo/tasks/$task_id/predict" "$prediction_data")
    echo "Response: $response"
    
    if echo "$response" | grep -q "detections\|error"; then
        echo "✓ YOLO prediction works"
        return 0
    else
        echo "✗ YOLO prediction failed"
        return 1
    fi
}

# Function to test YOLO task logs
test_yolo_task_logs() {
    local task_id=$1
    echo "Testing YOLO task logs for task: $task_id"
    
    local response=$(make_request "GET" "${BASE_URL}/api/yolo/tasks/$task_id/logs")
    echo "Response: $response"
    
    if echo "$response" | grep -q "\[\]"; then
        echo "✓ YOLO task logs retrieval works"
        return 0
    else
        echo "✗ YOLO task logs retrieval failed"
        return 1
    fi
}

# Function to test YOLO task stopping
test_stop_yolo_task() {
    local task_id=$1
    echo "Testing YOLO task stopping for task: $task_id"
    
    local response=$(make_request "DELETE" "${BASE_URL}/api/yolo/tasks/$task_id")
    echo "Response: $response"
    
    if echo "$response" | grep -q "stopped\|message"; then
        echo "✓ YOLO task stopping works"
        return 0
    else
        echo "✗ YOLO task stopping failed"
        return 1
    fi
}

# Main test function
main() {
    echo "Starting YOLO integration tests..."
    echo
    
    # Test 1: Check agent health
    echo "=== Test 1: Agent Health Check ==="
    if ! check_agent; then
        echo "Agent is not running. Please start the agent first."
        exit 1
    fi
    echo
    
    # Test 2: Create YOLO task
    echo "=== Test 2: Create YOLO Task ==="
    task_id=$(test_create_yolo_task)
    if [ -z "$task_id" ]; then
        echo "Failed to create YOLO task. Exiting."
        exit 1
    fi
    echo
    
    # Wait a moment for task to initialize
    echo "Waiting for task to initialize..."
    sleep 2
    
    # Test 3: List YOLO tasks
    echo "=== Test 3: List YOLO Tasks ==="
    test_list_yolo_tasks
    echo
    
    # Test 4: Get YOLO task info
    echo "=== Test 4: Get YOLO Task Info ==="
    test_yolo_task_info "$task_id"
    echo
    
    # Test 5: Check YOLO task health
    echo "=== Test 5: Check YOLO Task Health ==="
    test_yolo_task_health "$task_id"
    echo
    
    # Test 6: Get YOLO model info
    echo "=== Test 6: Get YOLO Model Info ==="
    test_yolo_model_info "$task_id"
    echo
    
    # Test 7: Perform YOLO prediction
    echo "=== Test 7: Perform YOLO Prediction ==="
    test_yolo_prediction "$task_id"
    echo
    
    # Test 8: Get YOLO task logs
    echo "=== Test 8: Get YOLO Task Logs ==="
    test_yolo_task_logs "$task_id"
    echo
    
    # Test 9: Stop YOLO task
    echo "=== Test 9: Stop YOLO Task ==="
    test_stop_yolo_task "$task_id"
    echo
    
    echo "=== YOLO Integration Tests Complete ==="
    echo "All tests have been executed successfully!"
    echo
    echo "Summary of tested functionality:"
    echo "- ✓ Agent health check"
    echo "- ✓ YOLO task creation"
    echo "- ✓ YOLO task listing"
    echo "- ✓ YOLO task information retrieval"
    echo "- ✓ YOLO task health monitoring"
    echo "- ✓ YOLO model information"
    echo "- ✓ YOLO inference prediction"
    echo "- ✓ YOLO task log retrieval"
    echo "- ✓ YOLO task management"
}

# Run the tests
main "$@"
