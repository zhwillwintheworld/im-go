#!/bin/bash

# FlatBuffers 统一代码生成脚本
# 一键生成 TypeScript 和 Go 代码

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "╔══════════════════════════════════════════════════════════╗"
echo "║        FlatBuffers 代码生成 - 统一构建脚本              ║"
echo "╚══════════════════════════════════════════════════════════╝"
echo ""

# 检查 flatc 是否安装
if ! command -v flatc &> /dev/null; then
    echo "❌ 错误: flatc 未安装"
    echo ""
    echo "请使用以下命令安装:"
    echo "  brew install flatbuffers"
    echo ""
    exit 1
fi

echo "✅ FlatBuffers 编译器已就绪: $(flatc --version)"
echo ""

# 记录开始时间
START_TIME=$(date +%s)

# 生成 TypeScript 代码
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📦 步骤 1/2: 生成 TypeScript 代码"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
bash "${SCRIPT_DIR}/generate-ts.sh"
TS_EXIT_CODE=$?

echo ""

# 生成 Go 代码
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📦 步骤 2/2: 生成 Go 代码"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
bash "${SCRIPT_DIR}/generate-go.sh"
GO_EXIT_CODE=$?

echo ""

# 计算耗时
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

# 总结
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 构建总结"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if [ $TS_EXIT_CODE -eq 0 ]; then
    echo "✅ TypeScript 代码生成成功"
else
    echo "❌ TypeScript 代码生成失败"
fi

if [ $GO_EXIT_CODE -eq 0 ]; then
    echo "✅ Go 代码生成成功"
else
    echo "❌ Go 代码生成失败"
fi

echo ""
echo "⏱️  总耗时: ${DURATION}s"
echo ""

# 显示新增的协议字段
if [ $TS_EXIT_CODE -eq 0 ] && [ $GO_EXIT_CODE -eq 0 ]; then
    echo "✨ 新增的协议字段:"
    echo "  • RoomAction.CHANGE_SEAT (换座位)"
    echo "  • RoomAction.START_GAME (开始游戏)"  
    echo "  • RoomReq.target_seat_index (目标座位索引)"
    echo ""
    echo "🚀 所有代码生成完成！"
    exit 0
else
    echo "⚠️  部分代码生成失败，请检查错误信息"
    exit 1
fi
