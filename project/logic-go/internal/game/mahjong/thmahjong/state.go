package thmahjong

import "sudooom.im.logic/internal/game/mahjong/core"

// THPlayerState 太湖麻将玩家状态
type THPlayerState struct {
	KongCount   int `json:"kongCount"`   // 杠的数量
	FlowerCount int `json:"flowerCount"` // 花牌数量
}

// Clone 克隆状态
func (s *THPlayerState) Clone() core.PlayerState {
	return &THPlayerState{
		KongCount:   s.KongCount,
		FlowerCount: s.FlowerCount,
	}
}
