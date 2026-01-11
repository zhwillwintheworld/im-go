package model

import "time"

// User 用户模型
type User struct {
	ID           int64     `json:"id,string" db:"id"`
	Username     string    `json:"username" db:"username"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Nickname     string    `json:"nickname" db:"nickname"`
	Avatar       string    `json:"avatar" db:"avatar"`
	Status       int       `json:"status" db:"status"`
	CreateAt     time.Time `json:"createAt" db:"create_at"`
	UpdateAt     time.Time `json:"updateAt" db:"update_at"`
	Deleted      int       `json:"-" db:"deleted"`
}

// UserStatus 用户状态
const (
	UserStatusNormal   = 0 // 正常
	UserStatusDisabled = 1 // 禁用
)
