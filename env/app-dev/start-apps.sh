#!/bin/bash

# 启动所有应用服务 (Access + Logic + Desktop)
# 注意: 需要先启动依赖服务 (./start-all.sh)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

echo ""
echo "=========================================="
echo "     IM 应用服务启动脚本"
echo "=========================================="
echo ""

# 检查依赖服务
log_info "检查依赖服务..."

# 检查 NATS
if ! podman ps --format "{{.Names}}" 2>/dev/null | grep -q "nats-dev"; then
    log_warn "NATS 未运行，请先执行: ./start-all.sh"
    exit 1
fi

# 检查 Redis
if ! podman ps --format "{{.Names}}" 2>/dev/null | grep -q "redis-dev"; then
    log_warn "Redis 未运行，请先执行: ./start-all.sh"
    exit 1
fi

# 检查 PostgreSQL
if ! podman ps --format "{{.Names}}" 2>/dev/null | grep -q "postgres-dev"; then
    log_warn "PostgreSQL 未运行，请先执行: ./start-all.sh"
    exit 1
fi

log_info "依赖服务检查通过 ✓"
echo ""

# 启动服务 (使用新终端)
log_step "启动服务..."
echo ""

# 检测终端类型
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS - 使用 Terminal.app
    log_info "在新终端窗口中启动服务..."

    # 启动 Logic
    osascript -e "tell application \"Terminal\" to do script \"cd '${SCRIPT_DIR}' && ./start-logic.sh\""
    sleep 2

    # 启动 Web
    osascript -e "tell application \"Terminal\" to do script \"cd '${SCRIPT_DIR}' && ./start-web.sh\""
    sleep 2

    # 启动 Access
    osascript -e "tell application \"Terminal\" to do script \"cd '${SCRIPT_DIR}' && ./start-access.sh\""
    sleep 2

    # 启动 Desktop
    osascript -e "tell application \"Terminal\" to do script \"cd '${SCRIPT_DIR}' && ./start-desktop.sh\""

else
    # Linux - 使用 tmux (如果可用)
    if command -v tmux &> /dev/null; then
        log_info "使用 tmux 启动服务..."

        tmux new-session -d -s im-services
        tmux send-keys -t im-services "${SCRIPT_DIR}/start-logic.sh" C-m
        tmux split-window -h -t im-services
        tmux send-keys -t im-services "${SCRIPT_DIR}/start-web.sh" C-m
        tmux split-window -v -t im-services
        tmux send-keys -t im-services "${SCRIPT_DIR}/start-access.sh" C-m
        tmux split-window -v -t im-services
        tmux send-keys -t im-services "${SCRIPT_DIR}/start-desktop.sh" C-m
        tmux attach -t im-services
    else
        log_warn "未检测到 tmux，请手动在不同终端中启动服务"
        echo ""
        echo "  终端1: ./start-logic.sh"
        echo "  终端2: ./start-web.sh"
        echo "  终端3: ./start-access.sh"
        echo "  终端4: ./start-desktop.sh"
        exit 0
    fi
fi

echo ""
echo "=========================================="
log_info "所有服务已启动!"
echo ""
echo "服务地址:"
echo "  - Access QUIC: :8443"
echo "  - Logic:       :8081"
echo "  - Web API:     :8082"
echo "  - Desktop Web: :5173"
echo "=========================================="
