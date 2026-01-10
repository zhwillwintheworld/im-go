package room

import "errors"

// 房间错误定义

var (
	ErrRoomNotFound        = errors.New("ROOM_NOT_FOUND")
	ErrRoomFull            = errors.New("ROOM_FULL")
	ErrRoomBusy            = errors.New("ROOM_BUSY")
	ErrInvalidPassword     = errors.New("INVALID_PASSWORD")
	ErrGameStarted         = errors.New("GAME_STARTED")
	ErrAlreadyInRoom       = errors.New("ALREADY_IN_ROOM")
	ErrNotInRoom           = errors.New("NOT_IN_ROOM")
	ErrNotRoomHost         = errors.New("NOT_ROOM_HOST")
	ErrSeatOccupied        = errors.New("SEAT_OCCUPIED")
	ErrInvalidSeat         = errors.New("INVALID_SEAT")
	ErrPlayerReady         = errors.New("PLAYER_READY")
	ErrNotAllReady         = errors.New("NOT_ALL_READY")
	ErrNotEnoughPlayers    = errors.New("NOT_ENOUGH_PLAYERS")
	ErrUnsupportedGameType = errors.New("UNSUPPORTED_GAME_TYPE")
	ErrCannotStartGame     = errors.New("CANNOT_START_GAME")
	ErrLockFailed          = errors.New("LOCK_FAILED")
)
