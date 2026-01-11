package model

// Conversation 会话信息
type Conversation struct {
	PeerID        int64 `json:"peerId,omitempty"`  // 私聊对方ID
	GroupID       int64 `json:"groupId,omitempty"` // 群聊ID
	LastMsgID     int64 `json:"lastMsgId"`         // 最后一条消息ID
	LastReadMsgID int64 `json:"lastReadMsgId"`     // 最后已读消息ID
	UnreadCount   int   `json:"unreadCount"`       // 未读数
	IsPinned      bool  `json:"isPinned"`          // 是否置顶
	IsMuted       bool  `json:"isMuted"`           // 是否静音
	UpdateAt      int64 `json:"updateAt"`          // 更新时间（毫秒）
}
