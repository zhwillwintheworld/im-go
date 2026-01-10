package mahjong

import (
	"context"

	"sudooom.im.shared/model"
)

// StartGame 初始化麻将游戏
func (s *MahjongService) StartGame(ctx context.Context, room *model.Room) error {
	s.logger.Info("Initializing mahjong game",
		"roomId", room.RoomID,
		"playerCount", len(room.Players))

	// TODO: 实现麻将游戏初始化逻辑
	// 1. 初始化牌堆（136张牌）
	// 2. 洗牌
	// 3. 发牌（每人13张，庄家14张）
	// 4. 设置游戏状态（当前玩家、剩余牌数等）
	// 5. 保存游戏状态到 Redis

	s.logger.Info("Mahjong game initialized", "roomId", room.RoomID)
	return nil
}
