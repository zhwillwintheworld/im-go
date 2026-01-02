package handler

import (
	"context"
	"log/slog"

	"github.com/redis/go-redis/v9"
	"sudooom.im.logic/internal/service"
	"sudooom.im.shared/proto"
)

// RoomActionHandler 房间操作处理器接口
type RoomActionHandler interface {
	Handle(ctx context.Context, req *proto.RoomRequest, accessNodeId string) error
}

// RoomHandler 房间请求处理器
type RoomHandler struct {
	actionHandlers map[string]RoomActionHandler
	redisClient    *redis.Client
	roomService    *service.RoomService
	logger         *slog.Logger
}

// NewRoomHandler 创建房间请求处理器
func NewRoomHandler(redisClient *redis.Client, roomService *service.RoomService) *RoomHandler {
	h := &RoomHandler{
		actionHandlers: make(map[string]RoomActionHandler),
		redisClient:    redisClient,
		roomService:    roomService,
		logger:         slog.Default(),
	}

	// 注册各种房间操作处理器
	h.registerActionHandlers()

	return h
}

// registerActionHandlers 注册各种房间操作处理器
func (h *RoomHandler) registerActionHandlers() {
	h.actionHandlers["CREATE"] = &CreateRoomHandler{roomService: h.roomService, logger: h.logger}
	h.actionHandlers["JOIN"] = &JoinRoomHandler{logger: h.logger}
	h.actionHandlers["LEAVE"] = &LeaveRoomHandler{logger: h.logger}
	h.actionHandlers["READY"] = &ReadyRoomHandler{logger: h.logger}
	h.actionHandlers["CHANGE_SEAT"] = &ChangeSeatHandler{logger: h.logger}
	h.actionHandlers["START_GAME"] = &StartGameHandler{logger: h.logger}
}

// Handle 处理房间请求
func (h *RoomHandler) Handle(ctx context.Context, req *proto.RoomRequest, accessNodeId string) error {
	handler, ok := h.actionHandlers[req.Action]
	if !ok {
		// 记录警告日志
		h.logger.Warn("Unknown room action", "action", req.Action, "userId", req.UserId, "reqId", req.ReqId)
		// TODO: 异步发送错误响应给客户端
		// 这里不返回错误，避免阻塞消息处理流程
		// 后续可以通过 RouterService 发送 RoomPush 错误事件给客户端
		return nil
	}

	return handler.Handle(ctx, req, accessNodeId)
}

// ============================================================================
// 各种房间操作策略实现
// ============================================================================

// CreateRoomHandler 创建房间
type CreateRoomHandler struct {
	roomService *service.RoomService
	logger      *slog.Logger
}

func (h *CreateRoomHandler) Handle(ctx context.Context, req *proto.RoomRequest, accessNodeId string) error {
	h.logger.Info("Create room",
		"userId", req.UserId,
		"reqId", req.ReqId,
		"gameType", req.GameType,
		"roomConfig", req.RoomConfig,
		"accessNodeId", accessNodeId)

	// 1. 创建房间
	room, err := h.roomService.CreateRoom(ctx, req, accessNodeId)
	if err != nil {
		h.logger.Error("Failed to create room", "error", err, "userId", req.UserId)
		// TODO: 发送错误响应给客户端
		return nil // 不阻塞流程
	}

	// 2. 发送房间创建成功响应
	if err := h.roomService.SendRoomCreatedResponse(ctx, room, accessNodeId); err != nil {
		h.logger.Error("Failed to send room created response", "error", err, "roomId", room.RoomID)
	}

	h.logger.Info("Room created and response sent",
		"roomId", room.RoomID,
		"roomName", room.RoomName,
		"userId", req.UserId)

	return nil
}

// JoinRoomHandler 加入房间
type JoinRoomHandler struct {
	logger *slog.Logger
}

func (h *JoinRoomHandler) Handle(ctx context.Context, req *proto.RoomRequest, accessNodeId string) error {
	h.logger.Info("Join room",
		"userId", req.UserId,
		"reqId", req.ReqId,
		"roomId", req.RoomId,
		"seatIndex", req.SeatIndex,
		"accessNodeId", accessNodeId)
	// TODO: 实现加入房间逻辑
	return nil
}

// LeaveRoomHandler 离开房间
type LeaveRoomHandler struct {
	logger *slog.Logger
}

func (h *LeaveRoomHandler) Handle(ctx context.Context, req *proto.RoomRequest, accessNodeId string) error {
	h.logger.Info("Leave room",
		"userId", req.UserId,
		"reqId", req.ReqId,
		"roomId", req.RoomId,
		"accessNodeId", accessNodeId)
	// TODO: 实现离开房间逻辑
	return nil
}

// ReadyRoomHandler 准备/取消准备
type ReadyRoomHandler struct {
	logger *slog.Logger
}

func (h *ReadyRoomHandler) Handle(ctx context.Context, req *proto.RoomRequest, accessNodeId string) error {
	h.logger.Info("Ready in room",
		"userId", req.UserId,
		"reqId", req.ReqId,
		"roomId", req.RoomId,
		"accessNodeId", accessNodeId)
	// TODO: 实现准备逻辑
	return nil
}

// ChangeSeatHandler 换座位
type ChangeSeatHandler struct {
	logger *slog.Logger
}

func (h *ChangeSeatHandler) Handle(ctx context.Context, req *proto.RoomRequest, accessNodeId string) error {
	h.logger.Info("Change seat",
		"userId", req.UserId,
		"reqId", req.ReqId,
		"roomId", req.RoomId,
		"targetSeatIndex", req.SeatIndex,
		"accessNodeId", accessNodeId)
	// TODO: 实现换座位逻辑
	return nil
}

// StartGameHandler 开始游戏
type StartGameHandler struct {
	logger *slog.Logger
}

func (h *StartGameHandler) Handle(ctx context.Context, req *proto.RoomRequest, accessNodeId string) error {
	h.logger.Info("Start game",
		"userId", req.UserId,
		"reqId", req.ReqId,
		"roomId", req.RoomId,
		"gameType", req.GameType,
		"accessNodeId", accessNodeId)
	// TODO: 实现开始游戏逻辑
	return nil
}
