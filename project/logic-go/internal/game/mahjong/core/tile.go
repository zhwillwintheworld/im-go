package core

import "sort"

// SortTiles 对牌进行排序
func SortTiles(tiles []Tile) {
	sort.Slice(tiles, func(i, j int) bool {
		if tiles[i].Suit != tiles[j].Suit {
			return tiles[i].Suit < tiles[j].Suit
		}
		return tiles[i].Value < tiles[j].Value
	})
}

// CountTile 统计某张牌的数量
func CountTile(tiles []Tile, target Tile) int {
	count := 0
	for _, t := range tiles {
		if t.Equal(target) {
			count++
		}
	}
	return count
}

// RemoveTile 从牌组中移除一张牌
func RemoveTile(tiles []Tile, target Tile) []Tile {
	for i, t := range tiles {
		if t.Equal(target) {
			return append(tiles[:i], tiles[i+1:]...)
		}
	}
	return tiles
}

// RemoveTiles 从牌组中移除多张牌
func RemoveTiles(tiles []Tile, targets []Tile) []Tile {
	result := make([]Tile, len(tiles))
	copy(result, tiles)

	for _, target := range targets {
		result = RemoveTile(result, target)
	}
	return result
}

// ContainsTile 检查牌组是否包含某张牌
func ContainsTile(tiles []Tile, target Tile) bool {
	for _, t := range tiles {
		if t.Equal(target) {
			return true
		}
	}
	return false
}

// ContainsTiles 检查牌组是否包含多张牌
func ContainsTiles(tiles []Tile, targets []Tile) bool {
	tileCopy := make([]Tile, len(tiles))
	copy(tileCopy, tiles)

	for _, target := range targets {
		if !ContainsTile(tileCopy, target) {
			return false
		}
		tileCopy = RemoveTile(tileCopy, target)
	}
	return true
}

// CloneTiles 克隆牌组
func CloneTiles(tiles []Tile) []Tile {
	result := make([]Tile, len(tiles))
	copy(result, tiles)
	return result
}

// GroupBySuit 按花色分组
func GroupBySuit(tiles []Tile) map[TileSuit][]Tile {
	groups := make(map[TileSuit][]Tile)
	for _, t := range tiles {
		groups[t.Suit] = append(groups[t.Suit], t)
	}
	return groups
}

// IsSequence 检查是否为顺子 (3张连续的牌)
func IsSequence(tiles []Tile) bool {
	if len(tiles) != 3 {
		return false
	}

	// 必须是同花色
	if tiles[0].Suit != tiles[1].Suit || tiles[1].Suit != tiles[2].Suit {
		return false
	}

	// 风、箭牌、花牌不能组成顺子
	if tiles[0].Suit >= TileSuitWind {
		return false
	}

	sorted := CloneTiles(tiles)
	SortTiles(sorted)

	// 检查是否连续
	return sorted[1].Value == sorted[0].Value+1 && sorted[2].Value == sorted[1].Value+1
}

// IsTriplet 检查是否为刻子 (3张相同的牌)
func IsTriplet(tiles []Tile) bool {
	if len(tiles) != 3 {
		return false
	}
	return tiles[0].Equal(tiles[1]) && tiles[1].Equal(tiles[2])
}

// IsQuad 检查是否为杠 (4张相同的牌)
func IsQuad(tiles []Tile) bool {
	if len(tiles) != 4 {
		return false
	}
	return tiles[0].Equal(tiles[1]) && tiles[1].Equal(tiles[2]) && tiles[2].Equal(tiles[3])
}

// IsPair 检查是否为对子 (2张相同的牌)
func IsPair(tiles []Tile) bool {
	if len(tiles) != 2 {
		return false
	}
	return tiles[0].Equal(tiles[1])
}

// GetUniqueTiles 获取去重后的牌
func GetUniqueTiles(tiles []Tile) []Tile {
	seen := make(map[Tile]bool)
	result := []Tile{}

	for _, t := range tiles {
		if !seen[t] {
			seen[t] = true
			result = append(result, t)
		}
	}
	return result
}

// CanFormMeld 检查是否可以组成明牌组合
func CanFormMeld(tiles []Tile, meldType MeldType) bool {
	switch meldType {
	case MeldTypePong:
		return IsTriplet(tiles)
	case MeldTypeKong:
		return IsQuad(tiles)
	case MeldTypeChi:
		return IsSequence(tiles)
	default:
		return false
	}
}
