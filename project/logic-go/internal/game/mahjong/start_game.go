package mahjong

import (
	"context"
	"fmt"

	"sudooom.im.shared/model"
)

// StartGame 初始化麻将游戏
func (s *MahjongService) StartGame(ctx context.Context, room *model.Room, gameType GameType) error {
	s.logger.Info("初始化麻将游戏",
		"roomId", room.RoomID,
		"gameType", gameType,
		"playerCount", len(room.Players))

	// 1. 从 GameManager 获取或创建 Game 对象
	gameObj := s.gameManager.GetOrCreate(room.RoomID, string(gameType))

	// 2. 创建 mahjong engine
	mahjongEngine, err := s.CreateEngine(ctx, gameType)
	if err != nil {
		return fmt.Errorf("创建游戏引擎失败: %w", err)
	}

	// 3. 包装为线程安全的 engine
	safeEngine := NewSafeMahjongEngine(mahjongEngine, string(gameType))

	// 4. 存储到 Game 对象
	gameObj.SetEngine(safeEngine)

	// 5. 初始化游戏
	playerIDs := make([]int64, len(room.Players))
	for i, player := range room.Players {
		playerIDs[i] = player.UserID
	}

	if err := gameObj.InitMahjongGame(ctx, playerIDs); err != nil {
		return fmt.Errorf("初始化游戏失败: %w", err)
	}

	s.logger.Info("麻将游戏初始化成功",
		"roomId", room.RoomID,
		"gameType", gameType)

	return nil
}
