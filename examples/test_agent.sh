#!/bin/bash

# CNET Agent Test Script
# This script demonstrates how to use the CNET agent

set -e

AGENT_URL="http://localhost:8080"

echo "🚀 Testing CNET Agent..."

# Check if agent is running
echo "📡 Checking agent health..."
curl -s "$AGENT_URL/health" | jq '.'

echo ""
echo "🏠 Getting node information..."
curl -s "$AGENT_URL/node" | jq '.'

echo ""
echo "💻 Getting resource information..."
curl -s "$AGENT_URL/resources" | jq '.cpu, .memory'

echo ""
echo "📊 Getting resource usage..."
curl -s "$AGENT_URL/resources/usage" | jq '.'

echo ""
echo "🔧 Creating a test task..."
TASK_RESPONSE=$(curl -s -X POST "$AGENT_URL/tasks" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "hello-world",
    "type": "process",
    "command": "echo",
    "args": ["Hello from CNET Agent!"],
    "env": {
      "GREETING": "Welcome to CNET"
    }
  }')

echo "$TASK_RESPONSE" | jq '.'

# Extract task ID
TASK_ID=$(echo "$TASK_RESPONSE" | jq -r '.id')
echo "Task ID: $TASK_ID"

echo ""
echo "⏳ Waiting for task to complete..."
sleep 2

echo ""
echo "📋 Listing all tasks..."
curl -s "$AGENT_URL/tasks" | jq '.'

echo ""
echo "📝 Getting task logs..."
curl -s "$AGENT_URL/tasks/$TASK_ID/logs" | jq '.'

echo ""
echo "✅ Test completed successfully!"
