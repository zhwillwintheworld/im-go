#!/bin/bash

INPUT="src/assets/mahjong/image.png"
OUTPUT_DIR="src/assets/mahjong"

# 每张牌的尺寸
TILE_WIDTH=80
TILE_HEIGHT=129

# 裁剪函数
crop_tile() {
    local type=$1
    local value=$2
    local x=$3
    local y=$4
    local output="${OUTPUT_DIR}/${type}-${value}.png"
    
    # 使用 sips 裁剪（macOS 自带工具）
    sips -c $TILE_HEIGHT $TILE_WIDTH "$INPUT" \
         --cropOffset $y $x \
         --out "$output" > /dev/null 2>&1
    
    if [ $? -eq 0 ]; then
        echo "Created $output"
    else
        echo "Error creating $output"
    fi
}

# 万牌：第0行，0-8列
for col in {0..8}; do
    value=$((col + 1))
    x=$((col * TILE_WIDTH))
    y=0
    crop_tile "wan" $value $x $y
done

# 条牌：第1行，0-8列
for col in {0..8}; do
    value=$((col + 1))
    x=$((col * TILE_WIDTH))
    y=$TILE_HEIGHT
    crop_tile "tiao" $value $x $y
done

# 筒牌：第2行，0-8列
for col in {0..8}; do
    value=$((col + 1))
    x=$((col * TILE_WIDTH))
    y=$((TILE_HEIGHT * 2))
    crop_tile "tong" $value $x $y
done

echo "完成！"
