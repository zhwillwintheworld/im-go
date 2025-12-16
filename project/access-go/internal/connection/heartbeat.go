package connection

import (
	"context"
	"log/slog"
	"time"
)

// HeartbeatChecker 心跳超时检测器
type HeartbeatChecker struct {
	manager       *Manager
	timeout       time.Duration
	checkInterval time.Duration
	logger        *slog.Logger
	onTimeout     func(conn *Connection) // 超时回调
}

// NewHeartbeatChecker 创建心跳检测器
func NewHeartbeatChecker(manager *Manager, timeout, checkInterval time.Duration, logger *slog.Logger, onTimeout func(conn *Connection)) *HeartbeatChecker {
	// 设置默认值
	if timeout <= 0 {
		timeout = 90 * time.Second
	}
	if checkInterval <= 0 {
		checkInterval = 30 * time.Second
	}

	return &HeartbeatChecker{
		manager:       manager,
		timeout:       timeout,
		checkInterval: checkInterval,
		logger:        logger,
		onTimeout:     onTimeout,
	}
}

// Start 启动心跳检测（阻塞，应在 goroutine 中调用）
func (h *HeartbeatChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()

	h.logger.Info("Heartbeat checker started",
		"timeout", h.timeout,
		"check_interval", h.checkInterval)

	for {
		select {
		case <-ctx.Done():
			h.logger.Info("Heartbeat checker stopped")
			return
		case <-ticker.C:
			h.checkConnections()
		}
	}
}

// checkConnections 检查所有连接的心跳是否超时
func (h *HeartbeatChecker) checkConnections() {
	conns := h.manager.GetAllConnections()
	now := time.Now()
	timeoutCount := 0

	for _, conn := range conns {
		lastActive := conn.LastActiveTime()
		if now.Sub(lastActive) > h.timeout {
			timeoutCount++
			h.logger.Debug("Connection heartbeat timeout",
				"conn_id", conn.ID(),
				"user_id", conn.UserID(),
				"last_active", lastActive,
				"timeout", h.timeout)

			// 调用超时回调
			if h.onTimeout != nil {
				h.onTimeout(conn)
			}

			// 关闭连接
			conn.Close()

			// 从 Manager 移除
			h.manager.Remove(conn.ID())
		}
	}

	if timeoutCount > 0 {
		h.logger.Info("Heartbeat check completed",
			"total", len(conns),
			"timeout", timeoutCount)
	}
}
