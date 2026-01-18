package htmajong

import (
	"sudooom.im.logic/internal/game/mahjong/core"
)

// Settler 会同麻将结算器
type Settler struct{}

// NewSettler 创建结算器
func NewSettler() *Settler {
	return &Settler{}
}

// Calculate 计算结算结果
func (s *Settler) Calculate(state *core.GameState, winnerID string, loserID string, winType core.WinType, patterns []core.WinPattern) *core.Settlement {
	// 计算番数
	totalFan := 0
	for _, p := range patterns {
		totalFan += p.Score
	}

	// 获取底分
	baseScore := state.Config.BaseScore
	if baseScore == 0 {
		baseScore = 1 // 默认底分为1
	}

	// 计算总分
	totalScore := baseScore * totalFan

	// 自摸加倍
	if winType == core.WinTypeDraw {
		totalScore *= 2
	}

	// 报听胡加倍
	if s.hasTingPattern(patterns) {
		totalScore *= 2
	}

	// 抢杠胡加倍
	if winType == core.WinTypeQiangKong {
		totalScore *= 2
	}

	// 检查是否烧庄 (连庄)
	winner := state.GetPlayer(winnerID)
	if winner != nil && state.GetPlayerIndex(winnerID) == state.DealerIndex {
		totalScore *= 2 // 烧庄加倍
		patterns = append(patterns, core.WinPattern{Name: "烧庄", Score: 0})
	}

	// 创建分数转移记录
	transfers := []core.Transfer{}

	if winType == core.WinTypeDraw {
		// 自摸: 每个玩家都输
		for _, player := range state.Players {
			if player.ID != winnerID {
				transfers = append(transfers, core.Transfer{
					FromID: player.ID,
					ToID:   winnerID,
					Amount: totalScore,
					Reason: "自摸",
				})
			}
		}
	} else {
		// 点炮: 只有输家输分
		if loserID != "" {
			transfers = append(transfers, core.Transfer{
				FromID: loserID,
				ToID:   winnerID,
				Amount: totalScore,
				Reason: winType.String(),
			})
		}
	}

	return &core.Settlement{
		WinnerID:   winnerID,
		LoserID:    loserID,
		WinType:    winType,
		Patterns:   patterns,
		BaseScore:  baseScore,
		TotalScore: totalScore,
		Transfers:  transfers,
	}
}

// hasTingPattern 检查是否有报听胡牌型
func (s *Settler) hasTingPattern(patterns []core.WinPattern) bool {
	for _, p := range patterns {
		if p.Name == "报听胡" {
			return true
		}
	}
	return false
}

// GetHuType 获取胡牌类型 (小胡、大胡、大大胡)
func (s *Settler) GetHuType(totalFan int) string {
	if totalFan >= 10 {
		return "大大胡"
	} else if totalFan >= 6 {
		return "大胡"
	}
	return "小胡"
}
