package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/example/im-access/internal/config"
	"github.com/example/im-access/internal/nats"
	"github.com/example/im-access/internal/server"
	"go.uber.org/zap"
)

func main() {
	// 初始化日志
	logger, _ := zap.NewProduction()
	defer logger.Sync()

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
		logger.Fatal("Failed to connect to NATS", zap.Error(err))
	}
	defer natsClient.Close()

	// 创建并启动服务器
	srv := server.New(cfg, natsClient, logger)
	go func() {
		if err := srv.Start(ctx); err != nil {
			logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	logger.Info("Access server started",
		zap.String("addr", cfg.Server.Addr),
	)

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	cancel()
	srv.Shutdown()
	logger.Info("Server stopped")
}
