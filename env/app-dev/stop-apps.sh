#!/bin/bash

# 停止所有 IM 应用服务
# 关闭 Access-Go, Logic-Go, Desktop Web 进程

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

echo ""
echo "=========================================="
echo "     IM 应用服务停止脚本"
echo "=========================================="
echo ""

# 停止 Access-Go
log_info "停止 Access-Go..."
pkill -f "go run cmd/access/main.go" 2>/dev/null || log_warn "Access-Go 未运行"
pkill -f "access-go" 2>/dev/null

# 停止 Logic-Go
log_info "停止 Logic-Go..."
pkill -f "go run cmd/logic/main.go" 2>/dev/null || log_warn "Logic-Go 未运行"
pkill -f "logic-go" 2>/dev/null

# 停止 Desktop Web (Vite)
log_info "停止 Desktop Web..."
pkill -f "vite" 2>/dev/null || log_warn "Desktop Web 未运行"
pkill -f "npm run dev" 2>/dev/null

# 验证
sleep 1

echo ""
log_info "进程检查:"

if pgrep -f "access" > /dev/null; then
    log_warn "Access 进程可能仍在运行"
else
    log_info "Access ✓ 已停止"
fi

if pgrep -f "logic" > /dev/null; then
    log_warn "Logic 进程可能仍在运行"
else
    log_info "Logic ✓ 已停止"
fi

if pgrep -f "vite" > /dev/null; then
    log_warn "Desktop 进程可能仍在运行"
else
    log_info "Desktop ✓ 已停止"
fi

echo ""
echo "=========================================="
log_info "应用服务已停止"
echo "=========================================="
