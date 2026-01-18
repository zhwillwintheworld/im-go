package thmahjong

import (
	"sudooom.im.logic/internal/game/mahjong/core"
)

// WinningAlgorithm 太湖麻将胡牌算法
type WinningAlgorithm struct{}

// NewWinningAlgorithm 创建胡牌算法
func NewWinningAlgorithm() *WinningAlgorithm {
	return &WinningAlgorithm{}
}

// CanWin 检查是否可以胡牌
func (w *WinningAlgorithm) CanWin(hand []core.Tile, newTile *core.Tile, state *core.GameState, playerID string) bool {
	allTiles := core.CloneTiles(hand)
	if newTile != nil {
		allTiles = append(allTiles, *newTile)
	}

	return w.checkBasicWin(allTiles) || w.checkSevenPairs(allTiles)
}

// GetWinPatterns 获取胡牌牌型
func (w *WinningAlgorithm) GetWinPatterns(hand []core.Tile, newTile *core.Tile, state *core.GameState, playerID string) []core.WinPattern {
	patterns := []core.WinPattern{}

	allTiles := core.CloneTiles(hand)
	if newTile != nil {
		allTiles = append(allTiles, *newTile)
	}

	player := state.GetPlayer(playerID)
	var flowerCount int
	if player != nil {
		if thState, ok := player.State.(*THPlayerState); ok {
			flowerCount = thState.FlowerCount
		}
	}

	// 检查无花果自摸
	if w.checkWuHuaGuoZiMo(allTiles, flowerCount, state.LastAction) {
		patterns = append(patterns, core.WinPattern{Name: "无花果自摸", Score: 10})
	}

	// 检查清一色
	if w.checkQingYiSe(allTiles) {
		patterns = append(patterns, core.WinPattern{Name: "清一色", Score: 10})
	}

	// 检查龙七对
	if w.checkLongQiDui(allTiles) {
		patterns = append(patterns, core.WinPattern{Name: "龙七对", Score: 8})
	} else if w.checkSevenPairs(allTiles) {
		patterns = append(patterns, core.WinPattern{Name: "七小对", Score: 6})
	}

	// 检查碰碰胡
	if w.checkPengPengHu(allTiles) {
		patterns = append(patterns, core.WinPattern{Name: "碰碰胡", Score: 6})
	}

	// 检查大吊车
	if w.checkDaDiaoChe(hand, newTile) {
		patterns = append(patterns, core.WinPattern{Name: "大吊车", Score: 5})
	}

	// 平胡
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

// checkBasicWin 检查基本胡牌
func (w *WinningAlgorithm) checkBasicWin(tiles []core.Tile) bool {
	if len(tiles) != 14 {
		return false
	}

	core.SortTiles(tiles)

	uniqueTiles := core.GetUniqueTiles(tiles)
	for _, jianPai := range uniqueTiles {
		if core.CountTile(tiles, jianPai) < 2 {
			continue
		}

		remaining := core.CloneTiles(tiles)
		remaining = core.RemoveTile(remaining, jianPai)
		remaining = core.RemoveTile(remaining, jianPai)

		if w.checkMianZi(remaining) {
			return true
		}
	}

	return false
}

// checkMianZi 检查是否都是面子
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

	// 尝试顺子
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

	tileCount := make(map[core.Tile]int)
	for _, t := range tiles {
		tileCount[t]++
	}

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

// checkLongQiDui 检查龙七对
func (w *WinningAlgorithm) checkLongQiDui(tiles []core.Tile) bool {
	if !w.checkSevenPairs(tiles) {
		return false
	}

	tileCount := make(map[core.Tile]int)
	for _, t := range tiles {
		tileCount[t]++
	}

	for _, count := range tileCount {
		if count == 4 {
			return true
		}
	}

	return false
}

// checkQingYiSe 检查清一色
func (w *WinningAlgorithm) checkQingYiSe(tiles []core.Tile) bool {
	if len(tiles) == 0 {
		return false
	}

	firstSuit := tiles[0].Suit
	// 清一色必须是万条筒
	if firstSuit >= core.TileSuitWind {
		return false
	}

	for _, t := range tiles {
		if t.Suit != firstSuit {
			return false
		}
	}

	return true
}

// checkPengPengHu 检查碰碰胡
func (w *WinningAlgorithm) checkPengPengHu(tiles []core.Tile) bool {
	if len(tiles) != 14 {
		return false
	}

	core.SortTiles(tiles)

	uniqueTiles := core.GetUniqueTiles(tiles)
	for _, jianPai := range uniqueTiles {
		if core.CountTile(tiles, jianPai) < 2 {
			continue
		}

		remaining := core.CloneTiles(tiles)
		remaining = core.RemoveTile(remaining, jianPai)
		remaining = core.RemoveTile(remaining, jianPai)

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

	tileCount := make(map[core.Tile]int)
	for _, t := range tiles {
		tileCount[t]++
	}

	for _, count := range tileCount {
		if count%3 != 0 {
			return false
		}
	}

	return true
}

// checkDaDiaoChe 检查大吊车 (单吊将牌)
func (w *WinningAlgorithm) checkDaDiaoChe(hand []core.Tile, newTile *core.Tile) bool {
	if newTile == nil || len(hand) != 13 {
		return false
	}

	// 检查手牌能否组成4个面子
	return w.checkMianZi(hand)
}

// checkWuHuaGuoZiMo 检查无花果自摸
func (w *WinningAlgorithm) checkWuHuaGuoZiMo(tiles []core.Tile, flowerCount int, lastAction *core.Action) bool {
	// 必须是自摸
	if lastAction != nil && lastAction.Type == core.ActionDiscard {
		return false
	}

	// 没有花牌
	return flowerCount == 0
}
