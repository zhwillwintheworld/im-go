package mahjong

import (
	"context"
	"fmt"

	"sudooom.im.logic/internal/game/mahjong/core"
	"sudooom.im.shared/model"
)

// StartGame 初始化麻将游戏
func (s *MahjongService) StartGame(ctx context.Context, room *model.Room, gameType GameType) error {
	s.logger.Info("初始化麻将游戏",
		"roomId", room.RoomID,
		"gameType", gameType,
		"playerCount", len(room.Players))

	// 创建游戏引擎
	engine, err := s.CreateEngine(ctx, gameType)
	if err != nil {
		return fmt.Errorf("创建游戏引擎失败: %w", err)
	}

	// 提取玩家ID列表
	playerIDs := make([]string, len(room.Players))
	for i, player := range room.Players {
		playerIDs[i] = fmt.Sprintf("%d", player.UserID) // 将 int64 转换为 string
	}

	// 创建游戏配置
	config := core.GameConfig{
		PlayerCount: len(playerIDs),
		BaseScore:   10, // 默认底分10
		Extra:       make(map[string]any),
	}

	// 初始化游戏
	if err := engine.Initialize(ctx, playerIDs, config); err != nil {
		return fmt.Errorf("初始化游戏失败: %w", err)
	}

	// TODO: 保存游戏状态到 Redis
	// 可以使用 engine.GetState() 获取游戏状态并序列化到 Redis

	s.logger.Info("麻将游戏初始化成功",
		"roomId", room.RoomID,
		"gameType", gameType)

	return nil
}
