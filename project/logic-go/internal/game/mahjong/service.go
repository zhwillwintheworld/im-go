package mahjong

import (
	"context"
	"fmt"
	"log/slog"

	"sudooom.im.logic/internal/game/mahjong/core"
	"sudooom.im.logic/internal/game/mahjong/htmajong"
	"sudooom.im.logic/internal/game/mahjong/thmahjong"
	"sudooom.im.shared/model"
)

// GameType 麻将游戏类型
type GameType string

const (
	GameTypeHuiTong GameType = "huitong" // 会同麻将
	GameTypeTaiHu   GameType = "taihu"   // 太湖麻将
)

// GameManager 游戏管理器接口（避免循环依赖）
type GameManager interface {
	GetOrCreate(roomID string, gameType string) (interface{}, error)
}

// GameObject 游戏对象接口（避免循环依赖）
type GameObject interface {
	SetEngine(engine interface{})
	InitGame(ctx context.Context, playerIDs []string) error
}

// MahjongService 麻将游戏服务
// 注意：不使用 Redis，所有数据在内存中（通过 GameManager 管理）
type MahjongService struct {
	gameManager GameManager // 游戏管理器接口
	logger      *slog.Logger
}

// NewMahjongService 创建麻将游戏服务
func NewMahjongService(gameManager GameManager) *MahjongService {
	return &MahjongService{
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

// StartGameByType 实现 game.GameStarter 接口
// 由 GameService 在 StartGame 流程中调用
func (s *MahjongService) StartGameByType(ctx context.Context, room *model.Room, internalGameType string) error {
	return s.StartGame(ctx, room, GameType(internalGameType))
}
