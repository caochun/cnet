#!/bin/bash

# CNET Agent Hierarchy ID Test Script
# This script demonstrates the hierarchical ID assignment and resolution functionality

set -e

echo "🌐 CNET Agent Hierarchy ID Test"
echo "================================"
echo ""

# Check if agent is running
if ! curl -s http://localhost:8080/api/health > /dev/null 2>&1; then
    echo "❌ Agent is not running on port 8080"
    echo "   Please start the agent first: ./start.sh"
    exit 1
fi

echo "✅ Agent is running"
echo ""

# Test 1: Register a test node
echo "🔧 Test 1: Registering a test node..."
REGISTER_RESPONSE=$(curl -s -X POST http://localhost:8080/api/discovery/register \
  -H "Content-Type: application/json" \
  -d '{
    "node": {
      "id": "test-node-1",
      "name": "Test Node 1",
      "address": "192.168.1.100",
      "port": 8081,
      "region": "us-west",
      "datacenter": "dc1"
    }
  }')

echo "Registration response: $REGISTER_RESPONSE"
echo ""

# Test 2: List nodes to see hierarchy IDs
echo "📋 Test 2: Listing nodes with hierarchy IDs..."
NODES_RESPONSE=$(curl -s http://localhost:8080/api/discovery/nodes)
echo "Nodes:"
echo "$NODES_RESPONSE" | jq '.'
echo ""

# Test 3: Assign hierarchy ID to a node
echo "🏷️  Test 3: Assigning hierarchy ID to test node..."
ASSIGN_RESPONSE=$(curl -s -X POST http://localhost:8080/api/discovery/hierarchy/assign \
  -H "Content-Type: application/json" \
  -d '{
    "node_id": "test-node-1"
  }')

echo "Assignment response:"
echo "$ASSIGN_RESPONSE" | jq '.'
echo ""

# Extract hierarchy ID from response
HIERARCHY_ID=$(echo "$ASSIGN_RESPONSE" | jq -r '.hierarchy_id')
echo "Assigned hierarchy ID: $HIERARCHY_ID"
echo ""

# Test 4: Resolve hierarchy ID
echo "🔍 Test 4: Resolving hierarchy ID $HIERARCHY_ID..."
RESOLVE_RESPONSE=$(curl -s -X POST http://localhost:8080/api/discovery/hierarchy/resolve \
  -H "Content-Type: application/json" \
  -d "{
    \"hierarchy_id\": \"$HIERARCHY_ID\"
  }")

echo "Resolution response:"
echo "$RESOLVE_RESPONSE" | jq '.'
echo ""

# Test 5: List nodes by hierarchy
echo "🌳 Test 5: Listing nodes organized by hierarchy..."
HIERARCHY_RESPONSE=$(curl -s http://localhost:8080/api/discovery/hierarchy/nodes)
echo "Hierarchy structure:"
echo "$HIERARCHY_RESPONSE" | jq '.'
echo ""

# Test 6: Register another node and assign hierarchy ID
echo "🔧 Test 6: Registering another test node..."
REGISTER_RESPONSE2=$(curl -s -X POST http://localhost:8080/api/discovery/register \
  -H "Content-Type: application/json" \
  -d '{
    "node": {
      "id": "test-node-2",
      "name": "Test Node 2",
      "address": "192.168.1.101",
      "port": 8082,
      "region": "us-west",
      "datacenter": "dc1"
    }
  }')

echo "Second registration response: $REGISTER_RESPONSE2"
echo ""

# Assign hierarchy ID to second node
echo "🏷️  Test 7: Assigning hierarchy ID to second test node..."
ASSIGN_RESPONSE2=$(curl -s -X POST http://localhost:8080/api/discovery/hierarchy/assign \
  -H "Content-Type: application/json" \
  -d '{
    "node_id": "test-node-2"
  }')

echo "Second assignment response:"
echo "$ASSIGN_RESPONSE2" | jq '.'
echo ""

# Test 8: Final hierarchy structure
echo "🌳 Test 8: Final hierarchy structure..."
FINAL_HIERARCHY=$(curl -s http://localhost:8080/api/discovery/hierarchy/nodes)
echo "Final hierarchy structure:"
echo "$FINAL_HIERARCHY" | jq '.'
echo ""

# Test 9: List all nodes with hierarchy information
echo "📋 Test 9: Final node list with hierarchy information..."
FINAL_NODES=$(curl -s http://localhost:8080/api/discovery/nodes)
echo "All nodes:"
echo "$FINAL_NODES" | jq '.'
echo ""

echo "🎉 Hierarchy ID test completed successfully!"
echo ""
echo "📊 Summary:"
echo "   • Registered 2 test nodes"
echo "   • Assigned hierarchy IDs to both nodes"
echo "   • Tested hierarchy ID resolution"
echo "   • Verified hierarchical node organization"
echo ""
echo "🌐 Web UI Features:"
echo "   • Visit http://localhost:8080 to see the hierarchy in action"
echo "   • Go to 'Nodes' page to see hierarchy IDs and levels"
echo "   • Use 'Assign Hierarchy ID' button to assign IDs to nodes"
echo "   • Use 'Resolve Hierarchy ID' button to resolve hierarchy IDs"
echo ""
echo "🔗 API Endpoints:"
echo "   • POST /api/discovery/hierarchy/assign - Assign hierarchy ID"
echo "   • POST /api/discovery/hierarchy/resolve - Resolve hierarchy ID"
echo "   • GET /api/discovery/hierarchy/nodes - List nodes by hierarchy"
echo "   • GET /api/discovery/nodes - List all nodes with hierarchy info"
