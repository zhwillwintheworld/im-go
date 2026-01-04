package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/redis/go-redis/v9"
	"sudooom.im.logic/internal/service"
	"sudooom.im.logic/internal/service/room"
	"sudooom.im.shared/model"
	"sudooom.im.shared/proto"
)

// RoomActionHandler 房间操作处理器接口
type RoomActionHandler interface {
	Handle(ctx context.Context, req *proto.RoomRequest, accessNodeId string, connId int64, platform string) error
}

// RoomHandler 房间请求处理器
type RoomHandler struct {
	actionHandlers map[string]RoomActionHandler
	redisClient    *redis.Client
	roomService    *room.RoomService
	logger         *slog.Logger
}

// NewRoomHandler 创建房间请求处理器
func NewRoomHandler(redisClient *redis.Client, roomService *room.RoomService) *RoomHandler {
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
	h.actionHandlers["JOIN"] = &JoinRoomHandler{
		redisClient:   h.redisClient,
		roomService:   h.roomService,
		routerService: h.roomService.GetRouterService(),
		logger:        h.logger,
	}
	h.actionHandlers["LEAVE"] = &LeaveRoomHandler{roomService: h.roomService, logger: h.logger}
	h.actionHandlers["READY"] = &ReadyRoomHandler{logger: h.logger}
	h.actionHandlers["CHANGE_SEAT"] = &ChangeSeatHandler{logger: h.logger}
	h.actionHandlers["START_GAME"] = &StartGameHandler{logger: h.logger}
}

// Handle 处理房间请求
func (h *RoomHandler) Handle(ctx context.Context, req *proto.RoomRequest, accessNodeId string, connId int64, platform string) error {
	handler, ok := h.actionHandlers[req.Action]
	if !ok {
		// 记录警告日志
		h.logger.Warn("Unknown room action", "action", req.Action, "userId", req.UserId, "reqId", req.ReqId)
		// 后续可以通过 RouterService 发送 RoomPush 错误事件给客户端
		return nil
	}

	return handler.Handle(ctx, req, accessNodeId, connId, platform)
}

// ============================================================================
// 各种房间操作策略实现
// ============================================================================

// CreateRoomHandler 创建房间
type CreateRoomHandler struct {
	roomService *room.RoomService
	logger      *slog.Logger
}

func (h *CreateRoomHandler) Handle(ctx context.Context, req *proto.RoomRequest, accessNodeId string, connId int64, platform string) error {
	h.logger.Info("Create room",
		"userId", req.UserId,
		"reqId", req.ReqId,
		"gameType", req.GameType,
		"roomConfig", req.RoomConfig,
		"accessNodeId", accessNodeId)

	// 创建房间（包含发送响应）
	roomCreate, err := h.roomService.CreateRoom(ctx, req, accessNodeId, connId, platform)
	if err != nil {
		h.logger.Error("Failed to create room", "error", err, "userId", req.UserId)
		// TODO: 发送创建失败响应
		return nil // 不阻塞流程
	}

	h.logger.Info("Room created successfully",
		"roomId", roomCreate.RoomID,
		"roomName", roomCreate.RoomName,
		"userId", req.UserId)
	return nil
}

// JoinRoomHandler 加入房间
type JoinRoomHandler struct {
	redisClient   *redis.Client
	roomService   *room.RoomService
	routerService *service.RouterService
	logger        *slog.Logger
}

func (h *JoinRoomHandler) Handle(ctx context.Context, req *proto.RoomRequest, accessNodeId string, connId int64, platform string) error {
	h.logger.Info("Join room",
		"userId", req.UserId,
		"reqId", req.ReqId,
		"roomId", req.RoomId,
		"seatIndex", req.SeatIndex,
		"accessNodeId", accessNodeId)

	// 调用 Service 层处理业务逻辑
	_, err := h.roomService.JoinRoom(ctx, room.JoinRoomParams{
		UserId:       req.UserId,
		RoomId:       req.RoomId,
		Password:     req.RoomConfig, // RoomConfig 用于传递密码
		SeatIndex:    req.SeatIndex,
		AccessNodeId: accessNodeId,
		ConnId:       connId,
		Platform:     platform,
	})

	if err != nil {
		h.logger.Warn("Failed to join room", "error", err, "userId", req.UserId, "roomId", req.RoomId)
		h.sendErrorResponse(ctx, accessNodeId, connId, platform, req.UserId, req.RoomId, err)
		return nil // 不阻塞消息处理
	}

	return nil
}

// sendErrorResponse 发送错误响应
func (h *JoinRoomHandler) sendErrorResponse(ctx context.Context, accessNodeId string, connId int64, platform string, userId int64, roomId string, err error) {
	// 将 error 映射到错误码和消息
	errorCode, errorMsg := h.mapErrorToCodeAndMsg(err)

	errorInfo := map[string]string{
		"error_code": errorCode,
		"error_msg":  errorMsg,
	}
	errorData, _ := json.Marshal(errorInfo)

	// 构造 sender location
	senderLoc := model.UserLocation{
		AccessNodeId: accessNodeId,
		ConnId:       connId,
		Platform:     platform,
		UserId:       userId,
	}

	// 发送给自己（只需快速响应，错误不需要多端同步）
	if err := h.routerService.SendRoomPushToSelf(senderLoc, "JOIN_ROOM_ERROR", roomId, errorData); err != nil {
		h.logger.Warn("Failed to send error response", "error", err, "userId", userId)
	}
}

// mapErrorToCodeAndMsg 将 error 映射到错误码和错误消息
func (h *JoinRoomHandler) mapErrorToCodeAndMsg(err error) (string, string) {
	switch {
	case errors.Is(err, room.ErrRoomNotFound):
		return "ROOM_NOT_FOUND", "房间已解散"
	case errors.Is(err, room.ErrRoomFull):
		return "ROOM_FULL", "房间已满"
	case errors.Is(err, room.ErrRoomBusy):
		return "ROOM_BUSY", "房间正在处理其他操作"
	case errors.Is(err, room.ErrInvalidPassword):
		return "INVALID_PASSWORD", "房间密码错误"
	case errors.Is(err, room.ErrGameStarted):
		return "GAME_STARTED", "游戏已开始"
	case errors.Is(err, room.ErrAlreadyInRoom):
		return "ALREADY_IN_ROOM", "您已在房间中"
	case errors.Is(err, room.ErrLockFailed):
		return "LOCK_FAILED", "无法获取房间锁"
	default:
		return "JOIN_FAILED", "加入房间失败"
	}
}

// LeaveRoomHandler 离开房间
type LeaveRoomHandler struct {
	roomService *room.RoomService
	logger      *slog.Logger
}

func (h *LeaveRoomHandler) Handle(ctx context.Context, req *proto.RoomRequest, accessNodeId string, connId int64, platform string) error {
	h.logger.Info("Leave room",
		"userId", req.UserId,
		"reqId", req.ReqId,
		"roomId", req.RoomId,
		"accessNodeId", accessNodeId)

	// 调用 Service 层处理离开房间逻辑
	err := h.roomService.LeaveRoom(ctx, room.LeaveRoomParams{
		UserId:       req.UserId,
		RoomId:       req.RoomId,
		AccessNodeId: accessNodeId,
		ConnId:       connId,
		Platform:     platform,
	})

	if err != nil {
		h.logger.Warn("Failed to leave room", "error", err, "userId", req.UserId, "roomId", req.RoomId)
		// 不阻塞消息处理，只记录日志
		return nil
	}

	h.logger.Info("User left room successfully", "userId", req.UserId, "roomId", req.RoomId)
	return nil
}

// ReadyRoomHandler 准备/取消准备
type ReadyRoomHandler struct {
	logger *slog.Logger
}

func (h *ReadyRoomHandler) Handle(ctx context.Context, req *proto.RoomRequest, accessNodeId string, connId int64, platform string) error {
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

func (h *ChangeSeatHandler) Handle(ctx context.Context, req *proto.RoomRequest, accessNodeId string, connId int64, platform string) error {
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

func (h *StartGameHandler) Handle(ctx context.Context, req *proto.RoomRequest, accessNodeId string, connId int64, platform string) error {
	h.logger.Info("Start game",
		"userId", req.UserId,
		"reqId", req.ReqId,
		"roomId", req.RoomId,
		"gameType", req.GameType,
		"accessNodeId", accessNodeId)
	// TODO: 实现开始游戏逻辑
	return nil
}
