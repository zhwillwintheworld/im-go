#!/bin/bash

# FlatBuffers 代码生成脚本
# 需要先安装 flatc: brew install flatbuffers

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCHEMA_FILE="${SCRIPT_DIR}/message.fbs"
OUTPUT_DIR="${SCRIPT_DIR}/../pkg/flatbuf"

# 检查 flatc 是否安装
if ! command -v flatc &> /dev/null; then
    echo "错误: flatc 未安装"
    echo "请使用以下命令安装:"
    echo "  brew install flatbuffers"
    exit 1
fi

# 创建输出目录
mkdir -p "${OUTPUT_DIR}"

# 生成 Go 代码
echo "生成 FlatBuffers Go 代码..."
flatc --go -o "${OUTPUT_DIR}" "${SCHEMA_FILE}"

if [ $? -eq 0 ]; then
    echo "代码生成成功: ${OUTPUT_DIR}"
    ls -la "${OUTPUT_DIR}"
else
    echo "代码生成失败"
    exit 1
fi
