#!/bin/bash

# PostgreSQL 容器管理脚本
# 用于在本地开发环境中启动 PostgreSQL 容器
# 每次启动都会删除旧容器，重新创建并初始化数据

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONTAINER_NAME="postgres-dev"
POSTGRES_IMAGE="postgres:16-alpine"
POSTGRES_PORT=5432
POSTGRES_USER="postgres"
POSTGRES_PASSWORD="password"
POSTGRES_DB="im_db"

# SQL 文件
SCHEMA_FILE="${SCRIPT_DIR}/schema.sql"
DML_FILE="${SCRIPT_DIR}/dml.sql"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

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
    else
        log_error "PostgreSQL 容器创建失败"
        exit 1
    fi
}

# 等待 PostgreSQL 就绪
wait_for_postgres() {
    log_info "等待 PostgreSQL 启动..."
    local retries=30
    while [ $retries -gt 0 ]; do
        if podman exec "${CONTAINER_NAME}" pg_isready -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" &>/dev/null; then
            log_info "PostgreSQL 已就绪"
            return 0
        fi
        retries=$((retries - 1))
        sleep 1
    done
    log_error "PostgreSQL 启动超时"
    exit 1
}

# 执行 Schema 初始化
init_schema() {
    if [ -f "${SCHEMA_FILE}" ]; then
        log_info "执行 Schema: ${SCHEMA_FILE}"
        podman exec -i "${CONTAINER_NAME}" psql -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" < "${SCHEMA_FILE}"
        if [ $? -eq 0 ]; then
            log_info "Schema 初始化完成"
        else
            log_error "Schema 初始化失败"
            exit 1
        fi
    else
        log_warn "未找到 Schema 文件: ${SCHEMA_FILE}"
    fi
}

# 执行 DML 初始化
init_dml() {
    if [ -f "${DML_FILE}" ]; then
        log_info "执行 DML: ${DML_FILE}"
        podman exec -i "${CONTAINER_NAME}" psql -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" < "${DML_FILE}"
        if [ $? -eq 0 ]; then
            log_info "DML 初始化完成"
        else
            log_warn "DML 初始化失败，请检查数据"
        fi
    else
        log_warn "未找到 DML 文件: ${DML_FILE}"
    fi
}

# 主逻辑
main() {
    echo ""
    echo "=========================================="
    echo "     PostgreSQL 开发环境初始化"
    echo "=========================================="
    echo ""

    check_podman

    # 删除旧容器，重新创建
    remove_container
    create_container
    wait_for_postgres

    # 初始化数据库
    init_schema
    init_dml

    echo ""
    echo "=========================================="
    log_info "PostgreSQL 开发环境就绪!"
    echo ""
    echo "连接信息:"
    echo "  Host:     localhost"
    echo "  Port:     ${POSTGRES_PORT}"
    echo "  Database: ${POSTGRES_DB}"
    echo "  User:     ${POSTGRES_USER}"
    echo "  Password: ${POSTGRES_PASSWORD}"
    echo "=========================================="
}

main "$@"
