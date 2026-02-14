package handler

import (
	"context"
	"log/slog"

	"sudooom.im.logic/internal/game"
	"sudooom.im.logic/internal/room"
	"sudooom.im.logic/internal/service"
	"sudooom.im.shared/proto"
)

// RoomActionHandler 房间操作处理器接口
type RoomActionHandler interface {
	Handle(ctx context.Context, req *proto.RoomRequest, accessNodeId string, connId int64, platform string) error
}

// RoomHandler 房间请求处理器
// 使用策略模式分发不同的房间操作
type RoomHandler struct {
	actionHandlers map[string]RoomActionHandler
	logger         *slog.Logger
}

// NewRoomHandler 创建房间请求处理器
func NewRoomHandler(roomService *room.RoomService, gameManager *game.GameManager, routerService *service.RouterService) *RoomHandler {
	h := &RoomHandler{
		actionHandlers: make(map[string]RoomActionHandler),
		logger:         slog.Default(),
	}

	// 创建基类（包含通用方法）
	base := NewBaseRoomHandler(roomService, routerService)

	// 注册各种房间操作处理器（使用组合模式）
	h.actionHandlers["CREATE"] = NewCreateRoomHandler(base)
	h.actionHandlers["JOIN"] = NewJoinRoomHandler(base)
	h.actionHandlers["LEAVE"] = NewLeaveRoomHandler(base)
	h.actionHandlers["READY"] = NewReadyRoomHandler(base)
	h.actionHandlers["CHANGE_SEAT"] = NewChangeSeatHandler(base)
	h.actionHandlers["START_GAME"] = NewStartGameHandler(base, gameManager)

	return h
}

// Handle 处理房间请求
func (h *RoomHandler) Handle(ctx context.Context, req *proto.RoomRequest, accessNodeId string, connId int64, platform string) error {
	handler, ok := h.actionHandlers[req.Action]
	if !ok {
		h.logger.Warn("Unknown room action", "action", req.Action, "userId", req.UserId, "reqId", req.ReqId)
		return nil
	}

	return handler.Handle(ctx, req, accessNodeId, connId, platform)
}
