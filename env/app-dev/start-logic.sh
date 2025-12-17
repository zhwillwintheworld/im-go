#!/bin/bash

# Logic-Go 启动脚本

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="${SCRIPT_DIR}/../../project/logic-go"
ENV_FILE="${SCRIPT_DIR}/../.env-dev"

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[LOGIC]${NC} $1"
}

cd "${PROJECT_DIR}"

# 加载环境变量
if [ -f "${ENV_FILE}" ]; then
    set -a
    source "${ENV_FILE}"
    set +a
    log_info "已加载环境变量: ${ENV_FILE}"
fi

log_info "启动 Logic-Go 服务..."
echo ""

go run cmd/logic/main.go
