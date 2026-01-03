package room

import "errors"

// 房间相关错误定义

var (
	// ErrRoomNotFound 房间不存在
	ErrRoomNotFound = errors.New("ROOM_NOT_FOUND")

	// ErrRoomFull 房间已满
	ErrRoomFull = errors.New("ROOM_FULL")

	// ErrRoomBusy 房间正在处理其他操作
	ErrRoomBusy = errors.New("ROOM_BUSY")

	// ErrInvalidPassword 房间密码错误
	ErrInvalidPassword = errors.New("INVALID_PASSWORD")

	// ErrGameStarted 游戏已开始
	ErrGameStarted = errors.New("GAME_STARTED")

	// ErrAlreadyInRoom 用户已在房间中
	ErrAlreadyInRoom = errors.New("ALREADY_IN_ROOM")

	// ErrNotInRoom 用户不在房间中
	ErrNotInRoom = errors.New("NOT_IN_ROOM")

	// ErrNotRoomHost 不是房主
	ErrNotRoomHost = errors.New("NOT_ROOM_HOST")

	// ErrSeatOccupied 座位已被占用
	ErrSeatOccupied = errors.New("SEAT_OCCUPIED")

	// ErrInvalidSeat 无效的座位索引
	ErrInvalidSeat = errors.New("INVALID_SEAT")

	// ErrCannotStartGame 无法开始游戏
	ErrCannotStartGame = errors.New("CANNOT_START_GAME")

	// ErrLockFailed 获取锁失败
	ErrLockFailed = errors.New("LOCK_FAILED")
)

// RoomError 房间错误（带上下文信息）
type RoomError struct {
	Code    string // 错误码
	Message string // 错误消息
	Err     error  // 原始错误
}

func (e *RoomError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *RoomError) Unwrap() error {
	return e.Err
}

// NewRoomError 创建房间错误
func NewRoomError(code string, message string, err error) *RoomError {
	return &RoomError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}
