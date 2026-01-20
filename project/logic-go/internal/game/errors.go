package game

import "errors"

// 游戏相关错误定义

var (
	// ErrGameNotFound 游戏不存在
	ErrGameNotFound = errors.New("game not found")

	// ErrInvalidGameState 无效的游戏状态
	ErrInvalidGameState = errors.New("invalid game state")

	// ErrInvalidAction 无效的游戏操作
	ErrInvalidAction = errors.New("invalid action")

	// ErrEngineNotInitialized 引擎未初始化
	ErrEngineNotInitialized = errors.New("engine not initialized")

	// ErrEngineNotSupported 引擎不支持该操作
	ErrEngineNotSupported = errors.New("engine does not support this operation")
)
