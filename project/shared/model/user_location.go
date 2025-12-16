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
