#!/bin/bash

# PostgreSQL 容器管理脚本
# 用于在本地开发环境中启动 PostgreSQL 容器

CONTAINER_NAME="postgres-dev"
POSTGRES_IMAGE="postgres:18.1-alpine"
POSTGRES_PORT=5432
POSTGRES_USER="postgres"
POSTGRES_PASSWORD="password"
POSTGRES_DB="im"

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
    log_info "创建 PostgreSQL 容器: ${CONTAINER_NAME}"
    podman run -d \
        --name "${CONTAINER_NAME}" \
        -p "${POSTGRES_PORT}:5432" \
        -e POSTGRES_USER="${POSTGRES_USER}" \
        -e POSTGRES_PASSWORD="${POSTGRES_PASSWORD}" \
        -e POSTGRES_DB="${POSTGRES_DB}" \
        "${POSTGRES_IMAGE}"

    if [ $? -eq 0 ]; then
        log_info "PostgreSQL 容器创建成功"
        log_info "PostgreSQL 地址: localhost:${POSTGRES_PORT}"
        log_info "数据库: ${POSTGRES_DB}"
        log_info "用户名: ${POSTGRES_USER}"
        log_info "密码: ${POSTGRES_PASSWORD}"

        # 等待 PostgreSQL 启动
        log_info "等待 PostgreSQL 启动..."
        sleep 5

        # 执行数据库迁移
        init_database
    else
        log_error "PostgreSQL 容器创建失败"
        exit 1
    fi
}

# 初始化数据库
init_database() {
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    MIGRATION_FILE="${SCRIPT_DIR}/../project/logic-go/migrations/001_init.sql"

    if [ -f "${MIGRATION_FILE}" ]; then
        log_info "执行数据库迁移: ${MIGRATION_FILE}"
        podman exec -i "${CONTAINER_NAME}" psql -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" < "${MIGRATION_FILE}"
        if [ $? -eq 0 ]; then
            log_info "数据库迁移完成"
        else
            log_warn "数据库迁移失败，请手动执行"
        fi
    else
        log_warn "未找到迁移文件: ${MIGRATION_FILE}"
    fi
}

# 启动已存在的容器
start_container() {
    log_info "启动已存在的 PostgreSQL 容器: ${CONTAINER_NAME}"
    podman start "${CONTAINER_NAME}"

    if [ $? -eq 0 ]; then
        log_info "PostgreSQL 容器启动成功"
    else
        log_error "PostgreSQL 容器启动失败"
        exit 1
    fi
}

# 主逻辑
main() {
    log_info "检查 PostgreSQL 开发环境..."

    check_podman

    if container_exists; then
        if container_running; then
            log_info "PostgreSQL 容器已在运行中"
        else
            log_warn "PostgreSQL 容器存在但未运行"
            start_container
        fi
    else
        log_warn "PostgreSQL 容器不存在"
        create_container
    fi

    log_info "PostgreSQL 开发环境就绪!"
}

main "$@"
