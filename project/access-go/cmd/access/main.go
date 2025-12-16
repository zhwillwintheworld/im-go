package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/redis/go-redis/v9"
	"sudooom.im.access/internal/config"
	"sudooom.im.access/internal/health"
	"sudooom.im.access/internal/nats"
	imRedis "sudooom.im.access/internal/redis"
	"sudooom.im.access/internal/server"
)

func main() {
	// 初始化日志
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	// 加载配置
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 初始化 NATS 客户端
	natsClient, err := nats.NewClient(cfg.NATS)
	if err != nil {
		logger.Error("Failed to connect to NATS", "error", err)
		os.Exit(1)
	}
	defer natsClient.Close()
	logger.Info("Connected to NATS", "url", cfg.NATS.URL)

	// 初始化 Redis 客户端
	redisClient := imRedis.NewClient(cfg.Redis, cfg.Server.NodeID)
	defer redisClient.Close()
	logger.Info("Connected to Redis", "addr", cfg.Redis.Addr)

	// 创建并启动服务器
	srv := server.New(cfg, natsClient, redisClient, logger)
	go func() {
		if err := srv.Start(ctx); err != nil {
			logger.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	// 启动健康检查 HTTP 服务
	go startHealthServer(natsClient, cfg.Redis, srv.ConnManager(), logger)

	logger.Info("Access server started",
		"addr", cfg.Server.Addr,
		"node_id", cfg.Server.NodeID)

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	cancel()
	srv.Shutdown()
	logger.Info("Server stopped")
}

// startHealthServer 启动健康检查 HTTP 服务
func startHealthServer(natsClient *nats.Client, redisCfg config.RedisConfig, connCounter health.ConnectionCounter, logger *slog.Logger) {
	// 创建独立的 Redis 客户端用于健康检查
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisCfg.Addr,
		Password: redisCfg.Password,
		DB:       redisCfg.DB,
	})
	defer redisClient.Close()

	healthChecker := health.NewChecker(natsClient.Conn(), redisClient, connCounter)

	mux := http.NewServeMux()
	mux.Handle("/health", healthChecker)
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if healthChecker.IsHealthy(r.Context()) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Not Ready"))
		}
	})

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	logger.Info("Health check server started", "addr", httpServer.Addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("Health check server failed", "error", err)
	}
}
