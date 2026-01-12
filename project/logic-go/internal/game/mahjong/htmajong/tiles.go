package htmajong

// PublicTiles 公开牌管理器（碰、杠的牌）
type PublicTiles struct {
	groups []TileGroup // 牌组列表
}

// TileGroup 牌组
type TileGroup struct {
	Type  TileGroupType // 牌组类型
	Tiles []Mahjong     // 牌组内的牌
}

// TileGroupType 牌组类型
type TileGroupType int

const (
	// GroupTypePong 碰（3张相同）
	GroupTypePong TileGroupType = iota
	// GroupTypeKong 杠（4张相同）
	GroupTypeKong
	// GroupTypeExposedKong 明杠（从别人出的牌杠）
	GroupTypeExposedKong
	// GroupTypeConcealedKong 暗杠（自己摸的牌杠）
	GroupTypeConcealedKong
)

// NewPublicTiles 创建公开牌管理器
func NewPublicTiles() *PublicTiles {
	return &PublicTiles{
		groups: make([]TileGroup, 0, 4), // 最多4组（碰或杠）
	}
}

// AddPong 添加碰牌组
func (p *PublicTiles) AddPong(tiles []Mahjong) error {
	if len(tiles) != 3 {
		return ErrInvalidMove.WithContext("operation", "pong").WithContext("tileCount", len(tiles))
	}
	p.groups = append(p.groups, TileGroup{
		Type:  GroupTypePong,
		Tiles: tiles,
	})
	return nil
}

// AddKong 添加杠牌组
func (p *PublicTiles) AddKong(tiles []Mahjong, kongType TileGroupType) error {
	if len(tiles) != 4 {
		return ErrInvalidMove.WithContext("operation", "kong").WithContext("tileCount", len(tiles))
	}
	p.groups = append(p.groups, TileGroup{
		Type:  kongType,
		Tiles: tiles,
	})
	return nil
}

// GetAllTiles 获取所有公开牌
func (p *PublicTiles) GetAllTiles() []Mahjong {
	tiles := make([]Mahjong, 0)
	for _, group := range p.groups {
		tiles = append(tiles, group.Tiles...)
	}
	return tiles
}

// GetGroups 获取所有牌组
func (p *PublicTiles) GetGroups() []TileGroup {
	groups := make([]TileGroup, len(p.groups))
	copy(groups, p.groups)
	return groups
}

// CountByNumber 统计指定数字的牌数量
func (p *PublicTiles) CountByNumber(number int) int {
	count := 0
	for _, group := range p.groups {
		for _, tile := range group.Tiles {
			if tile.Number == number {
				count++
			}
		}
	}
	return count
}

// HasPong 判断是否有碰牌组
func (p *PublicTiles) HasPong() bool {
	for _, group := range p.groups {
		if group.Type == GroupTypePong {
			return true
		}
	}
	return false
}

// IsEmpty 判断是否为空
func (p *PublicTiles) IsEmpty() bool {
	return len(p.groups) == 0
}

// Count 获取牌组数量
func (p *PublicTiles) Count() int {
	return len(p.groups)
}

// DiscardPile 出牌堆管理器
type DiscardPile struct {
	tiles []Mahjong // 出牌列表
}

// NewDiscardPile 创建出牌堆
func NewDiscardPile() *DiscardPile {
	return &DiscardPile{
		tiles: make([]Mahjong, 0, 20), // 预分配20张牌的空间
	}
}

// Add 添加出牌
func (d *DiscardPile) Add(tile Mahjong) {
	d.tiles = append(d.tiles, tile)
}

// GetLast 获取最后出的牌
func (d *DiscardPile) GetLast() (Mahjong, error) {
	if len(d.tiles) == 0 {
		return Mahjong{}, ErrHandEmpty
	}
	return d.tiles[len(d.tiles)-1], nil
}

// GetAll 获取所有出牌
func (d *DiscardPile) GetAll() []Mahjong {
	tiles := make([]Mahjong, len(d.tiles))
	copy(tiles, d.tiles)
	return tiles
}

// Size 获取出牌数量
func (d *DiscardPile) Size() int {
	return len(d.tiles)
}

// IsEmpty 判断是否为空
func (d *DiscardPile) IsEmpty() bool {
	return len(d.tiles) == 0
}

// Clear 清空出牌堆
func (d *DiscardPile) Clear() {
	d.tiles = d.tiles[:0]
}

// RemoveLast 移除最后一张牌（用于抢杠）
func (d *DiscardPile) RemoveLast() (Mahjong, error) {
	if len(d.tiles) == 0 {
		return Mahjong{}, ErrHandEmpty
	}
	lastTile := d.tiles[len(d.tiles)-1]
	d.tiles = d.tiles[:len(d.tiles)-1]
	return lastTile, nil
}
