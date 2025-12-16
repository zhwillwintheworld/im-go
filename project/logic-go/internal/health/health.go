package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
)

// Status 健康状态
type Status struct {
	NATS     string `json:"nats"`
	Redis    string `json:"redis"`
	Database string `json:"database"`
}

// Checker 健康检查器
type Checker struct {
	nc          *nats.Conn
	redisClient *redis.Client
	db          *pgxpool.Pool
}

// NewChecker 创建健康检查器
func NewChecker(nc *nats.Conn, redisClient *redis.Client, db *pgxpool.Pool) *Checker {
	return &Checker{
		nc:          nc,
		redisClient: redisClient,
		db:          db,
	}
}

// Check 执行健康检查
func (h *Checker) Check(ctx context.Context) *Status {
	status := &Status{}

	// 检查 NATS
	if h.nc.IsConnected() {
		status.NATS = "connected"
	} else {
		status.NATS = "disconnected"
	}

	// 检查 Redis
	redisCtx, redisCancel := context.WithTimeout(ctx, 2*time.Second)
	defer redisCancel()

	if err := h.redisClient.Ping(redisCtx).Err(); err == nil {
		status.Redis = "connected"
	} else {
		status.Redis = "disconnected"
	}

	// 检查 PostgreSQL
	dbCtx, dbCancel := context.WithTimeout(ctx, 2*time.Second)
	defer dbCancel()

	if err := h.db.Ping(dbCtx); err == nil {
		status.Database = "connected"
	} else {
		status.Database = "disconnected"
	}

	return status
}

// IsHealthy 检查是否健康
func (h *Checker) IsHealthy(ctx context.Context) bool {
	status := h.Check(ctx)
	return status.NATS == "connected" &&
		status.Redis == "connected" &&
		status.Database == "connected"
}

// ServeHTTP HTTP 健康检查端点
func (h *Checker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	status := h.Check(r.Context())

	if status.NATS != "connected" || status.Redis != "connected" || status.Database != "connected" {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
