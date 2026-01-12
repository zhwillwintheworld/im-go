package htmajong

// Position 位置枚举
type Position int

const (
	// EAST 东
	EAST Position = iota
	// SOUTH 南
	SOUTH
	// WEST 西
	WEST
	// NORTH 北
	NORTH
)

func (p Position) String() string {
	switch p {
	case EAST:
		return "EAST"
	case SOUTH:
		return "SOUTH"
	case WEST:
		return "WEST"
	case NORTH:
		return "NORTH"
	default:
		return "UNKNOWN"
	}
}
