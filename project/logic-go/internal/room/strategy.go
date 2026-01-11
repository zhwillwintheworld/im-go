package room

import "sudooom.im.shared/model"

// GameTypeStrategy 游戏类型策略接口
type GameTypeStrategy interface {
	ValidatePlayers(r *model.Room) error
	GetRequiredPlayerCount() int
	AllocateSeat(r *model.Room) (int32, error)
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

func (s *MahjongGameStrategy) AllocateSeat(r *model.Room) (int32, error) {
	// 麻将座位：0=东, 1=南, 2=西, 3=北
	occupied := make(map[int32]bool)
	for _, p := range r.Players {
		if p.SeatIndex != -1 {
			occupied[p.SeatIndex] = true
		}
	}

	// 分配第一个空闲座位
	for i := int32(0); i < 4; i++ {
		if !occupied[i] {
			return i, nil
		}
	}

	return -1, ErrNoAvailableSeat
}
