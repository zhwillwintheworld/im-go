#!/bin/bash

# 生成用于 WebTransport 的自签名证书 (ECDSA)
# 使用 PRIME256V1 曲线

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CERT_DIR="${SCRIPT_DIR}"

mkdir -p "${CERT_DIR}"

# 1. 生成私钥 (ECDSA)
openssl ecparam -name prime256v1 -genkey -noout -out "${CERT_DIR}/localhost.key"

# 2. 生成证书
# 只有 10 天有效期，因为 WebTransport 限制了自签名证书的最长有效期（通常为 14 天）
openssl req -new -x509 -key "${CERT_DIR}/localhost.key" -out "${CERT_DIR}/localhost.crt" -days 14 -subj "/CN=localhost" -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"

# 3. 计算并输出 SHA-256 指纹 (Base64 编码)
# WebTransport API 需要这种格式
FINGERPRINT=$(openssl x509 -in "${CERT_DIR}/localhost.crt" -outform DER | openssl dgst -sha256 -binary | base64)

echo ""
echo "Certificate generated!"
echo "Cert file: ${CERT_DIR}/localhost.crt"
echo "Key file:  ${CERT_DIR}/localhost.key"
echo ""
echo "serverCertificateHashes:"
echo "  - { algorithm: \"sha-256\", value: \"${FINGERPRINT}\" }"
echo ""
