#!/bin/bash

# CNET Agent Web UI Test Script
# This script tests the Web UI functionality

set -e

AGENT_URL="http://localhost:8080"
API_URL="http://localhost:8080/api"

echo "🌐 Testing CNET Agent Web UI..."

# Test Web UI accessibility
echo "📱 Testing Web UI accessibility..."
if curl -s "$AGENT_URL/" | grep -q "CNET Agent Dashboard"; then
    echo "✅ Web UI is accessible"
else
    echo "❌ Web UI is not accessible"
    exit 1
fi

# Test static assets
echo "🎨 Testing static assets..."
if curl -s "$AGENT_URL/static/css/style.css" | grep -q "CNET Agent Web UI Styles"; then
    echo "✅ CSS styles are accessible"
else
    echo "❌ CSS styles are not accessible"
    exit 1
fi

if curl -s "$AGENT_URL/static/js/app.js" | grep -q "CNETApp"; then
    echo "✅ JavaScript is accessible"
else
    echo "❌ JavaScript is not accessible"
    exit 1
fi

# Test API endpoints
echo "🔌 Testing API endpoints..."

# Health check
if curl -s "$API_URL/health" | grep -q "healthy"; then
    echo "✅ Health check API works"
else
    echo "❌ Health check API failed"
    exit 1
fi

# Node info
if curl -s "$API_URL/node" | grep -q "node_id"; then
    echo "✅ Node info API works"
else
    echo "❌ Node info API failed"
    exit 1
fi

# Resources
if curl -s "$API_URL/resources" | grep -q "cpu"; then
    echo "✅ Resources API works"
else
    echo "❌ Resources API failed"
    exit 1
fi

# Tasks
if curl -s "$API_URL/tasks" | grep -q "\[\]"; then
    echo "✅ Tasks API works"
else
    echo "❌ Tasks API failed"
    exit 1
fi

# Test task creation
echo "🔧 Testing task creation..."
TASK_RESPONSE=$(curl -s -X POST "$API_URL/tasks" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "webui-test-task",
    "type": "process",
    "command": "echo",
    "args": ["Web UI Test Successful!"],
    "env": {
      "TEST": "webui"
    }
  }')

if echo "$TASK_RESPONSE" | grep -q "webui-test-task"; then
    echo "✅ Task creation API works"
    TASK_ID=$(echo "$TASK_RESPONSE" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    echo "   Task ID: $TASK_ID"
else
    echo "❌ Task creation API failed"
    exit 1
fi

# Wait for task to complete
echo "⏳ Waiting for task to complete..."
sleep 2

# Test task listing
if curl -s "$API_URL/tasks" | grep -q "webui-test-task"; then
    echo "✅ Task listing API works"
else
    echo "❌ Task listing API failed"
    exit 1
fi

# Test task logs
if curl -s "$API_URL/tasks/$TASK_ID/logs" | grep -q "Web UI Test Successful"; then
    echo "✅ Task logs API works"
else
    echo "❌ Task logs API failed"
    exit 1
fi

echo ""
echo "🎉 All Web UI tests passed!"
echo ""
echo "📱 You can now access the Web UI at: http://localhost:8080"
echo "🔗 API documentation available at: http://localhost:8080/api/health"
echo ""
echo "✨ Features available in the Web UI:"
echo "   • Dashboard with resource monitoring"
echo "   • Task management and creation"
echo "   • Resource usage visualization"
echo "   • Node discovery and management"
echo "   • Real-time status updates"
