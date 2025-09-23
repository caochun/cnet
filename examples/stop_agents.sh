#!/bin/bash

# CNET Agent 停止脚本
# 用于停止所有运行的CNET Agent进程

echo "🛑 停止CNET Agent..."
echo "===================="

# 查找并停止所有cnet-agent进程
PIDS=$(pgrep -f "cnet-agent")
if [ -n "$PIDS" ]; then
    echo "📋 找到以下CNET Agent进程:"
    ps -p $PIDS -o pid,ppid,cmd
    echo ""
    
    echo "🛑 正在停止进程..."
    for pid in $PIDS; do
        echo "   停止进程 $pid"
        kill -TERM $pid 2>/dev/null
    done
    
    # 等待进程优雅退出
    sleep 3
    
    # 检查是否还有进程在运行
    REMAINING=$(pgrep -f "cnet-agent")
    if [ -n "$REMAINING" ]; then
        echo "⚠️  部分进程未响应，强制停止..."
        for pid in $REMAINING; do
            echo "   强制停止进程 $pid"
            kill -KILL $pid 2>/dev/null
        done
    fi
    
    echo "✅ 所有CNET Agent进程已停止"
else
    echo "ℹ️  没有找到运行中的CNET Agent进程"
fi

# 清理日志文件（可选）
if [ "$1" = "--clean" ]; then
    echo ""
    echo "🧹 清理日志文件..."
    rm -f agent1.log agent2.log
    echo "✅ 日志文件已清理"
fi

echo ""
echo "🎉 停止完成！"
