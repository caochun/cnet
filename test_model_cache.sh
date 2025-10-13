#!/bin/bash

# 测试Vision模型缓存性能

set -e

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║           Vision 模型缓存性能测试                             ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# 准备测试图片
if [ ! -f "test_images/test.jpg" ]; then
    echo "下载测试图片..."
    mkdir -p test_images
    curl -s -o test_images/test.jpg https://raw.githubusercontent.com/opencv/opencv/master/samples/data/lena.jpg
fi

# 启动agent
echo "启动CNET Agent..."
./bin/cnet-agent -config config.yaml > agent_cache_test.log 2>&1 &
AGENT_PID=$!
sleep 2

echo "Agent PID: $AGENT_PID"
echo ""

echo "【测试场景】连续提交5个人脸检测任务"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 提交5个任务
for i in {1..5}; do
    echo "提交任务 $i/5..."
    
    START_TIME=$(date +%s%N)
    
    RESULT=$(curl -s -X POST http://localhost:8080/api/workloads \
      -H "Content-Type: application/json" \
      -d "{
        \"name\": \"face-test-$i\",
        \"type\": \"vision\",
        \"requirements\": {\"cpu\": 1.0, \"memory\": 536870912},
        \"config\": {
          \"task\": \"face_detection\",
          \"input_path\": \"test_images/test.jpg\",
          \"output_path\": \"test_output/result_$i.jpg\"
        }
      }")
    
    END_TIME=$(date +%s%N)
    DURATION=$(( (END_TIME - START_TIME) / 1000000 ))
    
    STATUS=$(echo "$RESULT" | jq -r '.status')
    WORKLOAD_ID=$(echo "$RESULT" | jq -r '.id')
    
    echo "  状态: $STATUS"
    echo "  耗时: ${DURATION}ms"
    echo "  ID: $WORKLOAD_ID"
    echo ""
    
    # 短暂等待，避免并发
    sleep 0.5
done

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

echo "【分析日志中的缓存使用情况】"
echo ""
echo "首次加载模型的日志:"
grep "Loading cascade" agent_cache_test.log | head -1 || echo "  (未找到)"
echo ""
echo "使用缓存的日志:"
grep "Using cached" agent_cache_test.log | head -3 || echo "  (未找到)"
echo ""

echo "【查看所有任务状态】"
curl -s http://localhost:8080/api/workloads | jq '.workloads[] | {name, status, results: (.results | length)}'
echo ""

echo "【资源使用情况】"
curl -s http://localhost:8080/api/resources/stats | jq '{
  used_cpu: .local_resources.used.cpu,
  used_memory_mb: (.local_resources.used.memory / 1048576 | floor),
  workloads: .workloads_count
}'
echo ""

# 停止agent
echo "停止Agent..."
kill $AGENT_PID
sleep 1

echo ""
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║                    测试完成                                   ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""
echo "【模型缓存机制说明】"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "第1次任务:"
echo "  • 从磁盘加载 Haar Cascade 模型 (慢)"
echo "  • 加载到内存并缓存"
echo "  • 执行人脸检测"
echo ""
echo "第2-5次任务:"
echo "  • 直接使用内存中的缓存模型 (快！)"
echo "  • 无需重新加载"
echo "  • 速度提升明显"
echo ""
echo "模型在内存中保持到:"
echo "  ✓ Agent停止"
echo "  ✓ 手动清理缓存"
echo ""
echo "优势:"
echo "  ✓ 批量处理速度快"
echo "  ✓ 减少磁盘I/O"
echo "  ✓ 多任务并发高效"
echo ""
echo "查看详细日志:"
echo "  cat agent_cache_test.log | grep -E 'Loading|cached'"
echo ""

