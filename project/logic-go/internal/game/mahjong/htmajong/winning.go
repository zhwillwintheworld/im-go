package htmajong

import (
	"sudooom.im.logic/internal/game/mahjong/core"
)

// WinningAlgorithm 会同麻将胡牌算法
type WinningAlgorithm struct{}

// NewWinningAlgorithm 创建胡牌算法
func NewWinningAlgorithm() *WinningAlgorithm {
	return &WinningAlgorithm{}
}

// CanWin 检查是否可以胡牌
func (w *WinningAlgorithm) CanWin(hand []core.Tile, newTile *core.Tile, state *core.GameState, playerID string) bool {
	// 组合手牌
	allTiles := core.CloneTiles(hand)
	if newTile != nil {
		allTiles = append(allTiles, *newTile)
	}

	// 检查是否满足基本胡牌条件
	return w.checkBasicWin(allTiles) || w.checkSevenPairs(allTiles)
}

// GetWinPatterns 获取胡牌牌型
func (w *WinningAlgorithm) GetWinPatterns(hand []core.Tile, newTile *core.Tile, state *core.GameState, playerID string) []core.WinPattern {
	patterns := []core.WinPattern{}

	// 组合手牌
	allTiles := core.CloneTiles(hand)
	if newTile != nil {
		allTiles = append(allTiles, *newTile)
	}

	// 检查各种牌型
	if w.checkQingYiSe(allTiles) {
		patterns = append(patterns, core.WinPattern{Name: "清一色", Score: 10})
	}

	if w.checkLongQiDui(allTiles) {
		patterns = append(patterns, core.WinPattern{Name: "龙七对", Score: 8})
	} else if w.checkSevenPairs(allTiles) {
		patterns = append(patterns, core.WinPattern{Name: "七小对", Score: 6})
	}

	if w.checkJiangJiangHu(allTiles) {
		patterns = append(patterns, core.WinPattern{Name: "将将胡", Score: 6})
	}

	if w.checkQueYiMen(allTiles) {
		patterns = append(patterns, core.WinPattern{Name: "缺一门", Score: 2})
	}

	// 检查报听胡
	player := state.GetPlayer(playerID)
	if player != nil {
		htState, ok := player.State.(*HTPlayerState)
		if ok && htState.IsTing && htState.CanTingRound == 1 {
			patterns = append(patterns, core.WinPattern{Name: "报听胡", Score: 5})
		}
	}

	// 如果没有特殊牌型,则为平胡
	if len(patterns) == 0 {
		patterns = append(patterns, core.WinPattern{Name: "平胡", Score: 1})
	}

	return patterns
}

// CalculateScore 计算番数
func (w *WinningAlgorithm) CalculateScore(patterns []core.WinPattern) int {
	totalScore := 0
	for _, p := range patterns {
		totalScore += p.Score
	}
	return totalScore
}

// checkBasicWin 检查基本胡牌 (3*n + 2的组合)
func (w *WinningAlgorithm) checkBasicWin(tiles []core.Tile) bool {
	if len(tiles) != 14 {
		return false
	}

	core.SortTiles(tiles)

	// 尝试每张牌作为将牌
	uniqueTiles := core.GetUniqueTiles(tiles)
	for _, jianPai := range uniqueTiles {
		// 至少要有2张相同的牌才能作为将牌
		if core.CountTile(tiles, jianPai) < 2 {
			continue
		}

		// 移除将牌
		remaining := core.CloneTiles(tiles)
		remaining = core.RemoveTile(remaining, jianPai)
		remaining = core.RemoveTile(remaining, jianPai)

		// 检查剩余的牌是否能组成面子
		if w.checkMianZi(remaining) {
			return true
		}
	}

	return false
}

// checkMianZi 检查是否都是面子 (刻子或顺子)
func (w *WinningAlgorithm) checkMianZi(tiles []core.Tile) bool {
	if len(tiles) == 0 {
		return true
	}

	if len(tiles)%3 != 0 {
		return false
	}

	core.SortTiles(tiles)
	firstTile := tiles[0]

	// 尝试刻子
	if core.CountTile(tiles, firstTile) >= 3 {
		remaining := core.CloneTiles(tiles)
		for i := 0; i < 3; i++ {
			remaining = core.RemoveTile(remaining, firstTile)
		}
		if w.checkMianZi(remaining) {
			return true
		}
	}

	// 尝试顺子 (只有万条筒可以组成顺子)
	if firstTile.Suit < core.TileSuitWind && firstTile.Value <= 7 {
		tile2 := core.Tile{Suit: firstTile.Suit, Value: firstTile.Value + 1}
		tile3 := core.Tile{Suit: firstTile.Suit, Value: firstTile.Value + 2}

		if core.ContainsTile(tiles, tile2) && core.ContainsTile(tiles, tile3) {
			remaining := core.CloneTiles(tiles)
			remaining = core.RemoveTile(remaining, firstTile)
			remaining = core.RemoveTile(remaining, tile2)
			remaining = core.RemoveTile(remaining, tile3)
			if w.checkMianZi(remaining) {
				return true
			}
		}
	}

	return false
}

// checkSevenPairs 检查七小对
func (w *WinningAlgorithm) checkSevenPairs(tiles []core.Tile) bool {
	if len(tiles) != 14 {
		return false
	}

	// 统计每张牌的数量
	tileCount := make(map[core.Tile]int)
	for _, t := range tiles {
		tileCount[t]++
	}

	// 必须有7对
	pairCount := 0
	for _, count := range tileCount {
		if count == 2 {
			pairCount++
		} else if count != 4 && count != 0 {
			return false
		}
	}

	return pairCount >= 7 || len(tileCount) == 7
}

// checkLongQiDui 检查龙七对 (七对中有4张相同的牌)
func (w *WinningAlgorithm) checkLongQiDui(tiles []core.Tile) bool {
	if !w.checkSevenPairs(tiles) {
		return false
	}

	// 统计每张牌的数量
	tileCount := make(map[core.Tile]int)
	for _, t := range tiles {
		tileCount[t]++
	}

	// 检查是否有4张相同的牌
	for _, count := range tileCount {
		if count == 4 {
			return true
		}
	}

	return false
}

// checkQingYiSe 检查清一色 (只有一种花色)
func (w *WinningAlgorithm) checkQingYiSe(tiles []core.Tile) bool {
	if len(tiles) == 0 {
		return false
	}

	firstSuit := tiles[0].Suit
	for _, t := range tiles {
		if t.Suit != firstSuit {
			return false
		}
	}

	return true
}

// checkJiangJiangHu 检查将将胡 (全部是刻子+将)
func (w *WinningAlgorithm) checkJiangJiangHu(tiles []core.Tile) bool {
	if len(tiles) != 14 {
		return false
	}

	core.SortTiles(tiles)

	// 尝试每张牌作为将牌
	uniqueTiles := core.GetUniqueTiles(tiles)
	for _, jianPai := range uniqueTiles {
		if core.CountTile(tiles, jianPai) < 2 {
			continue
		}

		// 移除将牌
		remaining := core.CloneTiles(tiles)
		remaining = core.RemoveTile(remaining, jianPai)
		remaining = core.RemoveTile(remaining, jianPai)

		// 检查剩余的牌是否都是刻子
		if w.checkAllTriplets(remaining) {
			return true
		}
	}

	return false
}

// checkAllTriplets 检查是否都是刻子
func (w *WinningAlgorithm) checkAllTriplets(tiles []core.Tile) bool {
	if len(tiles) == 0 {
		return true
	}

	if len(tiles)%3 != 0 {
		return false
	}

	// 统计每张牌的数量
	tileCount := make(map[core.Tile]int)
	for _, t := range tiles {
		tileCount[t]++
	}

	// 每张牌必须是3的倍数
	for _, count := range tileCount {
		if count%3 != 0 {
			return false
		}
	}

	return true
}

// checkQueYiMen 检查缺一门 (缺少一种花色)
func (w *WinningAlgorithm) checkQueYiMen(tiles []core.Tile) bool {
	suits := make(map[core.TileSuit]bool)
	for _, t := range tiles {
		// 只统计万条筒
		if t.Suit <= core.TileSuitTong {
			suits[t.Suit] = true
		}
	}

	// 只有2种花色说明缺一门
	return len(suits) == 2
}
