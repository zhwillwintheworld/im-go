package model

import "time"

// Room 房间模型
type Room struct {
	RoomID       string            `json:"room_id"`       // 房间ID（雪花ID）
	RoomName     string            `json:"room_name"`     // 房间名称
	RoomPassword string            `json:"room_password"` // 房间密码（可选，空表示无密码）
	RoomType     string            `json:"room_type"`     // 房间类型（public/private）
	MaxPlayers   int               `json:"max_players"`   // 人数上限
	GameType     string            `json:"game_type"`     // 游戏类型（如：HT_MAHJONG）
	GameSettings map[string]string `json:"game_settings"` // 游戏设置（如：玩法、规则等）
	Extension    map[string]string `json:"extension"`     // 拓展信息
	CreatorID    int64             `json:"creator_id"`    // 创建者用户ID
	CreatedAt    time.Time         `json:"created_at"`    // 创建时间
	UpdatedAt    time.Time         `json:"updated_at"`    // 更新时间
	Status       string            `json:"status"`        // 房间状态（waiting/playing/finished）
	Players      []RoomPlayer      `json:"players"`       // 房间玩家列表
}

// RoomPlayer 房间玩家信息
type RoomPlayer struct {
	UserID    int64 `json:"user_id"`    // 用户ID
	SeatIndex int32 `json:"seat_index"` // 座位索引
	IsReady   bool  `json:"is_ready"`   // 是否准备
	IsHost    bool  `json:"is_host"`    // 是否房主
	UserInfo  *User `json:"user_info"`  // 用户信息
}

// RoomConfig 创建房间配置（从 roomConfig JSON 解析）
type RoomConfig struct {
	RoomName     string            `json:"room_name"`               // 房间名称
	RoomPassword string            `json:"room_password,omitempty"` // 房间密码（可选）
	RoomType     string            `json:"room_type"`               // 房间类型（public/private）
	MaxPlayers   int               `json:"max_players"`             // 人数上限
	GameSettings map[string]string `json:"game_settings,omitempty"` // 游戏设置
	Extension    map[string]string `json:"extension,omitempty"`     // 拓展信息
}
