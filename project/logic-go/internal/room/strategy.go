package room

import "sudooom.im.shared/model"

// GameTypeStrategy 游戏类型策略接口
type GameTypeStrategy interface {
	ValidatePlayers(r *model.Room) error
	GetRequiredPlayerCount() int
}

// MahjongGameStrategy 麻将游戏策略
type MahjongGameStrategy struct{}

func (s *MahjongGameStrategy) ValidatePlayers(r *model.Room) error {
	// 麻将需要4个玩家
	requiredCount := 4

	// 统计已准备的玩家数量
	readyCount := 0
	for _, player := range r.Players {
		if player.IsReady {
			readyCount++
		}
	}

	// 检查玩家数量
	if len(r.Players) < requiredCount {
		return ErrNotEnoughPlayers
	}

	// 检查是否所有玩家都已准备
	if readyCount < requiredCount {
		return ErrNotAllReady
	}

	return nil
}

func (s *MahjongGameStrategy) GetRequiredPlayerCount() int {
	return 4
}
