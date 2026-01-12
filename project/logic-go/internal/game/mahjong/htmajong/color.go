package htmajong

// Color 麻将颜色枚举
type Color int

const (
	// WAN 万子
	WAN Color = iota
	// TIAO 条子
	TIAO
	// BING 饼子
	BING
	// BACKGROUND 背部
	BACKGROUND
)

func (c Color) String() string {
	switch c {
	case WAN:
		return "WAN"
	case TIAO:
		return "TIAO"
	case BING:
		return "BING"
	case BACKGROUND:
		return "BACKGROUND"
	default:
		return "UNKNOWN"
	}
}
