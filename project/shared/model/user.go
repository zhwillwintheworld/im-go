package model

// User 用户基本信息（用于房间、聊天等场景显示）
type User struct {
	UserID   int64  `json:"userId,string"` // 用户ID
	Username string `json:"username"`      // 用户名
	Nickname string `json:"nickname"`      // 昵称
	Avatar   string `json:"avatar"`        // 头像URL
}
