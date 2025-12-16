package model

import "time"

// UserLocation 用户位置信息（用于消息路由）
type UserLocation struct {
	UserId       int64     `json:"userId"`
	AccessNodeId string    `json:"accessNodeId"`
	ConnId       int64     `json:"connId"`
	DeviceId     string    `json:"deviceId"`
	Platform     string    `json:"platform"`
	LoginTime    time.Time `json:"loginTime"`
}

// User 用户实体
type User struct {
	Id        int64     `json:"id"`
	Username  string    `json:"username"`
	Nickname  string    `json:"nickname"`
	Avatar    string    `json:"avatar"`
	Status    int       `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
