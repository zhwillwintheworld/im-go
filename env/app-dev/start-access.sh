#!/bin/bash

# Access-Go 启动脚本

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="${SCRIPT_DIR}/../../project/access-go"

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[ACCESS]${NC} $1"
}

cd "${PROJECT_DIR}"

log_info "启动 Access-Go 服务..."
log_info "QUIC 端口: :8443"
log_info "健康检查: http://localhost:8080/health"
echo ""

go run cmd/access/main.go
