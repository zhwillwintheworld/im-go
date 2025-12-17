package model

import "time"

// Friend 好友关系
type Friend struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	FriendID  int64     `json:"friend_id"`
	Remark    string    `json:"remark"`
	CreatedAt time.Time `json:"created_at"`
}

// FriendRequest 好友请求
type FriendRequest struct {
	ID         int64     `json:"id"`
	FromUserID int64     `json:"from_user_id"`
	ToUserID   int64     `json:"to_user_id"`
	Message    string    `json:"message"`
	Status     int       `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// FriendRequestStatus 好友请求状态
const (
	FriendRequestPending  = 0 // 待处理
	FriendRequestAccepted = 1 // 已接受
	FriendRequestRejected = 2 // 已拒绝
)

// FriendWithUser 包含用户信息的好友
type FriendWithUser struct {
	Friend
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

// FriendRequestWithUser 包含用户信息的好友请求
type FriendRequestWithUser struct {
	FriendRequest
	FromUsername string `json:"from_username"`
	FromNickname string `json:"from_nickname"`
	FromAvatar   string `json:"from_avatar"`
}
