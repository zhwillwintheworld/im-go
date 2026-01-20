package mahjong

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
	"sudooom.im.logic/internal/game"
	"sudooom.im.logic/internal/game/mahjong/core"
	"sudooom.im.logic/internal/game/mahjong/htmajong"
	"sudooom.im.logic/internal/game/mahjong/thmahjong"
)

// GameType 麻将游戏类型
type GameType string

const (
	GameTypeHuiTong GameType = "huitong" // 会同麻将
	GameTypeTaiHu   GameType = "taihu"   // 太湖麻将
)

// MahjongService 麻将游戏服务
type MahjongService struct {
	redisClient *redis.Client
	gameManager *game.GameManager // 游戏管理器
	logger      *slog.Logger
}

// NewMahjongService 创建麻将游戏服务
func NewMahjongService(redisClient *redis.Client, gameManager *game.GameManager) *MahjongService {
	return &MahjongService{
		redisClient: redisClient,
		gameManager: gameManager,
		logger:      slog.Default(),
	}
}

// CreateEngine 创建麻将游戏引擎
func (s *MahjongService) CreateEngine(ctx context.Context, gameType GameType) (core.GameEngine, error) {
	switch gameType {
	case GameTypeHuiTong:
		s.logger.Info("创建会同麻将引擎")
		return htmajong.NewEngine(), nil
	case GameTypeTaiHu:
		s.logger.Info("创建太湖麻将引擎")
		return thmahjong.NewEngine(), nil
	default:
		return nil, fmt.Errorf("不支持的游戏类型: %s", gameType)
	}
}
