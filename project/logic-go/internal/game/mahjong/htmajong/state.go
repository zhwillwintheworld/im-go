package htmajong

import "sudooom.im.logic/internal/game/mahjong/core"

// HTPlayerState 会同麻将玩家状态
type HTPlayerState struct {
	IsTing       bool `json:"isTing"`       // 是否已报听
	CanTingRound int  `json:"canTingRound"` // 可以报听的轮次 (第一轮才可以报听)
}

// Clone 克隆状态
func (s *HTPlayerState) Clone() core.PlayerState {
	return &HTPlayerState{
		IsTing:       s.IsTing,
		CanTingRound: s.CanTingRound,
	}
}
