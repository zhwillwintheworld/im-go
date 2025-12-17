#!/bin/bash

# Desktop Web 启动脚本

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="${SCRIPT_DIR}/../../project/desktop-web"
ENV_FILE="${SCRIPT_DIR}/../.env-dev"

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[DESKTOP]${NC} $1"
}

cd "${PROJECT_DIR}"

# 加载环境变量
if [ -f "${ENV_FILE}" ]; then
    set -a
    source "${ENV_FILE}"
    set +a
    log_info "已加载环境变量: ${ENV_FILE}"
fi

# 检查 node_modules
if [ ! -d "node_modules" ]; then
    log_info "安装依赖..."
    npm install
fi

log_info "启动 Desktop Web 开发服务器..."
log_info "地址: http://localhost:${DESKTOP_PORT:-5173}"
echo ""

npm run dev
