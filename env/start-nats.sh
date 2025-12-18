#!/bin/bash

# NATS 容器管理脚本
# 用于在本地开发环境中启动 NATS 容器

CONTAINER_NAME="nats-dev"
NATS_IMAGE="nats:latest"
NATS_PORT=4222
NATS_HTTP_PORT=8222

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

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查 podman 是否安装
check_podman() {
    if ! command -v podman &> /dev/null; then
        log_error "podman 未安装，请先安装 podman"
        exit 1
    fi
    log_info "podman 已安装: $(podman --version)"
}

# 检查容器是否存在
container_exists() {
    podman ps -a --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"
}

# 检查容器是否运行中
container_running() {
    podman ps --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"
}

# 删除旧容器
remove_container() {
    if container_exists; then
        log_warn "删除旧容器: ${CONTAINER_NAME}"
        podman stop "${CONTAINER_NAME}" 2>/dev/null
        podman rm "${CONTAINER_NAME}" 2>/dev/null
        log_info "旧容器已删除"
    fi
}

# 创建并启动容器
create_container() {
    log_info "创建 NATS 容器: ${CONTAINER_NAME}"
    podman run -d \
        --name "${CONTAINER_NAME}" \
        -p "${NATS_PORT}:4222" \
        -p "${NATS_HTTP_PORT}:8222" \
        "${NATS_IMAGE}"

    if [ $? -eq 0 ]; then
        log_info "NATS 容器创建成功"
        log_info "NATS 地址: nats://localhost:${NATS_PORT}"
        log_info "HTTP 监控: http://localhost:${NATS_HTTP_PORT}"
    else
        log_error "NATS 容器创建失败"
        exit 1
    fi
}

# 主逻辑
main() {
    echo ""
    echo "=========================================="
    echo "     NATS 开发环境初始化"
    echo "=========================================="
    echo ""

    check_podman

    # 删除旧容器，重新创建
    remove_container
    create_container

    echo ""
    echo "=========================================="
    log_info "NATS 开发环境就绪!"
    echo ""
    echo "连接信息:"
    echo "  NATS:  nats://localhost:${NATS_PORT}"
    echo "  HTTP:  http://localhost:${NATS_HTTP_PORT}"
    echo "=========================================="
}

main "$@"
