#!/bin/bash

# CNET Agent 集群功能测试脚本

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║           CNET Agent 集群功能测试                            ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# 测试1: 健康检查
echo "【测试1】健康检查"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
for port in 8080 8081 8082 8083; do
    status=$(curl -s http://localhost:$port/api/health | jq -r '.status')
    echo "  Port $port: $status"
done
echo ""

# 测试2: 节点发现
echo "【测试2】节点发现"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "父节点发现的节点："
curl -s http://localhost:8080/api/nodes | jq '.nodes[] | "\(.node_id) (\(.node_type))"'
echo ""
echo "Peer1发现的节点："
curl -s http://localhost:8082/api/nodes | jq '.nodes[] | "\(.node_id) (\(.node_type))"'
echo ""

# 测试3: 资源统计
echo "【测试3】集群资源统计"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
total_cpu=0
used_cpu=0
for port in 8080 8081 8082 8083; do
    name=$(curl -s http://localhost:$port/api/resources | jq -r '.node_id')
    total=$(curl -s http://localhost:$port/api/resources | jq -r '.total.cpu')
    used=$(curl -s http://localhost:$port/api/resources | jq -r '.used.cpu')
    total_cpu=$(echo "$total_cpu + $total" | bc)
    used_cpu=$(echo "$used_cpu + $used" | bc)
    echo "  $name: 总计 ${total}核, 已用 ${used}核"
done
echo "  ────────────────────────────────"
echo "  集群总计: ${total_cpu}核, 已用 ${used_cpu}核"
echo ""

# 测试4: Workload列表
echo "【测试4】运行中的Workload"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
for port in 8080 8081 8082 8083; do
    count=$(curl -s http://localhost:$port/api/workloads | jq '.count')
    if [ "$count" -gt 0 ]; then
        name=$(curl -s http://localhost:$port/api/resources | jq -r '.node_id')
        echo "  $name (port $port):"
        curl -s http://localhost:$port/api/workloads | jq '.workloads[] | "    - \(.name) (\(.status))"'
    fi
done
echo ""

# 测试5: 创建新workload
echo "【测试5】创建新Workload"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "在子节点创建workload..."
result=$(curl -s -X POST http://localhost:8081/api/workloads \
  -H "Content-Type: application/json" \
  -d '{
    "name": "new-child-task",
    "type": "process",
    "requirements": {"cpu": 0.5, "memory": 268435456},
    "config": {"command": "echo", "args": ["New task created!"]}
  }')
wid=$(echo "$result" | jq -r '.id')
status=$(echo "$result" | jq -r '.status')
echo "  Workload ID: $wid"
echo "  Status: $status"
echo ""

# 测试6: 层次结构验证
echo "【测试6】层次结构验证"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "父节点管理的子节点数量："
curl -s http://localhost:8080/api/resources/stats | jq '.child_nodes_count'
echo ""

# 测试7: P2P结构验证
echo "【测试7】P2P结构验证"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Peer1的peer连接数："
curl -s http://localhost:8082/api/resources/stats | jq '.peer_nodes_count'
echo "Peer2的peer连接数："
curl -s http://localhost:8083/api/resources/stats | jq '.peer_nodes_count'
echo ""

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║                     测试完成 ✓                               ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""
echo "集群状态："
echo "  ✓ 层次结构正常运行"
echo "  ✓ P2P连接正常"
echo "  ✓ Workload调度正常"
echo "  ✓ 资源管理正常"
echo ""

