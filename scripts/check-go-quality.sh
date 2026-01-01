#!/bin/bash

# Go 代码质量检查工具
# 支持按需检查指定模块，使用 golangci-lint 保证代码质量
# 用法: ./check-go-quality.sh [选项] [模块名...]

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# 项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
PROJECT_DIR="$PROJECT_ROOT/project"

# 显示帮助信息
show_help() {
    echo -e "${BLUE}Go 代码质量检查工具${NC}"
    echo ""
    echo -e "${BLUE}用法:${NC}"
    echo "  $0 [选项] [模块名...]"
    echo ""
    echo -e "${BLUE}选项:${NC}"
    echo "  -h, --help     显示帮助信息"
    echo "  -l, --list     列出所有可用模块"
    echo "  -f, --full     完整模式（更详细的 lint 检查）"
    echo ""
    echo -e "${BLUE}示例:${NC}"
    echo "  $0                      # 快速检查所有模块"
    echo "  $0 -f                   # 完整检查所有模块"
    echo "  $0 logic-go             # 快速检查 logic-go"
    echo "  $0 -f access-go logic-go # 完整检查 access-go 和 logic-go"
    echo ""
    echo -e "${BLUE}检查项目:${NC}"
    echo "  1. go fmt   - 代码格式检查"
    echo "  2. go vet   - 静态分析"
    echo "  3. golangci-lint - 代码质量检查"
    echo ""
}

# 列出所有模块
list_modules() {
    echo -e "${BLUE}可用的 Go 模块:${NC}"
    for go_mod in "$PROJECT_DIR"/*/go.mod; do
        if [ ! -f "$go_mod" ]; then
            continue
        fi
        module_name="$(basename "$(dirname "$go_mod")")"
        if [ "$module_name" != "desktop-web" ]; then
            echo "  - $module_name"
        fi
    done
}

# 检查单个模块
check_module() {
    local module_name=$1
    local full_mode=$2
    local module_dir="$PROJECT_DIR/$module_name"
    
    # 检查模块是否存在
    if [ ! -d "$module_dir" ] || [ ! -f "$module_dir/go.mod" ]; then
        echo -e "${RED}错误: 模块 '$module_name' 不存在或不是 Go 模块${NC}"
        return 1
    fi
    
    cd "$module_dir"
    local module_success=true
    
    # 1. go fmt
    echo -n "  fmt..."
    if go fmt ./... > /dev/null 2>&1; then
        echo -e " ${GREEN}✓${NC}"
    else
        echo -e " ${RED}✗${NC}"
        module_success=false
    fi
    
    # 2. go vet
    echo -n "  vet..."
    vet_output=$(go vet ./... 2>&1 || true)
    if [ -z "$vet_output" ]; then
        echo -e " ${GREEN}✓${NC}"
    else
        echo -e " ${RED}✗${NC}"
        if [ "$full_mode" = true ]; then
            echo "$vet_output" | head -10 | sed 's/^/    /'
        fi
        module_success=false
    fi
    
    # 3. golangci-lint
    echo -n "  lint..."
    if [ "$full_mode" = true ]; then
        # 完整模式：更详细的检查
        lint_output=$(golangci-lint run --timeout=5m ./... 2>&1 || true)
    else
        # 快速模式：减少超时时间
        lint_output=$(golangci-lint run --timeout=2m ./... 2>&1 || true)
    fi
    
    if echo "$lint_output" | grep -q "level=error"; then
        echo -e " ${RED}✗${NC}"
        echo -e "${RED}    错误详情:${NC}"
        echo "$lint_output" | grep "level=error" | head -10 | sed 's/^/    /'
        module_success=false
    elif [ -z "$lint_output" ]; then
        echo -e " ${GREEN}✓${NC}"
    else
        echo -e " ${YELLOW}⚠${NC}"
        echo -e "${YELLOW}    警告详情:${NC}"
        if [ "$full_mode" = true ]; then
            echo "$lint_output" | head -15 | sed 's/^/    /'
        else
            echo "$lint_output" | head -10 | sed 's/^/    /'
        fi
    fi
    
    if [ "$module_success" = true ]; then
        echo -e "${GREEN}  ✓ 通过${NC}"
        return 0
    else
        echo -e "${RED}  ✗ 失败${NC}"
        return 1
    fi
}

# 解析命令行参数
FULL_MODE=false
MODULES_TO_CHECK=()

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -l|--list)
            list_modules
            exit 0
            ;;
        -f|--full)
            FULL_MODE=true
            shift
            ;;
        *)
            MODULES_TO_CHECK+=("$1")
            shift
            ;;
    esac
done

# 检查 golangci-lint
if ! command -v golangci-lint &> /dev/null; then
    echo -e "${RED}错误: golangci-lint 未安装${NC}"
    echo "请运行: brew install golangci-lint"
    exit 1
fi

echo -e "${BLUE}================================${NC}"
if [ "$FULL_MODE" = true ]; then
    echo -e "${BLUE}Go 代码质量检查 (完整模式)${NC}"
else
    echo -e "${BLUE}Go 代码质量检查 (快速模式)${NC}"
fi
echo -e "${BLUE}================================${NC}"
echo ""

# 统计
TOTAL_MODULES=0
SUCCESS_MODULES=0
FAILED_MODULES=0
FAILED_MODULE_NAMES=()

# 确定要检查的模块列表
if [ ${#MODULES_TO_CHECK[@]} -eq 0 ]; then
    # 没有指定模块，检查所有
    echo -e "${CYAN}→ 检查所有模块${NC}"
    echo ""
    for go_mod in "$PROJECT_DIR"/*/go.mod; do
        if [ ! -f "$go_mod" ]; then
            continue
        fi
        module_name="$(basename "$(dirname "$go_mod")")"
        if [ "$module_name" != "desktop-web" ]; then
            MODULES_TO_CHECK+=("$module_name")
        fi
    done
else
    # 检查指定模块
    echo -e "${CYAN}→ 检查指定模块: ${MODULES_TO_CHECK[*]}${NC}"
    echo ""
fi

# 检查每个模块
for module_name in "${MODULES_TO_CHECK[@]}"; do
    TOTAL_MODULES=$((TOTAL_MODULES + 1))
    echo -e "${BLUE}[$TOTAL_MODULES] 检查: $module_name${NC}"
    
    if check_module "$module_name" "$FULL_MODE"; then
        SUCCESS_MODULES=$((SUCCESS_MODULES + 1))
    else
        FAILED_MODULES=$((FAILED_MODULES + 1))
        FAILED_MODULE_NAMES+=("$module_name")
    fi
    echo ""
done

# 输出总结
echo -e "${BLUE}================================${NC}"
echo "总模块: ${TOTAL_MODULES} | ${GREEN}通过: ${SUCCESS_MODULES}${NC} | ${RED}失败: ${FAILED_MODULES}${NC}"

if [ $FAILED_MODULES -gt 0 ]; then
    echo -e "\n${RED}失败的模块:${NC}"
    for module in "${FAILED_MODULE_NAMES[@]}"; do
        echo -e "  ✗ $module"
    done
    exit 1
else
    echo -e "\n${GREEN}🎉 所有检查通过！${NC}"
    exit 0
fi
