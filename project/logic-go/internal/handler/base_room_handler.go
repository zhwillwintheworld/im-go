package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"sudooom.im.logic/internal/room"
	"sudooom.im.logic/internal/service"
	sharedModel "sudooom.im.shared/model"
)

// ErrorResponse 统一错误响应
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	RoomID  string `json:"roomId,omitempty"`
}

// BaseRoomHandler 房间操作基类
// 提供通用的错误处理、响应发送等功能，避免代码重复
type BaseRoomHandler struct {
	roomService   *room.RoomService
	routerService *service.RouterService
	logger        *slog.Logger
}

// NewBaseRoomHandler 创建基类
func NewBaseRoomHandler(roomService *room.RoomService, routerService *service.RouterService) *BaseRoomHandler {
	return &BaseRoomHandler{
		roomService:   roomService,
		routerService: routerService,
		logger:        slog.Default(),
	}
}

// SendErrorResponse 统一错误响应（避免代码重复）
func (h *BaseRoomHandler) SendErrorResponse(
	_ context.Context,
	accessNodeId string,
	connId int64,
	platform string,
	userId int64,
	roomId string,
	err error,
) {
	errorResp := h.MapError(err, roomId)

	respBytes, marshalErr := json.Marshal(errorResp)
	if marshalErr != nil {
		h.logger.Error("Failed to marshal error response", "error", marshalErr)
		return
	}

	senderLoc := sharedModel.UserLocation{
		AccessNodeId: accessNodeId,
		ConnId:       connId,
		Platform:     platform,
		UserId:       userId,
	}

	if sendErr := h.routerService.SendRoomPushToSelf(senderLoc, "ERROR", roomId, respBytes); sendErr != nil {
		h.logger.Warn("Failed to send error response", "userId", userId, "error", sendErr)
	}
}

// MapError 统一错误映射（避免每个 Handler 重复实现）
func (h *BaseRoomHandler) MapError(err error, roomId string) *ErrorResponse {
	if err == nil {
		return &ErrorResponse{Code: "UNKNOWN_ERROR", Message: "未知错误", RoomID: roomId}
	}

	// 使用 errors.Is 判断错误类型
	switch {
	case errors.Is(err, room.ErrRoomNotFound):
		return &ErrorResponse{Code: "ROOM_NOT_FOUND", Message: "房间不存在", RoomID: roomId}
	case errors.Is(err, room.ErrRoomFull):
		return &ErrorResponse{Code: "ROOM_FULL", Message: "房间已满", RoomID: roomId}
	case errors.Is(err, room.ErrRoomBusy):
		return &ErrorResponse{Code: "ROOM_BUSY", Message: "房间繁忙", RoomID: roomId}
	case errors.Is(err, room.ErrInvalidPassword):
		return &ErrorResponse{Code: "INVALID_PASSWORD", Message: "密码错误", RoomID: roomId}
	case errors.Is(err, room.ErrGameStarted):
		return &ErrorResponse{Code: "GAME_STARTED", Message: "游戏已开始", RoomID: roomId}
	case errors.Is(err, room.ErrAlreadyInRoom):
		return &ErrorResponse{Code: "ALREADY_IN_ROOM", Message: "已在房间中", RoomID: roomId}
	case errors.Is(err, room.ErrLockFailed):
		return &ErrorResponse{Code: "LOCK_FAILED", Message: "获取锁失败", RoomID: roomId}
	case errors.Is(err, room.ErrNotRoomHost):
		return &ErrorResponse{Code: "NOT_ROOM_HOST", Message: "不是房主", RoomID: roomId}
	case errors.Is(err, room.ErrNotAllReady):
		return &ErrorResponse{Code: "NOT_ALL_READY", Message: "玩家未全部准备", RoomID: roomId}
	case errors.Is(err, room.ErrUnsupportedGameType):
		return &ErrorResponse{Code: "UNSUPPORTED_GAME_TYPE", Message: "不支持的游戏类型", RoomID: roomId}
	case errors.Is(err, room.ErrSeatOccupied):
		return &ErrorResponse{Code: "SEAT_OCCUPIED", Message: "座位已被占用", RoomID: roomId}
	case errors.Is(err, room.ErrInvalidSeat):
		return &ErrorResponse{Code: "INVALID_SEAT", Message: "无效的座位", RoomID: roomId}
	case errors.Is(err, room.ErrNotInRoom):
		return &ErrorResponse{Code: "NOT_IN_ROOM", Message: "不在房间中", RoomID: roomId}
	case errors.Is(err, room.ErrNotEnoughPlayers):
		return &ErrorResponse{Code: "NOT_ENOUGH_PLAYERS", Message: "玩家人数不足", RoomID: roomId}
	default:
		return &ErrorResponse{Code: "OPERATION_FAILED", Message: err.Error(), RoomID: roomId}
	}
}

// ParseRoomConfig 解析房间配置（通用方法）
func (h *BaseRoomHandler) ParseRoomConfig(configStr string) (map[string]string, error) {
	if configStr == "" {
		return map[string]string{
			"roomName": "房间",
			"roomType": "public",
		}, nil
	}

	var config map[string]string
	if err := json.Unmarshal([]byte(configStr), &config); err != nil {
		return nil, errors.New("配置格式错误")
	}

	// 设置默认值
	if config["roomName"] == "" {
		config["roomName"] = "房间"
	}
	if config["roomType"] == "" {
		if config["roomPassword"] == "" {
			config["roomType"] = "public"
		} else {
			config["roomType"] = "private"
		}
	}

	return config, nil
}

// ParseMaxPlayers 解析最大玩家数（通用方法）
func (h *BaseRoomHandler) ParseMaxPlayers(maxPlayersStr string) int {
	if maxPlayersStr == "" {
		return 4
	}
	if n, err := json.Number(maxPlayersStr).Int64(); err == nil && n > 0 {
		return int(n)
	}
	return 4
}
