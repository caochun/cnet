#!/bin/bash

# CNET Agent Quick Start Script

echo "🚀 Starting CNET Agent..."

# Stop any existing agent
pkill -f cnet-agent 2>/dev/null || true
sleep 1

# Build if needed
if [ ! -f "bin/cnet-agent" ]; then
    echo "📦 Building agent..."
    make build
fi

# Start agent
echo "🔧 Starting agent..."
./bin/cnet-agent -config config.yaml &

# Wait for startup
echo "⏳ Waiting for agent to start..."
for i in {1..10}; do
    if curl -s http://localhost:8080/api/health > /dev/null 2>&1; then
        echo "✅ Agent started successfully!"
        echo ""
        echo "🌐 Web UI: http://localhost:8080"
        echo "🔗 API: http://localhost:8080/api/health"
        echo ""
        echo "Press Ctrl+C to stop the agent"
        
        # Keep running until interrupted
        trap 'echo ""; echo "🛑 Stopping agent..."; pkill -f cnet-agent; echo "✅ Agent stopped"; exit 0' INT
        while true; do
            sleep 1
        done
    fi
    sleep 1
done

echo "❌ Failed to start agent"
exit 1
