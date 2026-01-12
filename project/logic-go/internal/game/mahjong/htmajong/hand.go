package htmajong

import "sort"

// Hand 手牌管理器
type Hand struct {
	tiles []Mahjong // 手牌列表
}

// NewHand 创建新的手牌管理器
func NewHand() *Hand {
	return &Hand{
		tiles: make([]Mahjong, 0, StandardHandSize+1),
	}
}

// Add 添加牌到手牌
func (h *Hand) Add(tile Mahjong) error {
	h.tiles = append(h.tiles, tile)
	return nil
}

// Remove 移除指定的牌
func (h *Hand) Remove(tile Mahjong) error {
	for i, t := range h.tiles {
		if t.Equals(tile) {
			h.tiles = append(h.tiles[:i], h.tiles[i+1:]...)
			return nil
		}
	}
	return ErrTileNotInHand.WithContext("tile", tile.Number)
}

// RemoveByNumber 根据数字移除牌
func (h *Hand) RemoveByNumber(number int) error {
	for i, t := range h.tiles {
		if t.Number == number {
			h.tiles = append(h.tiles[:i], h.tiles[i+1:]...)
			return nil
		}
	}
	return ErrTileNotInHand.WithContext("number", number)
}

// Contains 检查是否包含指定的牌
func (h *Hand) Contains(tile Mahjong) bool {
	for _, t := range h.tiles {
		if t.Equals(tile) {
			return true
		}
	}
	return false
}

// ContainsNumber 检查是否包含指定数字的牌
func (h *Hand) ContainsNumber(number int) bool {
	for _, t := range h.tiles {
		if t.Number == number {
			return true
		}
	}
	return false
}

// Count 统计指定数字的牌的数量
func (h *Hand) Count(number int) int {
	count := 0
	for _, t := range h.tiles {
		if t.Number == number {
			count++
		}
	}
	return count
}

// Size 获取手牌数量
func (h *Hand) Size() int {
	return len(h.tiles)
}

// IsEmpty 判断手牌是否为空
func (h *Hand) IsEmpty() bool {
	return len(h.tiles) == 0
}

// IsFull 判断手牌是否已满（通常13张为满）
func (h *Hand) IsFull() bool {
	return len(h.tiles) >= StandardHandSize
}

// Clear 清空手牌
func (h *Hand) Clear() {
	h.tiles = h.tiles[:0]
}

// GetTiles 获取手牌列表（返回副本，防止外部修改）
func (h *Hand) GetTiles() []Mahjong {
	tiles := make([]Mahjong, len(h.tiles))
	copy(tiles, h.tiles)
	return tiles
}

// GetTilesRef 获取手牌列表引用（仅用于内部算法，外部不应使用）
func (h *Hand) GetTilesRef() []Mahjong {
	return h.tiles
}

// Sort 对手牌进行排序
func (h *Hand) Sort() {
	sort.Slice(h.tiles, func(i, j int) bool {
		return h.tiles[i].Number < h.tiles[j].Number
	})
}

// ToNumbers 转换为数字列表
func (h *Hand) ToNumbers() []int {
	numbers := make([]int, len(h.tiles))
	for i, t := range h.tiles {
		numbers[i] = t.Number
	}
	return numbers
}

// ToCountMap 转换为计数映射
func (h *Hand) ToCountMap() map[int]int {
	countMap := make(map[int]int, len(h.tiles))
	for _, t := range h.tiles {
		countMap[t.Number]++
	}
	return countMap
}

// Clone 克隆手牌
func (h *Hand) Clone() *Hand {
	newHand := NewHand()
	newHand.tiles = make([]Mahjong, len(h.tiles))
	copy(newHand.tiles, h.tiles)
	return newHand
}

// AddMultiple 批量添加牌
func (h *Hand) AddMultiple(tiles []Mahjong) {
	h.tiles = append(h.tiles, tiles...)
}

// SetTiles 设置手牌（用于初始化）
func (h *Hand) SetTiles(tiles []Mahjong) {
	h.tiles = make([]Mahjong, len(tiles))
	copy(h.tiles, tiles)
}

// GetColorDistribution 获取颜色分布
func (h *Hand) GetColorDistribution() map[int]int {
	distribution := make(map[int]int, 3)
	for _, t := range h.tiles {
		colorType := t.GetColorType()
		distribution[colorType]++
	}
	return distribution
}

// IsAllSameColor 判断是否全是同一花色
func (h *Hand) IsAllSameColor() bool {
	if len(h.tiles) == 0 {
		return false
	}
	firstColor := h.tiles[0].GetColorType()
	for _, t := range h.tiles {
		if t.GetColorType() != firstColor {
			return false
		}
	}
	return true
}

// IsReady 判断是否听牌（简单判断，需要配合算法）
func (h *Hand) IsReady() bool {
	// 这里只是简单判断手牌数量，具体听牌判断需要调用算法
	return h.Size() == StandardHandSize
}
