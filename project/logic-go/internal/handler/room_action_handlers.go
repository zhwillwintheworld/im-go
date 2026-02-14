package handler

import (
	"context"

	"sudooom.im.logic/internal/game"
	"sudooom.im.logic/internal/room"
	"sudooom.im.shared/proto"
)

// CreateRoomHandler 创建房间处理器
type CreateRoomHandler struct {
	*BaseRoomHandler
}

// NewCreateRoomHandler 创建处理器
func NewCreateRoomHandler(base *BaseRoomHandler) *CreateRoomHandler {
	return &CreateRoomHandler{BaseRoomHandler: base}
}

// Handle 处理创建房间请求
func (h *CreateRoomHandler) Handle(ctx context.Context, req *proto.RoomRequest, accessNodeId string, connId int64, platform string) error {
	h.logger.Info("Create room",
		"userId", req.UserId,
		"reqId", req.ReqId,
		"gameType", req.GameType,
		"accessNodeId", accessNodeId)

	// 解析配置
	config, err := h.ParseRoomConfig(req.RoomConfig)
	if err != nil {
		h.SendErrorResponse(ctx, accessNodeId, connId, platform, req.UserId, "", err)
		return nil
	}

	// 创建房间
	roomInfo, err := h.roomService.CreateRoom(ctx, room.CreateRoomParams{
		UserId:       req.UserId,
		RoomName:     config["roomName"],
		RoomType:     config["roomType"],
		RoomPassword: config["roomPassword"],
		MaxPlayers:   h.ParseMaxPlayers(config["maxPlayers"]),
		GameType:     req.GameType,
		GameSettings: config,
	})

	if err != nil {
		h.logger.Error("Failed to create room", "error", err, "userId", req.UserId)
		h.SendErrorResponse(ctx, accessNodeId, connId, platform, req.UserId, "", err)
		return nil
	}

	h.logger.Info("Room created successfully",
		"roomId", roomInfo.RoomID,
		"roomName", roomInfo.RoomName,
		"userId", req.UserId)

	return nil
}

// JoinRoomHandler 加入房间处理器
type JoinRoomHandler struct {
	*BaseRoomHandler
}

// NewJoinRoomHandler 创建处理器
func NewJoinRoomHandler(base *BaseRoomHandler) *JoinRoomHandler {
	return &JoinRoomHandler{BaseRoomHandler: base}
}

// Handle 处理加入房间请求
func (h *JoinRoomHandler) Handle(ctx context.Context, req *proto.RoomRequest, accessNodeId string, connId int64, platform string) error {
	h.logger.Info("Join room",
		"userId", req.UserId,
		"reqId", req.ReqId,
		"roomId", req.RoomId,
		"accessNodeId", accessNodeId)

	_, err := h.roomService.JoinRoom(ctx, room.JoinRoomParams{
		UserId:   req.UserId,
		RoomId:   req.RoomId,
		Password: req.RoomConfig, // RoomConfig 用于传递密码
	})

	if err != nil {
		h.logger.Warn("Failed to join room", "error", err, "userId", req.UserId, "roomId", req.RoomId)
		h.SendErrorResponse(ctx, accessNodeId, connId, platform, req.UserId, req.RoomId, err)
		return nil
	}

	h.logger.Info("User joined room successfully", "userId", req.UserId, "roomId", req.RoomId)
	return nil
}

// LeaveRoomHandler 离开房间处理器
type LeaveRoomHandler struct {
	*BaseRoomHandler
}

// NewLeaveRoomHandler 创建处理器
func NewLeaveRoomHandler(base *BaseRoomHandler) *LeaveRoomHandler {
	return &LeaveRoomHandler{BaseRoomHandler: base}
}

// Handle 处理离开房间请求
func (h *LeaveRoomHandler) Handle(ctx context.Context, req *proto.RoomRequest, accessNodeId string, connId int64, platform string) error {
	h.logger.Info("Leave room",
		"userId", req.UserId,
		"reqId", req.ReqId,
		"roomId", req.RoomId,
		"accessNodeId", accessNodeId)

	_, err := h.roomService.LeaveRoom(ctx, room.LeaveRoomParams{
		UserId: req.UserId,
		RoomId: req.RoomId,
	})

	if err != nil {
		h.logger.Warn("Failed to leave room", "error", err, "userId", req.UserId, "roomId", req.RoomId)
		// 离开房间失败不发送错误响应，只记录日志
		return nil
	}

	h.logger.Info("User left room successfully", "userId", req.UserId, "roomId", req.RoomId)
	return nil
}

// ReadyRoomHandler 准备/取消准备处理器
type ReadyRoomHandler struct {
	*BaseRoomHandler
}

// NewReadyRoomHandler 创建处理器
func NewReadyRoomHandler(base *BaseRoomHandler) *ReadyRoomHandler {
	return &ReadyRoomHandler{BaseRoomHandler: base}
}

// Handle 处理准备状态切换请求
func (h *ReadyRoomHandler) Handle(ctx context.Context, req *proto.RoomRequest, accessNodeId string, connId int64, platform string) error {
	h.logger.Info("Toggle ready in room",
		"userId", req.UserId,
		"reqId", req.ReqId,
		"roomId", req.RoomId,
		"accessNodeId", accessNodeId)

	_, err := h.roomService.ReadyRoom(ctx, room.ReadyRoomParams{
		UserId: req.UserId,
		RoomId: req.RoomId,
	})

	if err != nil {
		h.logger.Warn("Failed to toggle ready", "error", err, "userId", req.UserId, "roomId", req.RoomId)
		h.SendErrorResponse(ctx, accessNodeId, connId, platform, req.UserId, req.RoomId, err)
		return nil
	}

	h.logger.Info("Ready state toggled successfully", "userId", req.UserId, "roomId", req.RoomId)
	return nil
}

// ChangeSeatHandler 换座位处理器
type ChangeSeatHandler struct {
	*BaseRoomHandler
}

// NewChangeSeatHandler 创建处理器
func NewChangeSeatHandler(base *BaseRoomHandler) *ChangeSeatHandler {
	return &ChangeSeatHandler{BaseRoomHandler: base}
}

// Handle 处理换座位请求
func (h *ChangeSeatHandler) Handle(ctx context.Context, req *proto.RoomRequest, accessNodeId string, connId int64, platform string) error {
	h.logger.Info("Change seat",
		"userId", req.UserId,
		"reqId", req.ReqId,
		"roomId", req.RoomId,
		"targetSeatIndex", req.SeatIndex,
		"accessNodeId", accessNodeId)

	_, err := h.roomService.ChangeSeat(ctx, room.ChangeSeatParams{
		UserId:     req.UserId,
		RoomId:     req.RoomId,
		TargetSeat: req.SeatIndex,
	})

	if err != nil {
		h.logger.Warn("Failed to change seat", "error", err, "userId", req.UserId, "roomId", req.RoomId)
		h.SendErrorResponse(ctx, accessNodeId, connId, platform, req.UserId, req.RoomId, err)
		return nil
	}

	h.logger.Info("Seat changed successfully", "userId", req.UserId, "roomId", req.RoomId, "newSeat", req.SeatIndex)
	return nil
}

// StartGameHandler 开始游戏处理器
type StartGameHandler struct {
	*BaseRoomHandler
	gameManager *game.GameManager
}

// NewStartGameHandler 创建处理器
func NewStartGameHandler(base *BaseRoomHandler, gameManager *game.GameManager) *StartGameHandler {
	return &StartGameHandler{
		BaseRoomHandler: base,
		gameManager:     gameManager,
	}
}

// Handle 处理开始游戏请求
func (h *StartGameHandler) Handle(ctx context.Context, req *proto.RoomRequest, accessNodeId string, connId int64, platform string) error {
	h.logger.Info("Start game",
		"userId", req.UserId,
		"reqId", req.ReqId,
		"roomId", req.RoomId,
		"gameType", req.GameType,
		"accessNodeId", accessNodeId)

	// 调用 RoomService 验证并更新房间状态
	roomInfo, err := h.roomService.StartGame(ctx, room.StartGameParams{
		UserId: req.UserId,
		RoomId: req.RoomId,
	})

	if err != nil {
		h.logger.Warn("Failed to start game", "error", err, "userId", req.UserId, "roomId", req.RoomId)
		h.SendErrorResponse(ctx, accessNodeId, connId, platform, req.UserId, req.RoomId, err)
		return nil
	}

	// 调用 GameManager 启动游戏（初始化游戏状态并广播）
	if err := h.gameManager.StartGame(ctx, roomInfo); err != nil {
		h.logger.Warn("Failed to initialize game", "error", err, "roomId", req.RoomId)
		h.SendErrorResponse(ctx, accessNodeId, connId, platform, req.UserId, req.RoomId, err)
		return nil
	}

	h.logger.Info("Game started successfully", "userId", req.UserId, "roomId", req.RoomId, "gameType", roomInfo.GameType)
	return nil
}
