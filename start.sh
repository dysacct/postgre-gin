#!/bin/bash
set -e

PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
echo "=== CMDB 服务启动 ==="
echo "项目目录: $PROJECT_DIR"

# 1. 构建前端
echo ""
echo "[1/4] 构建 Vue3 前端..."
cd "$PROJECT_DIR/frontend"
pnpm install
pnpm build
echo "前端构建完成 → $PROJECT_DIR/frontend/dist"

# 2. 部署前端到 OpenResty 目录
echo ""
echo "[2/4] 部署前端静态文件..."
sudo mkdir -p /app/dist
sudo cp -r "$PROJECT_DIR/frontend/dist/"* /app/dist/
echo "前端已部署到 /app/dist/"

# 3. 构建 Go 后端
echo ""
echo "[3/4] 构建 Go API..."
cd "$PROJECT_DIR"
go build -o cmdb-api .
echo "Go API 构建完成 → $PROJECT_DIR/cmdb-api"

# 4. 配置 OpenResty
echo ""
echo "[4/4] 配置 OpenResty..."
sudo cp "$PROJECT_DIR/openresty/nginx.conf" /usr/local/openresty/nginx/conf/nginx.conf

# 重启 OpenResty
sudo /usr/local/openresty/bin/openresty -t && sudo /usr/local/openresty/bin/openresty -s reload 2>/dev/null || sudo /usr/local/openresty/bin/openresty
echo "OpenResty 已启动/重载 (端口 80)"

# 5. 启动 Go API（后台运行）
echo ""
echo "启动 Go API (端口 34185)..."
export DB_HOST="${DB_HOST:-3418.s.kuaicdn.cn}"
export DB_PORT="${DB_PORT:-34189}"
export DB_USER="${DB_USER:-kuaicdn}"
export DB_PASSWORD="${DB_PASSWORD:-abcd001002}"
export DB_NAME="${DB_NAME:-machine_info}"
export DB_SSL_MODE="${DB_SSL_MODE:-disable}"
export JWT_SECRET="${JWT_SECRET:-9f8e7d6c5b4a3928170605f4e3d2c1b0a9f8e7d6c5b4a3928170605f4e3d2c1b0}"
export JWT_EXPIRE="${JWT_EXPIRE:-72}"

# 先杀掉旧进程
pkill -f "cmdb-api" 2>/dev/null || true
sleep 1

nohup "$PROJECT_DIR/cmdb-api" > /var/log/cmdb-api.log 2>&1 &
echo "Go API 已启动 (PID: $!)"

echo ""
echo "=== 启动完成 ==="
echo "Web 前端: http://$(hostname -I 2>/dev/null | awk '{print $1}' || echo 'localhost')"
echo "API 端口: 34185"
echo "API 日志: /var/log/cmdb-api.log"
echo ""
echo "登录账号: admin (管理员) / bdkj (普通用户)"
