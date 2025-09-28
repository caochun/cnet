#!/bin/bash

# CNET Agent Hierarchy ID Demo Script
# This script demonstrates the complete hierarchy ID functionality

set -e

echo "🌐 CNET Agent Hierarchy ID Demo"
echo "==============================="
echo ""

# Check if agent is running
if ! curl -s http://localhost:8080/api/health > /dev/null 2>&1; then
    echo "❌ Agent is not running on port 8080"
    echo "   Please start the agent first: ./start.sh"
    exit 1
fi

echo "✅ Agent is running"
echo ""

echo "🏷️  Hierarchy ID Feature Demo"
echo "=============================="
echo ""
echo "This demo shows how CNET Agent supports hierarchical node identification:"
echo "• Parent nodes can assign hierarchical IDs to child nodes"
echo "• Example: Parent node '34.23.1' assigns '34.23.1.8' to a child"
echo "• Hierarchical IDs can be resolved to get node API addresses"
echo ""

# Step 1: Show current nodes
echo "📋 Step 1: Current nodes in the cluster"
echo "----------------------------------------"
NODES=$(curl -s http://localhost:8080/api/discovery/nodes)
echo "$NODES" | jq '.'
echo ""

# Step 2: Register a new node
echo "🔧 Step 2: Registering a new worker node"
echo "----------------------------------------"
REGISTER_RESPONSE=$(curl -s -X POST http://localhost:8080/api/discovery/register \
  -H "Content-Type: application/json" \
  -d '{
    "node": {
      "id": "worker-node-1",
      "name": "Worker Node 1",
      "address": "192.168.1.100",
      "port": 8081,
      "region": "us-west",
      "datacenter": "dc1"
    }
  }')

echo "Registration successful!"
echo ""

# Step 3: Assign hierarchy ID
echo "🏷️  Step 3: Assigning hierarchical ID to the worker node"
echo "-------------------------------------------------------"
ASSIGN_RESPONSE=$(curl -s -X POST http://localhost:8080/api/discovery/hierarchy/assign \
  -H "Content-Type: application/json" \
  -d '{
    "node_id": "worker-node-1"
  }')

echo "Hierarchy ID assignment response:"
echo "$ASSIGN_RESPONSE" | jq '.'
echo ""

# Extract hierarchy ID
HIERARCHY_ID=$(echo "$ASSIGN_RESPONSE" | jq -r '.hierarchy_id')
echo "✅ Assigned hierarchy ID: $HIERARCHY_ID"
echo ""

# Step 4: Resolve hierarchy ID
echo "🔍 Step 4: Resolving hierarchy ID to get node information"
echo "-------------------------------------------------------"
RESOLVE_RESPONSE=$(curl -s -X POST http://localhost:8080/api/discovery/hierarchy/resolve \
  -H "Content-Type: application/json" \
  -d "{
    \"hierarchy_id\": \"$HIERARCHY_ID\"
  }")

echo "Resolution result:"
echo "$RESOLVE_RESPONSE" | jq '.'
echo ""

# Step 5: Show hierarchy structure
echo "🌳 Step 5: Showing hierarchical node organization"
echo "------------------------------------------------"
HIERARCHY_RESPONSE=$(curl -s http://localhost:8080/api/discovery/hierarchy/nodes)
echo "Hierarchical structure:"
echo "$HIERARCHY_RESPONSE" | jq '.'
echo ""

# Step 6: Register another node and show the pattern
echo "🔧 Step 6: Registering another worker node"
echo "------------------------------------------"
REGISTER_RESPONSE2=$(curl -s -X POST http://localhost:8080/api/discovery/register \
  -H "Content-Type: application/json" \
  -d '{
    "node": {
      "id": "worker-node-2",
      "name": "Worker Node 2",
      "address": "192.168.1.101",
      "port": 8082,
      "region": "us-west",
      "datacenter": "dc1"
    }
  }')

echo "Second worker node registered!"
echo ""

# Assign hierarchy ID to second node
ASSIGN_RESPONSE2=$(curl -s -X POST http://localhost:8080/api/discovery/hierarchy/assign \
  -H "Content-Type: application/json" \
  -d '{
    "node_id": "worker-node-2"
  }')

HIERARCHY_ID2=$(echo "$ASSIGN_RESPONSE2" | jq -r '.hierarchy_id')
echo "✅ Assigned hierarchy ID to second node: $HIERARCHY_ID2"
echo ""

# Step 7: Show final hierarchy
echo "🌳 Step 7: Final hierarchical structure"
echo "---------------------------------------"
FINAL_HIERARCHY=$(curl -s http://localhost:8080/api/discovery/hierarchy/nodes)
echo "Complete hierarchy:"
echo "$FINAL_HIERARCHY" | jq '.'
echo ""

# Step 8: Show all nodes with hierarchy info
echo "📋 Step 8: All nodes with hierarchy information"
echo "---------------------------------------------"
FINAL_NODES=$(curl -s http://localhost:8080/api/discovery/nodes)
echo "All nodes:"
echo "$FINAL_NODES" | jq '.'
echo ""

echo "🎉 Hierarchy ID Demo Completed!"
echo ""
echo "📊 Summary:"
echo "   • Registered 2 worker nodes"
echo "   • Assigned hierarchical IDs: $HIERARCHY_ID and $HIERARCHY_ID2"
echo "   • Demonstrated hierarchy ID resolution"
echo "   • Showed hierarchical node organization"
echo ""
echo "🌐 Web UI Features:"
echo "   • Visit http://localhost:8080"
echo "   • Go to 'Nodes' page to see hierarchy IDs and levels"
echo "   • Use 'Assign Hierarchy ID' button to assign IDs"
echo "   • Use 'Resolve Hierarchy ID' button to resolve IDs"
echo ""
echo "🔗 API Endpoints:"
echo "   • POST /api/discovery/hierarchy/assign - Assign hierarchy ID"
echo "   • POST /api/discovery/hierarchy/resolve - Resolve hierarchy ID"
echo "   • GET /api/discovery/hierarchy/nodes - List nodes by hierarchy"
echo "   • GET /api/discovery/nodes - List all nodes with hierarchy info"
echo ""
echo "💡 Use Cases:"
echo "   • Multi-level cluster management"
echo "   • Geographic node organization"
echo "   • Service discovery with hierarchical addressing"
echo "   • Load balancing across hierarchy levels"
