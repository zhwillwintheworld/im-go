#!/bin/bash

# 停止所有 IM 应用服务
# 关闭 Access-Go, Logic-Go, Desktop Web 进程

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# 查找并杀死进程
kill_process() {
    local keyword="$1"
    local app_name="$2"

    # 搜索进程PID (排除grep自身)
    # mac下使用 ps aux
    local pids=$(ps aux | grep "$keyword" | grep -v grep | awk '{print $2}')

    if [ -z "$pids" ]; then
        log_warn "$app_name 未运行 ($keyword)"
        return
    fi
    
    log_info "正在停止 $app_name ($keyword)..."
    
    for pid in $pids; do
        # 再次确认进程存在
        if ps -p "$pid" > /dev/null 2>&1; then
             # 打印进程信息 (为了美观，截取一部分)
             local p_info=$(ps -p "$pid" -o command= | head -n 1)
             if [ ${#p_info} -gt 60 ]; then
                 p_info="${p_info:0:60}..."
             fi
             echo -e "  -> 找到进程 [PID:$pid]: $p_info"
             
             kill "$pid" 2>/dev/null
             
             # 简单的等待和检查
             sleep 0.2
             if ps -p "$pid" > /dev/null 2>&1; then
                 kill -9 "$pid" 2>/dev/null
                 echo -e "  -> [PID:$pid] 已强制停止"
             else
                 echo -e "  -> [PID:$pid] 已停止"
             fi
        fi
    done
}

echo ""
echo "=========================================="
echo "     IM 应用服务停止脚本"
echo "=========================================="
echo ""

# 停止 Access-Go
kill_process "cmd/access/main.go" "Access-Go (Source)"
kill_process "access-go" "Access-Go (Binary)"

# 停止 Logic-Go
kill_process "cmd/logic/main.go" "Logic-Go (Source)"
kill_process "logic-go" "Logic-Go (Binary)"

# 停止 Desktop Web (Vite)
kill_process "vite" "Desktop Web (Vite)"
kill_process "npm run dev" "Desktop Web (NPM)"

# 检查端口占用并杀死进程
check_port() {
    local port=$1
    local name=$2
    
    # 使用 lsof 获取占用端口的 PID
    # -t: 仅输出 PID
    local pids=$(lsof -i :"$port" -sTCP:LISTEN -t)

    if [ -z "$pids" ]; then
        log_info "端口 $port ($name) ✓ 已释放"
        return
    fi

    log_warn "端口 $port ($name) 仍被占用，正在清理..."
    
    for pid in $pids; do
        if ps -p "$pid" > /dev/null 2>&1; then
             local p_info=$(ps -p "$pid" -o command= | head -n 1)
             if [ ${#p_info} -gt 60 ]; then
                 p_info="${p_info:0:60}..."
             fi
             echo -e "  -> 发现进程 [PID:$pid]: $p_info"
             
             log_info "  -> 杀死进程 [PID:$pid]..."
             kill "$pid" 2>/dev/null
             
             sleep 0.5
             
             if ps -p "$pid" > /dev/null 2>&1; then
                 kill -9 "$pid" 2>/dev/null
                 echo -e "  -> [PID:$pid] 已强制停止"
             else
                 echo -e "  -> [PID:$pid] 已停止"
             fi
        fi
    done
    
    # 再次检查
    if lsof -i :"$port" -sTCP:LISTEN -t >/dev/null 2>&1; then
        log_warn "端口 $port ($name) 清理失败，请手动检查"
    else
        log_info "端口 $port ($name) ✓ 已释放 (清理后)"
    fi
}

# 验证
sleep 1

echo ""
log_info "进程与端口检查:"

if pgrep -f "access" > /dev/null; then
    log_warn "Access 进程可能仍在运行"
else
    log_info "Access 进程 ✓ 已停止"
fi

if pgrep -f "logic" > /dev/null; then
    log_warn "Logic 进程可能仍在运行"
else
    log_info "Logic 进程 ✓ 已停止"
fi

if pgrep -f "vite" > /dev/null; then
    log_warn "Desktop 进程可能仍在运行"
else
    log_info "Desktop 进程 ✓ 已停止"
fi

echo ""
check_port 8443 "Access QUIC"
check_port 8080 "Access HTTP"
check_port 8081 "Logic HTTP"
check_port 3000 "Desktop Web"

echo ""
echo "=========================================="
log_info "应用服务已停止"
echo "=========================================="
