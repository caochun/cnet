#!/bin/bash

# CNET Agent Demo Script
# This script demonstrates the complete CNET Agent functionality

set -e

echo "🚀 CNET Agent Demo"
echo "=================="
echo ""

# Check if agent is already running
if curl -s http://localhost:8080/api/health > /dev/null 2>&1; then
    echo "⚠️  Agent is already running on port 8080"
    echo "   Stopping existing agent..."
    pkill -f cnet-agent 2>/dev/null || true
    sleep 2
fi

echo "📦 Building CNET Agent..."
make build

echo ""
echo "🔧 Starting CNET Agent..."
./bin/cnet-agent -config config.yaml &
AGENT_PID=$!

# Wait for agent to start
echo "⏳ Waiting for agent to start..."
for i in {1..15}; do
    if curl -s http://localhost:8080/api/health > /dev/null 2>&1; then
        echo "✅ Agent started successfully"
        break
    fi
    if [ $i -eq 15 ]; then
        echo "❌ Failed to start agent after 15 seconds"
        echo "   Check the logs above for errors"
        kill $AGENT_PID 2>/dev/null || true
        exit 1
    fi
    echo "   Attempt $i/15..."
    sleep 1
done

echo ""
echo "🌐 Web UI Demo"
echo "=============="
echo "📱 Open your browser and visit: http://localhost:8080"
echo "   • Dashboard: Real-time resource monitoring"
echo "   • Tasks: Create and manage tasks"
echo "   • Resources: Detailed system information"
echo "   • Nodes: Discovered nodes in the cluster"
echo "   • Logs: Agent and task logs"
echo ""

echo "🔌 API Demo"
echo "==========="

# Test health endpoint
echo "🏥 Testing health endpoint..."
curl -s http://localhost:8080/api/health | jq '.'

# Test node info
echo ""
echo "🏠 Testing node information..."
curl -s http://localhost:8080/api/node | jq '.'

# Test resources
echo ""
echo "💻 Testing resource information..."
curl -s http://localhost:8080/api/resources | jq '.cpu, .memory'

# Test resource usage
echo ""
echo "📊 Testing resource usage..."
curl -s http://localhost:8080/api/resources/usage | jq '.cpu, .memory, .disk'

# Create a test task
echo ""
echo "🔧 Creating a test task..."
TASK_RESPONSE=$(curl -s -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "demo-task",
    "type": "process",
    "command": "echo",
    "args": ["Hello from CNET Agent Demo!"],
    "env": {
      "DEMO": "true",
      "TIMESTAMP": "'$(date)'"
    }
  }')

echo "$TASK_RESPONSE" | jq '.'

# Extract task ID
TASK_ID=$(echo "$TASK_RESPONSE" | jq -r '.id')
echo "Task ID: $TASK_ID"

# Wait for task to complete
echo ""
echo "⏳ Waiting for task to complete..."
sleep 2

# List tasks
echo ""
echo "📋 Listing all tasks..."
curl -s http://localhost:8080/api/tasks | jq '.'

# Get task logs
echo ""
echo "📝 Getting task logs..."
curl -s "http://localhost:8080/api/tasks/$TASK_ID/logs" | jq '.'

# Test discovery
echo ""
echo "🌐 Testing node discovery..."
curl -s http://localhost:8080/api/discovery/nodes | jq '.'

echo ""
echo "🎉 Demo completed successfully!"
echo ""
echo "📱 Web UI Features:"
echo "   • Real-time dashboard with resource monitoring"
echo "   • Task creation and management interface"
echo "   • Resource usage visualization"
echo "   • Node discovery and cluster management"
echo "   • Task logs and execution history"
echo ""
echo "🔗 Access Points:"
echo "   • Web UI: http://localhost:8080"
echo "   • API Health: http://localhost:8080/api/health"
echo "   • API Docs: Check README.md for full API reference"
echo ""
echo "🛑 To stop the agent, run: pkill -f cnet-agent"
echo "   Or press Ctrl+C to stop this demo"

# Keep the agent running
echo ""
echo "🔄 Agent is running... Press Ctrl+C to stop"
trap 'echo ""; echo "🛑 Stopping agent..."; kill $AGENT_PID 2>/dev/null || true; echo "✅ Agent stopped"; exit 0' INT

# Wait for user to stop
while true; do
    sleep 1
done
