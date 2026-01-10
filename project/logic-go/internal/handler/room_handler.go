package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/redis/go-redis/v9"
	"sudooom.im.logic/internal/game"
	"sudooom.im.logic/internal/room"
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
	gameService    *game.GameService
	logger         *slog.Logger
}

// NewRoomHandler 创建房间请求处理器
func NewRoomHandler(redisClient *redis.Client, roomService *room.RoomService, gameService *game.GameService) *RoomHandler {
	h := &RoomHandler{
		actionHandlers: make(map[string]RoomActionHandler),
		redisClient:    redisClient,
		roomService:    roomService,
		gameService:    gameService,
		logger:         slog.Default(),
	}

	// 注册各种房间操作处理器
	h.registerActionHandlers()

	return h
}

// registerActionHandlers 注册各种房间操作处理器
func (h *RoomHandler) registerActionHandlers() {
	h.actionHandlers["CREATE"] = &CreateRoomHandler{
		roomService: h.roomService,
		logger:      h.logger,
	}
	h.actionHandlers["JOIN"] = &JoinRoomHandler{
		roomService: h.roomService,
		logger:      h.logger,
	}
	h.actionHandlers["LEAVE"] = &LeaveRoomHandler{
		roomService: h.roomService,
		logger:      h.logger,
	}
	h.actionHandlers["READY"] = &ReadyRoomHandler{
		roomService: h.roomService,
		logger:      h.logger,
	}
	h.actionHandlers["CHANGE_SEAT"] = &ChangeSeatHandler{
		roomService: h.roomService,
		logger:      h.logger,
	}
	h.actionHandlers["START_GAME"] = &StartGameHandler{
		roomService: h.roomService,
		gameService: h.gameService,
		logger:      h.logger,
	}
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

	// 解析房间配置
	var config map[string]string
	if req.RoomConfig != "" {
		if err := json.Unmarshal([]byte(req.RoomConfig), &config); err != nil {
			h.logger.Error("Failed to parse room config", "error", err, "userId", req.UserId)
			// TODO: 发送创建失败响应
			return nil // 不阻塞流程
		}
	}
	if config == nil {
		config = make(map[string]string)
	}

	// 提取房间配置参数
	roomName := config["roomName"]
	if roomName == "" {
		roomName = "房间" // 默认房间名
	}
	roomType := config["roomType"]
	if roomType == "" {
		roomType = "NORMAL" // 默认房间类型
	}
	roomPassword := config["roomPassword"]
	maxPlayers := 4 // 默认最大玩家数
	if maxPlayersStr, ok := config["maxPlayers"]; ok && maxPlayersStr != "" {
		if n, err := json.Number(maxPlayersStr).Int64(); err == nil {
			maxPlayers = int(n)
		}
	}

	// 创建房间
	roomCreate, err := h.roomService.CreateRoom(ctx, room.CreateRoomParams{
		UserId:       req.UserId,
		RoomName:     roomName,
		RoomType:     roomType,
		RoomPassword: roomPassword,
		MaxPlayers:   maxPlayers,
		GameType:     req.GameType,
		GameSettings: config,
	})
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
	roomService *room.RoomService
	logger      *slog.Logger
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
		UserId:    req.UserId,
		RoomId:    req.RoomId,
		Password:  req.RoomConfig, // RoomConfig 用于传递密码
		SeatIndex: req.SeatIndex,
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

	// TODO: 发送错误响应给客户端
	// 错误响应暂时通过日志记录，后续可以通过 RouterService 发送
	h.logger.Warn("Join room error", "userId", userId, "roomId", roomId, "errorCode", errorCode, "errorMsg", errorMsg)
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
	_, err := h.roomService.LeaveRoom(ctx, room.LeaveRoomParams{
		UserId: req.UserId,
		RoomId: req.RoomId,
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
	roomService *room.RoomService
	logger      *slog.Logger
}

func (h *ReadyRoomHandler) Handle(ctx context.Context, req *proto.RoomRequest, accessNodeId string, connId int64, platform string) error {
	h.logger.Info("Toggle ready in room",
		"userId", req.UserId,
		"reqId", req.ReqId,
		"roomId", req.RoomId,
		"accessNodeId", accessNodeId)

	// 调用 Service 层处理准备状态切换
	_, err := h.roomService.ReadyRoom(ctx, room.ReadyRoomParams{
		UserId: req.UserId,
		RoomId: req.RoomId,
	})

	if err != nil {
		h.logger.Warn("Failed to toggle ready", "error", err, "userId", req.UserId, "roomId", req.RoomId)
		// 不阻塞消息处理，只记录日志
		return nil
	}

	h.logger.Info("Ready state toggled successfully", "userId", req.UserId, "roomId", req.RoomId)
	return nil
}

// ChangeSeatHandler 换座位
type ChangeSeatHandler struct {
	roomService *room.RoomService
	logger      *slog.Logger
}

func (h *ChangeSeatHandler) Handle(ctx context.Context, req *proto.RoomRequest, accessNodeId string, connId int64, platform string) error {
	h.logger.Info("Change seat",
		"userId", req.UserId,
		"reqId", req.ReqId,
		"roomId", req.RoomId,
		"targetSeatIndex", req.SeatIndex,
		"accessNodeId", accessNodeId)

	// 调用 Service 层处理换座位
	_, err := h.roomService.ChangeSeat(ctx, room.ChangeSeatParams{
		UserId:     req.UserId,
		RoomId:     req.RoomId,
		TargetSeat: req.SeatIndex,
	})

	if err != nil {
		h.logger.Warn("Failed to change seat", "error", err, "userId", req.UserId, "roomId", req.RoomId)
		// 不阻塞消息处理，只记录日志
		return nil
	}

	h.logger.Info("Seat changed successfully", "userId", req.UserId, "roomId", req.RoomId, "newSeat", req.SeatIndex)
	return nil
}

// StartGameHandler 开始游戏
type StartGameHandler struct {
	roomService *room.RoomService
	gameService *game.GameService
	logger      *slog.Logger
}

func (h *StartGameHandler) Handle(ctx context.Context, req *proto.RoomRequest, accessNodeId string, connId int64, platform string) error {
	h.logger.Info("Start game",
		"userId", req.UserId,
		"reqId", req.ReqId,
		"roomId", req.RoomId,
		"gameType", req.GameType,
		"accessNodeId", accessNodeId)

	// 调用 Service 层进行验证并更新房间状态
	room, err := h.roomService.StartGame(ctx, room.StartGameParams{
		UserId: req.UserId,
		RoomId: req.RoomId,
	})

	if err != nil {
		h.logger.Warn("Failed to start game", "error", err, "userId", req.UserId, "roomId", req.RoomId)
		// 不阻塞消息处理，只记录日志
		return nil
	}

	// 调用 GameService 启动游戏（初始化游戏状态并广播）
	if err := h.gameService.StartGame(ctx, room); err != nil {
		h.logger.Warn("Failed to initialize game", "error", err, "roomId", req.RoomId)
		// 不阻塞消息处理，只记录日志
		return nil
	}

	h.logger.Info("Game started successfully", "userId", req.UserId, "roomId", req.RoomId, "gameType", room.GameType)
	return nil
}
