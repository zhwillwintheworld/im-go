package model

import "time"

// Group 群组实体
type Group struct {
	Id          int64     `json:"id"`
	Name        string    `json:"name"`
	OwnerId     int64     `json:"ownerId"`
	Avatar      string    `json:"avatar"`
	Description string    `json:"description"`
	MemberCount int       `json:"memberCount"`
	Status      int       `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// GroupMember 群成员
type GroupMember struct {
	GroupId   int64     `json:"groupId"`
	UserId    int64     `json:"userId"`
	Role      int       `json:"role"` // 0: 成员, 1: 管理员, 2: 群主
	Nickname  string    `json:"nickname"`
	JoinedAt  time.Time `json:"joinedAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
