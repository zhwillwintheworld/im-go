#!/bin/bash

# 停止所有 IM 开发环境容器

CONTAINERS=("nats-dev" "redis-dev" "postgres-dev")

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

echo ""
echo "=========================================="
echo "     IM 开发环境停止脚本"
echo "=========================================="
echo ""

for container in "${CONTAINERS[@]}"; do
    if podman ps --format "{{.Names}}" | grep -q "^${container}$"; then
        log_info "停止容器: ${container}"
        podman stop "${container}"
    else
        log_warn "容器未运行: ${container}"
    fi
done

echo ""
log_info "所有容器已停止"
echo "=========================================="
