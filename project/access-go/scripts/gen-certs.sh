#!/bin/bash

# 生成 TLS 证书脚本
# 用于本地开发环境的 QUIC 服务

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CERT_DIR="${SCRIPT_DIR}/../certs"

mkdir -p "${CERT_DIR}"

# 生成自签名证书
openssl req -x509 \
    -newkey rsa:4096 \
    -keyout "${CERT_DIR}/server.key" \
    -out "${CERT_DIR}/server.crt" \
    -days 365 \
    -nodes \
    -subj "/CN=localhost" \
    -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"

if [ $? -eq 0 ]; then
    echo "证书生成成功:"
    echo "  - 证书: ${CERT_DIR}/server.crt"
    echo "  - 私钥: ${CERT_DIR}/server.key"
    echo ""
    echo "注意: 这是自签名证书，仅用于开发环境"
else
    echo "证书生成失败"
    exit 1
fi
