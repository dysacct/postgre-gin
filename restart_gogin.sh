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

docker rm -f cmdb-api


docker rmi gin-api:1.1

docker build -t gin-api:1.1 .

sleep 5

docker run -d \
  --name cmdb-api \
  --network host \
  --restart always \
  -e GIN_MODE=release \
  -e DB_HOST=localhost \
  -e DB_PORT=34189 \
  -e DB_USER=kuaicdn \
  -e DB_PASSWORD=abcd001002 \
  -e DB_NAME=machine_info \
  -e REDIS_HOST=localhost \
  -e REDIS_PORT=6379 \
  -e JWT_SECRET=9f8e7d6c5b4a3928170605f4e3d2c1b0a9f8e7d6c5b4a3928170605f4e3d2c1b0 \
  gin-api:1.1
