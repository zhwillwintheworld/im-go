package model

// Conversation 会话信息（存储在 Redis）
type Conversation struct {
	PeerID        int64 `json:"peer_id,omitempty"`  // 私聊对方ID
	GroupID       int64 `json:"group_id,omitempty"` // 群聊ID
	LastMsgID     int64 `json:"last_msg_id"`        // 最后一条消息ID
	LastReadMsgID int64 `json:"last_read_msg_id"`   // 最后已读消息ID
	UnreadCount   int   `json:"unread_count"`       // 未读数
	IsPinned      bool  `json:"is_pinned"`          // 是否置顶
	IsMuted       bool  `json:"is_muted"`           // 是否静音
	UpdateAt      int64 `json:"update_at"`          // 更新时间（毫秒）
}
