#!/bin/bash

# Desktop-Web 代码质量检查脚本
# 检查 TypeScript/React 代码质量

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DESKTOP_WEB_DIR="$PROJECT_ROOT/project/desktop-web"

echo -e "${BLUE}================================${NC}"
echo -e "${BLUE}Desktop-Web 代码质量检查${NC}"
echo -e "${BLUE}================================${NC}"
echo ""

cd "$DESKTOP_WEB_DIR"

SUCCESS=true

# 1. TypeScript 类型检查
echo -e "${BLUE}1/2 TypeScript 类型检查...${NC}"
tsc_output=$(npx tsc --noEmit 2>&1 || true)
if echo "$tsc_output" | grep -q "error TS"; then
    echo -e "${RED}  ✗ 类型检查失败${NC}"
    echo "$tsc_output" | grep "error TS" | head -10 | sed 's/^/    /'
    SUCCESS=false
else
    echo -e "${GREEN}  ✓ 类型检查通过${NC}"
fi
echo ""

# 2. ESLint 检查
echo -e "${BLUE}2/2 ESLint 代码质量检查...${NC}"
eslint_output=$(npm run lint 2>&1 || true)
if echo "$eslint_output" | grep -qE "^\s+[0-9]+:[0-9]+\s+(error|warning)"; then
    error_count=$(echo "$eslint_output" | grep -cE "^\s+[0-9]+:[0-9]+\s+error" || true)
    warning_count=$(echo "$eslint_output" | grep -cE "^\s+[0-9]+:[0-9]+\s+warning" || true)
    
    if [ "$error_count" -gt 0 ]; then
        echo -e "${RED}  ✗ ESLint 检查失败 (${error_count} 个错误, ${warning_count} 个警告)${NC}"
        echo "$eslint_output" | grep -E "^\s+[0-9]+:[0-9]+\s+error" | head -10 | sed 's/^/    /'
        SUCCESS=false
    else
        echo -e "${YELLOW}  ⚠ 有 ${warning_count} 个警告 (不影响通过)${NC}"
        echo "$eslint_output" | grep -E "^\s+[0-9]+:[0-9]+\s+warning" | head -5 | sed 's/^/    /'
    fi
else
    echo -e "${GREEN}  ✓ ESLint 检查通过${NC}"
fi
echo ""

# 总结
echo -e "${BLUE}================================${NC}"
if [ "$SUCCESS" = true ]; then
    echo -e "${GREEN}🎉 Desktop-Web 检查通过！${NC}"
    exit 0
else
    echo -e "${RED}❌ Desktop-Web 检查失败${NC}"
    exit 1
fi
