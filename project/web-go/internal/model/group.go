package model

import "time"

// GroupStatus 群组状态
const (
    GroupStatusNormal    = 0 // 正常
    GroupStatusDissolved = 1 // 解散
)

// GroupMemberRole 群成员角色
const (
    GroupMemberRoleMember = 0 // 成员
    GroupMemberRoleAdmin  = 1 // 管理员
    GroupMemberRoleOwner  = 2 // 群主
)

// Group 群组
type Group struct {
    ID          int64     `json:"id" db:"id"`
    ObjectCode  string    `json:"object_code" db:"object_code"`
    Name        string    `json:"name" db:"name"`
    OwnerID     int64     `json:"owner_id" db:"owner_id"`
    Avatar      string    `json:"avatar" db:"avatar"`
    Description string    `json:"description" db:"description"`
    MaxMembers  int       `json:"max_members" db:"max_members"`
    Status      int       `json:"status" db:"status"`
    CreateAt    time.Time `json:"create_at" db:"create_at"`
    UpdateAt    time.Time `json:"update_at" db:"update_at"`
    Deleted     int       `json:"-" db:"deleted"`
}

// GroupMember 群成员
type GroupMember struct {
    ID         int64     `json:"id" db:"id"`
    ObjectCode string    `json:"object_code" db:"object_code"`
    GroupID    int64     `json:"group_id" db:"group_id"`
    UserID     int64     `json:"user_id" db:"user_id"`
    Role       int       `json:"role" db:"role"`
    Nickname   string    `json:"nickname" db:"nickname"`
    CreateAt   time.Time `json:"create_at" db:"create_at"`
    UpdateAt   time.Time `json:"update_at" db:"update_at"`
    Deleted    int       `json:"-" db:"deleted"`
}

// GroupWithMemberCount 带成员数量的群组
type GroupWithMemberCount struct {
    Group
    MemberCount int `json:"member_count"`
}

// GroupMemberWithUser 带用户信息的群成员
type GroupMemberWithUser struct {
    GroupMember
    Username string `json:"username"`
    Nickname string `json:"user_nickname"`
    Avatar   string `json:"avatar"`
}
