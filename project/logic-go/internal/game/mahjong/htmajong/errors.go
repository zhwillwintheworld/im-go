package htmajong

import "fmt"

// GameError 游戏错误类型
type GameError struct {
	Code    string                 // 错误代码
	Message string                 // 错误消息
	Cause   error                  // 原因错误
	Context map[string]interface{} // 错误上下文
}

func (e *GameError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *GameError) Unwrap() error {
	return e.Cause
}

// NewGameError 创建游戏错误
func NewGameError(code, message string) *GameError {
	return &GameError{
		Code:    code,
		Message: message,
		Context: make(map[string]interface{}),
	}
}

// WithCause 添加原因错误
func (e *GameError) WithCause(cause error) *GameError {
	e.Cause = cause
	return e
}

// WithContext 添加上下文信息
func (e *GameError) WithContext(key string, value interface{}) *GameError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// 麻将牌相关错误
var (
	ErrInvalidMahjongNumber = NewGameError("INVALID_MAHJONG_NUMBER", "麻将牌数字必须在有效范围内（1-9万，11-19条，21-29饼）")
	ErrInvalidMahjongColor  = NewGameError("INVALID_MAHJONG_COLOR", "无效的麻将牌颜色")
)

// 手牌相关错误
var (
	ErrHandFull        = NewGameError("HAND_FULL", "手牌已满，无法继续摸牌")
	ErrHandEmpty       = NewGameError("HAND_EMPTY", "手牌为空")
	ErrTileNotInHand   = NewGameError("TILE_NOT_IN_HAND", "手牌中没有指定的牌")
	ErrInvalidHandSize = NewGameError("INVALID_HAND_SIZE", "手牌数量不正确")
)

// 座位相关错误
var (
	ErrInvalidPosition = NewGameError("INVALID_POSITION", "无效的座位位置")
	ErrSeatNotFound    = NewGameError("SEAT_NOT_FOUND", "座位不存在")
	ErrPlayerNotInSeat = NewGameError("PLAYER_NOT_IN_SEAT", "玩家不在该座位")
)

// 牌桌相关错误
var (
	ErrTableNotFound      = NewGameError("TABLE_NOT_FOUND", "牌桌不存在")
	ErrTableFull          = NewGameError("TABLE_FULL", "牌桌已满")
	ErrGameNotStarted     = NewGameError("GAME_NOT_STARTED", "游戏尚未开始")
	ErrGameAlreadyStarted = NewGameError("GAME_ALREADY_STARTED", "游戏已经开始")
	ErrNotEnoughPlayers   = NewGameError("NOT_ENOUGH_PLAYERS", "玩家数量不足")
	ErrInvalidGamePhase   = NewGameError("INVALID_GAME_PHASE", "当前游戏阶段不允许此操作")
	ErrDeckEmpty          = NewGameError("DECK_EMPTY", "牌堆已空")
)

// 游戏规则相关错误
var (
	ErrInvalidMove        = NewGameError("INVALID_MOVE", "无效的操作")
	ErrCannotWin          = NewGameError("CANNOT_WIN", "不能胡牌")
	ErrCannotPong         = NewGameError("CANNOT_PONG", "不能碰")
	ErrCannotKong         = NewGameError("CANNOT_KONG", "不能杠")
	ErrCannotDeclareReady = NewGameError("CANNOT_DECLARE_READY", "不能报听")
	ErrResponseTimeout    = NewGameError("RESPONSE_TIMEOUT", "玩家响应超时")
)

// 状态相关错误
var (
	ErrInvalidStateTransition = NewGameError("INVALID_STATE_TRANSITION", "无效的状态转换")
	ErrInvalidState           = NewGameError("INVALID_STATE", "无效的状态")
)
