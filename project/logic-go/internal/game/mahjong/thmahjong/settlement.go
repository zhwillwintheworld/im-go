package thmahjong

import (
	"sudooom.im.logic/internal/game/mahjong/core"
)

// Settler 太湖麻将结算器
type Settler struct{}

// NewSettler 创建结算器
func NewSettler() *Settler {
	return &Settler{}
}

// Calculate 计算结算结果
// 总花数 = 牌型花数 + 花牌数 + 胡牌类型花数 + 杠数量花数
// 总分 = (总花数 / 10) * 底分
func (s *Settler) Calculate(state *core.GameState, winnerID string, loserID string, winType core.WinType, patterns []core.WinPattern) *core.Settlement {
	// 计算牌型花数
	patternFlowers := 0
	for _, p := range patterns {
		patternFlowers += p.Score
	}

	// 获取花牌数和杠数
	winner := state.GetPlayer(winnerID)
	var flowerCount, kongCount int
	if winner != nil {
		if thState, ok := winner.State.(*THPlayerState); ok {
			flowerCount = thState.FlowerCount
			kongCount = thState.KongCount
		}
	}

	// 胡牌类型花数
	winTypeFlowers := 0
	if winType == core.WinTypeDraw {
		winTypeFlowers = 2 // 自摸加2花
	}

	// 杠的花数
	kongFlowers := kongCount * 2 // 每个杠加2花

	// 总花数
	totalFlowers := patternFlowers + flowerCount + winTypeFlowers + kongFlowers

	// 获取底分
	baseScore := state.Config.BaseScore
	if baseScore == 0 {
		baseScore = 1
	}

	// 计算总分: (总花数 / 10) * 底分
	// 至少1倍
	multiplier := totalFlowers / 10
	if multiplier == 0 {
		multiplier = 1
	}
	totalScore := multiplier * baseScore

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
