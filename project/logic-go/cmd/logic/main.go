package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"sudooom.im.logic/internal/config"
	"sudooom.im.logic/internal/handler"
	imNats "sudooom.im.logic/internal/nats"
	"sudooom.im.logic/internal/service"
	"sudooom.im.shared/snowflake"
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
	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Error("Failed to close Redis client", "error", err)
		}
	}()
	logger.Info("Connected to Redis", "host", cfg.Redis.Host)

	// 连接数据库
	db, err := connectDatabase(ctx, cfg.Database)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("Connected to PostgreSQL", "host", cfg.Database.Host)

	// 初始化雪花ID生成器
	sfNode, err := snowflake.NewNode(1)
	if err != nil {
		logger.Error("Failed to create snowflake node", "error", err)
		os.Exit(1)
	}

	// 初始化服务
	publisher := imNats.NewMessagePublisher(natsClient.Conn())
	routerService := service.NewRouterService(redisClient, publisher)
	groupService := service.NewGroupService(db)
	messageService := service.NewMessageService(db)

	// 创建消息批量写入器
	messageBatcher := service.NewMessageBatcher(db, sfNode, service.MessageBatcherConfig{
		BatchSize:     cfg.Batch.Size,
		FlushInterval: cfg.Batch.FlushInterval,
	})
	messageBatcher.Start(ctx)

	// 创建会话服务
	conversationService := service.NewConversationService(redisClient)

	// 创建消息处理器
	msgHandler := handler.NewMessageHandler(
		messageBatcher,
		messageService,
		groupService,
		routerService,
		conversationService,
		redisClient,
	)

	// 启动订阅者
	subscriber := imNats.NewMessageSubscriber(natsClient.Conn(), msgHandler, imNats.SubscriberConfig{
		WorkerCount: cfg.NATS.WorkerCount,
		BufferSize:  cfg.NATS.BufferSize,
	})
	if err := subscriber.Start(ctx); err != nil {
		logger.Error("Failed to start subscriber", "error", err)
		os.Exit(1)
	}

	logger.Info("Logic service started", "name", cfg.App.Name)

	// 优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down...")
	cancel()
	if err := subscriber.Stop(); err != nil {
		logger.Error("Failed to stop subscriber", "error", err)
	}
	messageBatcher.Stop()
	logger.Info("Logic service stopped")
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
