#!/bin/bash

# 测试层次化节点注册修复
# 这个脚本测试节点注册后是否能正确响应下级节点的注册请求

echo "=== 测试层次化节点注册修复 ==="

# 清理之前的日志
rm -f *.log

# 启动根节点 (discovery-server)
echo "1. 启动根节点 (discovery-server)..."
./bin/cnet-agent -config config.yaml > discovery-server.log 2>&1 &
DISCOVERY_PID=$!
sleep 3

# 检查根节点是否启动成功
if ! curl -s http://localhost:8080/health > /dev/null; then
    echo "错误: 根节点启动失败"
    kill $DISCOVERY_PID 2>/dev/null
    exit 1
fi
echo "✓ 根节点启动成功"

# 启动Level 2节点
echo "2. 启动Level 2节点..."
./bin/cnet-agent -config config_level2.yaml > level2.log 2>&1 &
LEVEL2_PID=$!
sleep 3

# 检查Level 2节点是否启动成功
if ! curl -s http://localhost:8082/health > /dev/null; then
    echo "错误: Level 2节点启动失败"
    kill $DISCOVERY_PID $LEVEL2_PID 2>/dev/null
    exit 1
fi
echo "✓ Level 2节点启动成功"

# 检查Level 2节点是否已注册到根节点
echo "3. 检查Level 2节点注册状态..."
LEVEL2_REGISTERED=$(curl -s http://localhost:8080/api/discovery/nodes | jq -r '.[] | select(.id == "level2-node") | .hierarchy_id')
if [ "$LEVEL2_REGISTERED" != "null" ] && [ "$LEVEL2_REGISTERED" != "" ]; then
    echo "✓ Level 2节点已注册到根节点，层次化ID: $LEVEL2_REGISTERED"
else
    echo "错误: Level 2节点未成功注册到根节点"
    kill $DISCOVERY_PID $LEVEL2_PID 2>/dev/null
    exit 1
fi

# 启动Level 3节点，向Level 2节点注册
echo "4. 启动Level 3节点，向Level 2节点注册..."
./bin/cnet-agent -config config_level3.yaml > level3.log 2>&1 &
LEVEL3_PID=$!
sleep 3

# 检查Level 3节点是否启动成功
if ! curl -s http://localhost:8083/health > /dev/null; then
    echo "错误: Level 3节点启动失败"
    kill $DISCOVERY_PID $LEVEL2_PID $LEVEL3_PID 2>/dev/null
    exit 1
fi
echo "✓ Level 3节点启动成功"

# 检查Level 3节点是否已注册到Level 2节点
echo "5. 检查Level 3节点注册状态..."
LEVEL3_REGISTERED=$(curl -s http://localhost:8082/api/discovery/nodes | jq -r '.[] | select(.id == "level3-node") | .hierarchy_id')
if [ "$LEVEL3_REGISTERED" != "null" ] && [ "$LEVEL3_REGISTERED" != "" ]; then
    echo "✓ Level 3节点已注册到Level 2节点，层次化ID: $LEVEL3_REGISTERED"
else
    echo "错误: Level 3节点未成功注册到Level 2节点"
    echo "Level 2节点的节点列表:"
    curl -s http://localhost:8082/api/discovery/nodes | jq .
    kill $DISCOVERY_PID $LEVEL2_PID $LEVEL3_PID 2>/dev/null
    exit 1
fi

# 启动Level 4节点，向Level 3节点注册
echo "6. 启动Level 4节点，向Level 3节点注册..."
./bin/cnet-agent -config config_level4_node1.yaml > level4_node1.log 2>&1 &
LEVEL4_PID=$!
sleep 3

# 检查Level 4节点是否启动成功
if ! curl -s http://localhost:8084/health > /dev/null; then
    echo "错误: Level 4节点启动失败"
    kill $DISCOVERY_PID $LEVEL2_PID $LEVEL3_PID $LEVEL4_PID 2>/dev/null
    exit 1
fi
echo "✓ Level 4节点启动成功"

# 检查Level 4节点是否已注册到Level 3节点
echo "7. 检查Level 4节点注册状态..."
LEVEL4_REGISTERED=$(curl -s http://localhost:8083/api/discovery/nodes | jq -r '.[] | select(.id == "level4-node1") | .hierarchy_id')
if [ "$LEVEL4_REGISTERED" != "null" ] && [ "$LEVEL4_REGISTERED" != "" ]; then
    echo "✓ Level 4节点已注册到Level 3节点，层次化ID: $LEVEL4_REGISTERED"
else
    echo "错误: Level 4节点未成功注册到Level 3节点"
    echo "Level 3节点的节点列表:"
    curl -s http://localhost:8083/api/discovery/nodes | jq .
    kill $DISCOVERY_PID $LEVEL2_PID $LEVEL3_PID $LEVEL4_PID 2>/dev/null
    exit 1
fi

# 显示完整的层次化结构
echo "8. 显示完整的层次化结构..."
echo "根节点 (discovery-server) 的节点列表:"
curl -s http://localhost:8080/api/discovery/hierarchy/nodes | jq .

echo ""
echo "Level 2节点的节点列表:"
curl -s http://localhost:8082/api/discovery/hierarchy/nodes | jq .

echo ""
echo "Level 3节点的节点列表:"
curl -s http://localhost:8083/api/discovery/hierarchy/nodes | jq .

echo ""
echo "=== 测试完成 ==="
echo "所有节点都成功注册到相应的上级节点，层次化结构正常工作！"

# 清理
echo "清理测试环境..."
kill $DISCOVERY_PID $LEVEL2_PID $LEVEL3_PID $LEVEL4_PID 2>/dev/null
wait

echo "测试完成！"
