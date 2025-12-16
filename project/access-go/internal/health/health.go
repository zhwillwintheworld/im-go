package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
)

// Status 健康状态
type Status struct {
	Service     string `json:"service"`
	NATS        string `json:"nats"`
	Redis       string `json:"redis"`
	Connections int    `json:"connections"`
}

// ConnectionCounter 连接计数器接口
type ConnectionCounter interface {
	Count() int
}

// Checker 健康检查器
type Checker struct {
	nc          *nats.Conn
	redisClient *redis.Client
	connCounter ConnectionCounter
}

// NewChecker 创建健康检查器
func NewChecker(nc *nats.Conn, redisClient *redis.Client, connCounter ConnectionCounter) *Checker {
	return &Checker{
		nc:          nc,
		redisClient: redisClient,
		connCounter: connCounter,
	}
}

// Check 执行健康检查
func (h *Checker) Check(ctx context.Context) *Status {
	status := &Status{
		Service: "access",
	}

	// 检查 NATS
	if h.nc != nil && h.nc.IsConnected() {
		status.NATS = "connected"
	} else {
		status.NATS = "disconnected"
	}

	// 检查 Redis
	if h.redisClient != nil {
		redisCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		if err := h.redisClient.Ping(redisCtx).Err(); err == nil {
			status.Redis = "connected"
		} else {
			status.Redis = "disconnected"
		}
	} else {
		status.Redis = "not configured"
	}

	// 连接数
	if h.connCounter != nil {
		status.Connections = h.connCounter.Count()
	}

	return status
}

// IsHealthy 检查是否健康
func (h *Checker) IsHealthy(ctx context.Context) bool {
	status := h.Check(ctx)
	return status.NATS == "connected"
}

// ServeHTTP HTTP 健康检查端点
func (h *Checker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	status := h.Check(r.Context())

	if status.NATS != "connected" {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
