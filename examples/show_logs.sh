#!/bin/bash

# CNET Agent 日志文件位置演示脚本

echo "📁 CNET Agent ProcessExecutor 日志文件位置"
echo "=========================================="
echo ""

# 获取系统临时目录
TEMP_DIR=$(go run -c 'import "os"; fmt.Println(os.TempDir())' 2>/dev/null || echo "$TMPDIR")
LOGS_DIR="$TEMP_DIR/cnet/logs"

echo "🔍 日志文件位置:"
echo "   目录: $LOGS_DIR"
echo ""

# 检查目录是否存在
if [ -d "$LOGS_DIR" ]; then
    echo "✅ 日志目录存在"
    echo ""
    echo "📋 目录内容:"
    ls -la "$LOGS_DIR" 2>/dev/null || echo "   目录为空"
    echo ""
    
    # 显示文件数量
    FILE_COUNT=$(find "$LOGS_DIR" -name "*.log" 2>/dev/null | wc -l)
    echo "📊 日志文件数量: $FILE_COUNT"
    
    if [ $FILE_COUNT -gt 0 ]; then
        echo ""
        echo "📄 最新的日志文件:"
        find "$LOGS_DIR" -name "*.log" -type f -exec ls -lt {} + | head -5
    fi
else
    echo "❌ 日志目录不存在"
    echo "   目录将在创建第一个任务时自动创建"
fi

echo ""
echo "🔧 日志文件命名规则:"
echo "   格式: {task-id}.log"
echo "   示例: 94ed6336-19da-497a-b3a3-506f516c05b5.log"
echo ""

echo "📝 日志文件内容:"
echo "   • 进程的标准输出 (stdout)"
echo "   • 进程的标准错误 (stderr)"
echo "   • 所有输出都重定向到同一个日志文件"
echo ""

echo "🌐 通过Web UI查看日志:"
echo "   • 访问: http://localhost:8080"
echo "   • 进入 Tasks 页面"
echo "   • 点击任务ID查看详情"
echo "   • 在任务详情中查看日志"
echo ""

echo "🔌 通过API查看日志:"
echo "   curl http://localhost:8080/api/tasks/{task-id}/logs"
echo ""

echo "💡 提示:"
echo "   • 日志文件在任务创建时自动生成"
echo "   • 文件路径存储在任务的 LogFile 字段中"
echo "   • 可以通过 Web UI 或 API 查看日志内容"
echo "   • 日志文件会一直保留，直到手动删除"
