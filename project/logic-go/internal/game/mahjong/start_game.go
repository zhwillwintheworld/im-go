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
	gameObjInterface, err := s.gameManager.GetOrCreate(room.RoomID, string(gameType))
	if err != nil {
		return fmt.Errorf("获取游戏对象失败: %w", err)
	}

	// 类型断言为 GameObject
	gameObj, ok := gameObjInterface.(GameObject)
	if !ok {
		return fmt.Errorf("游戏对象类型错误")
	}

	// 2. 创建 mahjong engine
	mahjongEngine, err := s.CreateEngine(ctx, gameType)
	if err != nil {
		return fmt.Errorf("创建游戏引擎失败: %w", err)
	}

	// 3. 创建适配器（不包含锁，依赖 Round.opMu 保证并发安全）
	engineAdapter := NewMahjongEngineAdapter(mahjongEngine, string(gameType))

	// 4. 存储到 Game 对象（MahjongEngineAdapter 实现了 game.RoundEngine 接口）
	gameObj.SetEngine(engineAdapter)

	// 5. 初始化游戏
	playerIDs := make([]string, len(room.Players))
	for i, player := range room.Players {
		playerIDs[i] = fmt.Sprintf("%d", player.UserID)
	}

	if err := gameObj.InitGame(ctx, playerIDs); err != nil {
		return fmt.Errorf("初始化游戏失败: %w", err)
	}

	s.logger.Info("麻将游戏初始化成功",
		"roomId", room.RoomID,
		"gameType", gameType)

	return nil
}
