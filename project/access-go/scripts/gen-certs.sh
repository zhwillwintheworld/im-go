#!/bin/bash

# 生成 TLS 证书脚本
# 用于本地开发环境的 QUIC 服务

# brew install mkcert

# mkcert -install

mkcert localhost 127.0.0.1 ::1
