package htmajong

// 麻将牌常量
const (
	// TileCountPerType 每种牌的数量（通常为4）
	TileCountPerType = 4
	// TotalTileTypes 总共的牌型数量（27种：万子9种+条子9种+饼子9种）
	TotalTileTypes = 27
	// TotalTiles 总牌数
	TotalTiles = TotalTileTypes * TileCountPerType
)

// 麻将牌数字范围常量
const (
	// WanMin 万子最小值
	WanMin = 1
	// WanMax 万子最大值
	WanMax = 9
	// TiaoMin 条子最小值
	TiaoMin = 11
	// TiaoMax 条子最大值
	TiaoMax = 19
	// BingMin 饼子最小值
	BingMin = 21
	// BingMax 饼子最大值
	BingMax = 29
)

// Mahjong 麻将牌对象（值对象，不可变）
type Mahjong struct {
	Color  Color // 颜色
	Number int   // 数字（1-9万，11-19条，21-29饼）
	Order  int   // 生成顺序（用于调试和追踪）
}

// NewMahjong 创建麻将牌（带验证）
func NewMahjong(color Color, number int, order int) (Mahjong, error) {
	if !isValidMahjong(color, number) {
		return Mahjong{}, ErrInvalidMahjongNumber.WithContext("color", color).WithContext("number", number)
	}
	return Mahjong{
		Color:  color,
		Number: number,
		Order:  order,
	}, nil
}

// isValidMahjong 验证麻将牌是否合法
func isValidMahjong(color Color, number int) bool {
	switch color {
	case WAN:
		return number >= WanMin && number <= WanMax
	case TIAO:
		return number >= TiaoMin && number <= TiaoMax
	case BING:
		return number >= BingMin && number <= BingMax
	default:
		return false
	}
}

// Equals 判断两张牌是否相同（忽略 Order）
func (m Mahjong) Equals(other Mahjong) bool {
	return m.Color == other.Color && m.Number == other.Number
}

// Compare 比较两张牌的大小（按 Number 排序）
func (m Mahjong) Compare(other Mahjong) int {
	return m.Number - other.Number
}

// GetColorType 获取牌的颜色类型（0-万，1-条，2-饼）
func (m Mahjong) GetColorType() int {
	return m.Number / 10
}

// GetValue 获取牌面值（1-9）
func (m Mahjong) GetValue() int {
	return m.Number % 10
}

// IsSequential 判断三张牌是否是顺子
func IsSequential(tiles []Mahjong) bool {
	if len(tiles) != 3 {
		return false
	}
	// 必须是同一花色
	if tiles[0].GetColorType() != tiles[1].GetColorType() ||
		tiles[1].GetColorType() != tiles[2].GetColorType() {
		return false
	}
	// 必须连续
	return tiles[0].Number+1 == tiles[1].Number &&
		tiles[1].Number+1 == tiles[2].Number
}

// Generate 生成指定循环次数的麻将牌
//
// 参数：
//   - loop: 每种牌的数量（通常为4）
//
// 返回：
//   - 生成的麻将牌列表（共 27*loop 张）
func Generate(loop int) []Mahjong {
	tiles := make([]Mahjong, 0, TotalTileTypes*loop)
	order := 1

	// 万子 1-9
	for i := WanMin; i <= WanMax; i++ {
		for j := 0; j < loop; j++ {
			tiles = append(tiles, Mahjong{
				Color:  WAN,
				Number: i,
				Order:  order,
			})
			order++
		}
	}

	// 条子 11-19
	for i := TiaoMin; i <= TiaoMax; i++ {
		for j := 0; j < loop; j++ {
			tiles = append(tiles, Mahjong{
				Color:  TIAO,
				Number: i,
				Order:  order,
			})
			order++
		}
	}

	// 饼子 21-29
	for i := BingMin; i <= BingMax; i++ {
		for j := 0; j < loop; j++ {
			tiles = append(tiles, Mahjong{
				Color:  BING,
				Number: i,
				Order:  order,
			})
			order++
		}
	}

	return tiles
}

// GenerateByNumber 根据数字生成麻将牌
//
// 参数：
//   - number: 麻将牌数字（1-9万，11-19条，21-29饼）
//
// 返回：
//   - 麻将牌对象
//   - 错误（如果数字无效）
func GenerateByNumber(number int) (Mahjong, error) {
	var color Color
	switch {
	case number >= WanMin && number <= WanMax:
		color = WAN
	case number >= TiaoMin && number <= TiaoMax:
		color = TIAO
	case number >= BingMin && number <= BingMax:
		color = BING
	default:
		return Mahjong{}, ErrInvalidMahjongNumber.WithContext("number", number)
	}
	return Mahjong{Color: color, Number: number, Order: number}, nil
}
