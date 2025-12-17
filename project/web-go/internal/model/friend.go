package model

import "time"

// FriendRequest 好友邀请
type FriendRequest struct {
    ID         int64     `json:"id" db:"id"`
    ObjectCode string    `json:"object_code" db:"object_code"`
    FromUserID int64     `json:"from_user_id" db:"from_user_id"`
    ToUserID   int64     `json:"to_user_id" db:"to_user_id"`
    Message    string    `json:"message" db:"message"`
    Status     int       `json:"status" db:"status"`
    CreateAt   time.Time `json:"create_at" db:"create_at"`
    UpdateAt   time.Time `json:"update_at" db:"update_at"`
    Deleted    int       `json:"-" db:"deleted"`
}

// FriendRequestStatus 好友邀请状态
const (
    FriendRequestStatusPending  = 0 // 待处理
    FriendRequestStatusAccepted = 1 // 已同意
    FriendRequestStatusRejected = 2 // 已拒绝
)

// FriendRequestWithUser 包含用户信息的好友邀请
type FriendRequestWithUser struct {
    FriendRequest
    FromUsername string `json:"from_username"`
    FromNickname string `json:"from_nickname"`
    FromAvatar   string `json:"from_avatar"`
}

// Friend 好友关系（只存储已确认的好友）
type Friend struct {
    ID         int64     `json:"id" db:"id"`
    ObjectCode string    `json:"object_code" db:"object_code"`
    UserID     int64     `json:"user_id" db:"user_id"`
    FriendID   int64     `json:"friend_id" db:"friend_id"`
    Remark     string    `json:"remark" db:"remark"`
    CreateAt   time.Time `json:"create_at" db:"create_at"`
    UpdateAt   time.Time `json:"update_at" db:"update_at"`
    Deleted    int       `json:"-" db:"deleted"`
}

// FriendWithUser 包含用户信息的好友
type FriendWithUser struct {
    Friend
    Username string `json:"username"`
    Nickname string `json:"nickname"`
    Avatar   string `json:"avatar"`
}
