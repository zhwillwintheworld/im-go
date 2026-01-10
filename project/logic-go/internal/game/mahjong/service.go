package mahjong

import (
	"log/slog"

	"github.com/redis/go-redis/v9"
)

// MahjongService 麻将游戏服务
type MahjongService struct {
	redisClient *redis.Client
	logger      *slog.Logger
}

// NewMahjongService 创建麻将游戏服务
func NewMahjongService(redisClient *redis.Client) *MahjongService {
	return &MahjongService{
		redisClient: redisClient,
		logger:      slog.Default(),
	}
}
