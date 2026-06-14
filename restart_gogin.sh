#!/bin/bash
set -e

echo "=== CMDB 服务重启 ==="

# 停止并移除旧容器
docker rm -f cmdb_frontend cmdb_api 2>/dev/null || true

# 重新构建并启动
docker compose up -d --build

echo "=== 启动完成 ==="
echo "API:  http://$(hostname -I 2>/dev/null || echo 'localhost'):34185"
echo "Web:  http://$(hostname -I 2>/dev/null || echo 'localhost'):34186"
