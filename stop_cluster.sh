#!/bin/bash

# 停止CNET Agent集群

echo "停止所有CNET Agent..."
pkill -f cnet-agent

sleep 1

echo "清理日志文件..."
rm -rf logs/

echo "集群已停止"

