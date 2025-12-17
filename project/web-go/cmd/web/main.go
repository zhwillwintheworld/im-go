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

	"sudooom.im.shared/jwt"
	"sudooom.im.shared/snowflake"
	"sudooom.im.web/internal/config"
	"sudooom.im.web/internal/handler"
	"sudooom.im.web/internal/repository"
	"sudooom.im.web/internal/router"
	"sudooom.im.web/internal/service"
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

	// 连接数据库
	db, err := connectDatabase(ctx, cfg.Database)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("Connected to PostgreSQL", "host", cfg.Database.Host)

	// 连接 Redis
	redisClient := connectRedis(cfg.Redis)
	defer redisClient.Close()
	logger.Info("Connected to Redis", "host", cfg.Redis.Host)

	// 初始化 JWT 服务
	jwtService := jwt.NewService(
		cfg.JWT.SecretKey,
		cfg.JWT.AccessExpire,
		cfg.JWT.RefreshExpire,
	)

	// 初始化雪花ID生成器
	sfNode, err := snowflake.NewNode(1)
	if err != nil {
		logger.Error("Failed to create snowflake node", "error", err)
		os.Exit(1)
	}

	// 初始化 Repository
	userRepo := repository.NewUserRepository(db)
	friendRepo := repository.NewFriendRepository(db)

	// 初始化 Service
	authService := service.NewAuthService(userRepo, jwtService, sfNode)
	userService := service.NewUserService(userRepo)
	friendService := service.NewFriendService(friendRepo, userRepo, sfNode)

	// 初始化 Handler
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	friendHandler := handler.NewFriendHandler(friendService)

	// 设置路由
	r := router.SetupRouter(cfg, jwtService, authHandler, userHandler, friendHandler)

	// 启动服务器
	addr := fmt.Sprintf(":%d", cfg.App.Port)
	go func() {
		logger.Info("Web server started", "addr", addr, "mode", cfg.App.Mode)
		if err := r.Run(addr); err != nil {
			logger.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	// 优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	cancel()
	logger.Info("Server stopped")
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

// connectRedis 连接 Redis
func connectRedis(cfg config.RedisConfig) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})
}
