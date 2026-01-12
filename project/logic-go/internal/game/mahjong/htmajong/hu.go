package htmajong

// HuType 胡牌类型枚举
type HuType int

const (
	// GENERAL 普通胡
	GENERAL HuType = iota
	// CLEAR 清一色
	CLEAR
	// PENG_PENG_HU 碰碰胡
	PENG_PENG_HU
	// TWO_FIVE_EIGHT 258
	TWO_FIVE_EIGHT
	// SEVEN_PAIR 七小对
	SEVEN_PAIR
	// LOONG_SEVEN_PAIR 龙七对
	LOONG_SEVEN_PAIR
	// BAO_TING 报听
	BAO_TING
	// TWO_COLOR 缺一门
	TWO_COLOR
	// NO_JIANG 无将糊
	NO_JIANG
)

func (h HuType) String() string {
	switch h {
	case GENERAL:
		return "GENERAL"
	case CLEAR:
		return "CLEAR"
	case PENG_PENG_HU:
		return "PENG_PENG_HU"
	case TWO_FIVE_EIGHT:
		return "TWO_FIVE_EIGHT"
	case SEVEN_PAIR:
		return "SEVEN_PAIR"
	case LOONG_SEVEN_PAIR:
		return "LOONG_SEVEN_PAIR"
	case BAO_TING:
		return "BAO_TING"
	case TWO_COLOR:
		return "TWO_COLOR"
	case NO_JIANG:
		return "NO_JIANG"
	default:
		return "UNKNOWN"
	}
}

// HuDetail 胡牌详情
type HuDetail struct {
	Position  Position     // 位置
	HuType    []HuType     // 胡牌类型列表
	Operation SupplierType // 操作类型
	Points    int          // 分数
}

// LoseDetail 输牌详情
type LoseDetail struct {
	Position  Position     // 位置
	Operation SupplierType // 操作类型
	Points    int          // 分数
}
