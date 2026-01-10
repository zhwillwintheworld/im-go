package game

import "errors"

// 游戏相关错误定义

var (
	// ErrGameNotFound 游戏不存在
	ErrGameNotFound = errors.New("GAME_NOT_FOUND")

	// ErrInvalidGameState 无效的游戏状态
	ErrInvalidGameState = errors.New("INVALID_GAME_STATE")

	// ErrInvalidAction 无效的游戏操作
	ErrInvalidAction = errors.New("INVALID_ACTION")
)
