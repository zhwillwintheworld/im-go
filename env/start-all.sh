#!/bin/bash

# IM 开发环境一键启动脚本
# 启动所有依赖服务 (NATS, Redis, PostgreSQL)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

echo ""
echo "=========================================="
echo "     IM 开发环境启动脚本"
echo "=========================================="
echo ""

# 启动 NATS
log_step "1/3 启动 NATS..."
"${SCRIPT_DIR}/start-nats.sh"
echo ""

# 启动 Redis
log_step "2/3 启动 Redis..."
"${SCRIPT_DIR}/start-redis.sh"
echo ""

# 启动 PostgreSQL
log_step "3/3 启动 PostgreSQL..."
"${SCRIPT_DIR}/start-postgres.sh"
echo ""

echo "=========================================="
log_info "所有服务已就绪!"
echo ""
echo "服务端口信息:"
echo "  - NATS:       nats://localhost:4222"
echo "  - NATS HTTP:  http://localhost:8222"
echo "  - Redis:      localhost:6379"
echo "  - PostgreSQL: localhost:5432"
echo ""
echo "启动应用:"
echo "  - Access: cd project/access-go && go run cmd/access/main.go"
echo "  - Logic:  cd project/logic-go && go run cmd/logic/main.go"
echo "=========================================="
