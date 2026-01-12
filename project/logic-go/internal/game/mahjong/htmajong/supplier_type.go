package htmajong

// SupplierType 提供方式枚举
type SupplierType int

const (
	// CATCH 抓牌
	CATCH SupplierType = iota
	// OUT 出牌
	OUT
	// GANG 杠
	GANG
)

func (s SupplierType) String() string {
	switch s {
	case CATCH:
		return "CATCH"
	case OUT:
		return "OUT"
	case GANG:
		return "GANG"
	default:
		return "UNKNOWN"
	}
}
