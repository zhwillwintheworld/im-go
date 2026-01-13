package htmajong

import "sort"

// Hand 手牌管理器
type Hand struct {
	tiles    []Mahjong   // 手牌列表
	countMap map[int]int // 牌数量缓存，用于加速查找（O(1)）
}

// NewHand 创建新的手牌管理器
func NewHand() *Hand {
	return &Hand{
		tiles:    make([]Mahjong, 0, StandardHandSize+1),
		countMap: make(map[int]int),
	}
}

// Add 添加牌到手牌
func (h *Hand) Add(tile Mahjong) error {
	h.tiles = append(h.tiles, tile)
	h.countMap[tile.Number]++
	return nil
}

// Remove 移除指定的牌
func (h *Hand) Remove(tile Mahjong) error {
	// 先检查 countMap，快速判断是否存在
	if h.countMap[tile.Number] == 0 {
		return ErrTileNotInHand.WithContext("tile", tile.Number)
	}

	// 线性查找并删除
	for i, t := range h.tiles {
		if t.Equals(tile) {
			h.tiles = append(h.tiles[:i], h.tiles[i+1:]...)
			h.countMap[tile.Number]--
			if h.countMap[tile.Number] == 0 {
				delete(h.countMap, tile.Number)
			}
			return nil
		}
	}
	return ErrTileNotInHand.WithContext("tile", tile.Number)
}

// RemoveByNumber 根据数字移除牌
func (h *Hand) RemoveByNumber(number int) error {
	// 先检查 countMap，快速判断是否存在
	if h.countMap[number] == 0 {
		return ErrTileNotInHand.WithContext("number", number)
	}

	// 线性查找并删除
	for i, t := range h.tiles {
		if t.Number == number {
			h.tiles = append(h.tiles[:i], h.tiles[i+1:]...)
			h.countMap[number]--
			if h.countMap[number] == 0 {
				delete(h.countMap, number)
			}
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

// Count 统计指定数字的牌的数量（O(1)优化）
func (h *Hand) Count(number int) int {
	return h.countMap[number]
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
	h.countMap = make(map[int]int)
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

// ToCountMap 转换为计数映射（直接返回缓存的 countMap）
func (h *Hand) ToCountMap() map[int]int {
	// 返回副本以防止外部修改
	result := make(map[int]int, len(h.countMap))
	for k, v := range h.countMap {
		result[k] = v
	}
	return result
}

// Clone 克隆手牌
func (h *Hand) Clone() *Hand {
	newHand := NewHand()
	newHand.tiles = make([]Mahjong, len(h.tiles))
	copy(newHand.tiles, h.tiles)
	// 复制 countMap
	for k, v := range h.countMap {
		newHand.countMap[k] = v
	}
	return newHand
}

// AddMultiple 批量添加牌
func (h *Hand) AddMultiple(tiles []Mahjong) {
	h.tiles = append(h.tiles, tiles...)
	// 更新 countMap
	for _, t := range tiles {
		h.countMap[t.Number]++
	}
}

// SetTiles 设置手牌（用于初始化）
func (h *Hand) SetTiles(tiles []Mahjong) {
	h.tiles = make([]Mahjong, len(tiles))
	copy(h.tiles, tiles)
	// 重建 countMap
	h.countMap = make(map[int]int)
	for _, t := range tiles {
		h.countMap[t.Number]++
	}
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
