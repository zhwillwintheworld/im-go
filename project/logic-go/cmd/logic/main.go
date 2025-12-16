package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"sudooom.im.logic/internal/config"
	"sudooom.im.logic/internal/handler"
	"sudooom.im.logic/internal/health"
	imNats "sudooom.im.logic/internal/nats"
	"sudooom.im.logic/internal/service"
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
		logger.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 连接 NATS
	natsClient, err := imNats.NewClient(cfg.NATS)
	if err != nil {
		logger.Error("Failed to connect to NATS", "error", err)
		os.Exit(1)
	}
	defer natsClient.Close()
	logger.Info("Connected to NATS", "url", cfg.NATS.URL)

	// 连接 Redis
	redisClient := connectRedis(cfg.Redis)
	defer redisClient.Close()
	logger.Info("Connected to Redis", "host", cfg.Redis.Host)

	// 连接数据库
	db, err := connectDatabase(ctx, cfg.Database)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("Connected to PostgreSQL", "host", cfg.Database.Host)

	// 初始化服务
	publisher := imNats.NewMessagePublisher(natsClient.Conn())
	routerService := service.NewRouterService(redisClient, publisher)
	userService := service.NewUserService(redisClient)
	groupService := service.NewGroupService(db)
	messageService := service.NewMessageService(db)

	// 创建消息处理器
	msgHandler := handler.NewMessageHandler(
		messageService,
		userService,
		groupService,
		routerService,
	)

	// 启动订阅者
	subscriber := imNats.NewMessageSubscriber(natsClient.Conn(), msgHandler)
	if err := subscriber.Start(ctx); err != nil {
		logger.Error("Failed to start subscriber", "error", err)
		os.Exit(1)
	}

	// 启动健康检查 HTTP 服务
	healthChecker := health.NewChecker(natsClient.Conn(), redisClient, db)
	go startHealthServer(healthChecker, logger)

	logger.Info("Logic service started", "name", cfg.App.Name)

	// 优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down...")
	cancel()
	subscriber.Stop()
	logger.Info("Logic service stopped")
}

// startHealthServer 启动健康检查 HTTP 服务
func startHealthServer(healthChecker *health.Checker, logger *slog.Logger) {
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

	server := &http.Server{
		Addr:    ":8081",
		Handler: mux,
	}

	logger.Info("Health check server started", "addr", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("Health check server failed", "error", err)
	}
}

// connectRedis 连接 Redis
func connectRedis(cfg config.RedisConfig) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})
}

// connectDatabase 连接 PostgreSQL
func connectDatabase(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Name,
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.MaxIdleConns)
	poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime
	poolConfig.MaxConnIdleTime = 10 * time.Minute

	return pgxpool.NewWithConfig(ctx, poolConfig)
}
