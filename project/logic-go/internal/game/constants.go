package game

// 游戏类型常量（与客户端协议保持一致）
const (
	GameTypeHTMahjong = "HT_MAHJONG" // 会同麻将
	GameTypeTHMahjong = "TH_MAHJONG" // 太湖麻将
)

// gameTypeInternalMap 外部协议类型 → 内部引擎类型映射
var gameTypeInternalMap = map[string]string{
	GameTypeHTMahjong: "huitong",
	GameTypeTHMahjong: "taihu",
}

// GetInternalGameType 获取内部游戏类型
func GetInternalGameType(externalType string) (string, bool) {
	t, ok := gameTypeInternalMap[externalType]
	return t, ok
}
