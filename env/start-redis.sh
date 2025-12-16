#!/bin/bash

# Redis 容器管理脚本
# 用于在本地开发环境中启动 Redis 容器

CONTAINER_NAME="redis-dev"
REDIS_IMAGE="redis:8.4-alpine"
REDIS_PORT=6379

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

# 创建并启动容器
create_container() {
    log_info "创建 Redis 容器: ${CONTAINER_NAME}"
    podman run -d \
        --name "${CONTAINER_NAME}" \
        -p "${REDIS_PORT}:6379" \
        "${REDIS_IMAGE}"

    if [ $? -eq 0 ]; then
        log_info "Redis 容器创建成功"
        log_info "Redis 地址: localhost:${REDIS_PORT}"
    else
        log_error "Redis 容器创建失败"
        exit 1
    fi
}

# 启动已存在的容器
start_container() {
    log_info "启动已存在的 Redis 容器: ${CONTAINER_NAME}"
    podman start "${CONTAINER_NAME}"

    if [ $? -eq 0 ]; then
        log_info "Redis 容器启动成功"
    else
        log_error "Redis 容器启动失败"
        exit 1
    fi
}

# 主逻辑
main() {
    log_info "检查 Redis 开发环境..."

    check_podman

    if container_exists; then
        if container_running; then
            log_info "Redis 容器已在运行中"
        else
            log_warn "Redis 容器存在但未运行"
            start_container
        fi
    else
        log_warn "Redis 容器不存在"
        create_container
    fi

    log_info "Redis 开发环境就绪!"
}

main "$@"
