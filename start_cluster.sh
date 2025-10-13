#!/bin/bash

# CNET Agent 集群启动脚本
# 演示层次结构和P2P结构

set -e

echo "=== CNET Agent 集群启动 ==="
echo ""

# 清理旧进程
pkill -f cnet-agent 2>/dev/null || true
sleep 1

# 创建日志目录
mkdir -p logs

echo "1. 启动江苏省节点 (jiangsu, port 8080)..."
./bin/cnet-agent -config configs/config_jiangsu.yaml > logs/jiangsu.log 2>&1 &
JIANGSU_PID=$!
echo "   PID: $JIANGSU_PID"
sleep 2

echo ""
echo "2. 启动南京市节点 (nanjing, port 8081)..."
echo "   将注册到江苏省节点 localhost:8080"
./bin/cnet-agent -config configs/config_nanjing.yaml > logs/nanjing.log 2>&1 &
NANJING_PID=$!
echo "   PID: $NANJING_PID"
sleep 2

echo ""
echo "3. 启动宿迁市节点 (suqian, port 8082)..."
./bin/cnet-agent -config configs/config_suqian.yaml > logs/suqian.log 2>&1 &
SUQIAN_PID=$!
echo "   PID: $SUQIAN_PID"
sleep 2

echo ""
echo "4. 启动常州市节点 (changzhou, port 8083)..."
echo "   将与南京、宿迁互相发现"
./bin/cnet-agent -config configs/config_changzhou.yaml > logs/changzhou.log 2>&1 &
CHANGZHOU_PID=$!
echo "   PID: $CHANGZHOU_PID"
sleep 2

echo ""
echo "=== 集群拓扑结构 ==="
echo ""
echo "层次结构："
echo "  江苏省 (jiangsu):8080"
echo "  └── 南京市 (nanjing):8081"
echo ""
echo "P2P结构（三角形网状）："
echo "         南京:8081"
echo "        ╱        ╲"
echo "  宿迁:8082 ←→ 常州:8083"
echo ""

echo "=== 节点列表 ==="
echo "江苏省:  http://localhost:8080  (PID: $JIANGSU_PID)"
echo "南京市:  http://localhost:8081  (PID: $NANJING_PID)"
echo "宿迁市:  http://localhost:8082  (PID: $SUQIAN_PID)"
echo "常州市:  http://localhost:8083  (PID: $CHANGZHOU_PID)"
echo ""

echo "等待节点互相发现..."
sleep 3

echo ""
echo "=== 验证集群状态 ==="
echo ""

echo "1. 检查江苏省的下级节点："
curl -s http://localhost:8080/api/nodes | jq '.nodes[] | select(.node_type=="child") | {node_id, node_type, status}'

echo ""
echo "2. 检查宿迁市发现的peer节点："
curl -s http://localhost:8082/api/nodes | jq '.nodes[] | select(.node_type=="peer") | {node_id, node_type, status}'

echo ""
echo "3. 检查常州市发现的peer节点："
curl -s http://localhost:8083/api/nodes | jq '.nodes[] | select(.node_type=="peer") | {node_id, node_type, status}'

echo ""
echo "=== 集群启动完成 ==="
echo ""
echo "查看日志："
echo "  tail -f logs/jiangsu.log"
echo "  tail -f logs/nanjing.log"
echo "  tail -f logs/suqian.log"
echo "  tail -f logs/changzhou.log"
echo ""
echo "停止集群："
echo "  ./stop_cluster.sh"
echo "  或"
echo "  pkill -f cnet-agent"
echo ""

