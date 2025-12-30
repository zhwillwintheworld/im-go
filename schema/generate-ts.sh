#!/bin/bash

# FlatBuffers TypeScript 代码生成脚本
# 需要先安装 flatc: brew install flatbuffers

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCHEMA_FILE="${SCRIPT_DIR}/message.fbs"
TS_OUTPUT_DIR="${SCRIPT_DIR}/../project/desktop-web/src/protocol/im/protocol"

# 检查 flatc 是否安装
if ! command -v flatc &> /dev/null; then
    echo "错误: flatc 未安装"
    echo "请使用以下命令安装:"
    echo "  brew install flatbuffers"
    exit 1
fi

# 创建输出目录
mkdir -p "${TS_OUTPUT_DIR}"

# 生成 TypeScript 代码
echo "生成 FlatBuffers TypeScript 代码..."
flatc --ts -o "${TS_OUTPUT_DIR}/.." "${SCHEMA_FILE}"

if [ $? -eq 0 ]; then
    echo "✅ TypeScript 代码生成成功: ${TS_OUTPUT_DIR}"
    echo ""
    echo "已生成的文件:"
    ls -la "${TS_OUTPUT_DIR}" | grep -E "\.(ts|js)$" | head -10
else
    echo "❌ 代码生成失败"
    exit 1
fi
