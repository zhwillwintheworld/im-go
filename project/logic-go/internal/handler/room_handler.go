package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"sudooom.im.logic/internal/game"
	"sudooom.im.logic/internal/room"
	"sudooom.im.logic/internal/service"
	sharedModel "sudooom.im.shared/model"
	"sudooom.im.shared/proto"
)

// RoomHandler 房间请求处理器
type RoomHandler struct {
	roomService   *room.RoomService
	gameManager   *game.GameManager
	routerService *service.RouterService
	logger        *slog.Logger
}

// NewRoomHandler 创建房间请求处理器
func NewRoomHandler(roomService *room.RoomService, gameManager *game.GameManager, routerService *service.RouterService) *RoomHandler {
	return &RoomHandler{
		roomService:   roomService,
		gameManager:   gameManager,
		routerService: routerService,
		logger:        slog.Default(),
	}
}

// Handle 处理房间请求
func (h *RoomHandler) Handle(ctx context.Context, req *proto.RoomRequest, accessNodeId string, connId int64, platform string) error {
	switch req.Action {
	case "CREATE":
		return h.handleCreate(ctx, req, accessNodeId, connId, platform)
	case "JOIN":
		return h.handleJoin(ctx, req, accessNodeId, connId, platform)
	case "LEAVE":
		return h.handleLeave(ctx, req, accessNodeId, connId, platform)
	case "READY":
		return h.handleReady(ctx, req, accessNodeId, connId, platform)
	case "CHANGE_SEAT":
		return h.handleChangeSeat(ctx, req, accessNodeId, connId, platform)
	case "START_GAME":
		return h.handleStartGame(ctx, req, accessNodeId, connId, platform)
	default:
		h.logger.Warn("Unknown room action", "action", req.Action, "userId", req.UserId, "reqId", req.ReqId)
		return nil
	}
}

// handleCreate 处理创建房间
func (h *RoomHandler) handleCreate(ctx context.Context, req *proto.RoomRequest, accessNodeId string, connId int64, platform string) error {
	h.logger.Info("Create room", "userId", req.UserId, "reqId", req.ReqId, "gameType", req.GameType)

	config, err := h.parseRoomConfig(req.RoomConfig)
	if err != nil {
		h.sendErrorResponse(ctx, accessNodeId, connId, platform, req.UserId, "", err)
		return nil
	}

	roomInfo, err := h.roomService.CreateRoom(ctx, room.CreateRoomParams{
		UserId:       req.UserId,
		RoomName:     config["roomName"],
		RoomType:     config["roomType"],
		RoomPassword: config["roomPassword"],
		MaxPlayers:   h.parseMaxPlayers(config["maxPlayers"]),
		GameType:     req.GameType,
		GameSettings: config,
	})

	if err != nil {
		h.logger.Error("Failed to create room", "error", err, "userId", req.UserId)
		h.sendErrorResponse(ctx, accessNodeId, connId, platform, req.UserId, "", err)
		return nil
	}

	h.logger.Info("Room created successfully", "roomId", roomInfo.RoomID, "roomName", roomInfo.RoomName, "userId", req.UserId)
	return nil
}

// handleJoin 处理加入房间
func (h *RoomHandler) handleJoin(ctx context.Context, req *proto.RoomRequest, accessNodeId string, connId int64, platform string) error {
	h.logger.Info("Join room", "userId", req.UserId, "reqId", req.ReqId, "roomId", req.RoomId)

	_, err := h.roomService.JoinRoom(ctx, room.JoinRoomParams{
		UserId:   req.UserId,
		RoomId:   req.RoomId,
		Password: req.RoomConfig,
	})

	if err != nil {
		h.logger.Warn("Failed to join room", "error", err, "userId", req.UserId, "roomId", req.RoomId)
		h.sendErrorResponse(ctx, accessNodeId, connId, platform, req.UserId, req.RoomId, err)
		return nil
	}

	h.logger.Info("User joined room successfully", "userId", req.UserId, "roomId", req.RoomId)
	return nil
}

// handleLeave 处理离开房间
func (h *RoomHandler) handleLeave(ctx context.Context, req *proto.RoomRequest, accessNodeId string, connId int64, platform string) error {
	h.logger.Info("Leave room", "userId", req.UserId, "reqId", req.ReqId, "roomId", req.RoomId)

	_, err := h.roomService.LeaveRoom(ctx, room.LeaveRoomParams{
		UserId: req.UserId,
		RoomId: req.RoomId,
	})

	if err != nil {
		h.logger.Warn("Failed to leave room", "error", err, "userId", req.UserId, "roomId", req.RoomId)
		return nil
	}

	h.logger.Info("User left room successfully", "userId", req.UserId, "roomId", req.RoomId)
	return nil
}

// handleReady 处理准备/取消准备
func (h *RoomHandler) handleReady(ctx context.Context, req *proto.RoomRequest, accessNodeId string, connId int64, platform string) error {
	h.logger.Info("Toggle ready in room", "userId", req.UserId, "reqId", req.ReqId, "roomId", req.RoomId)

	_, err := h.roomService.ReadyRoom(ctx, room.ReadyRoomParams{
		UserId: req.UserId,
		RoomId: req.RoomId,
	})

	if err != nil {
		h.logger.Warn("Failed to toggle ready", "error", err, "userId", req.UserId, "roomId", req.RoomId)
		h.sendErrorResponse(ctx, accessNodeId, connId, platform, req.UserId, req.RoomId, err)
		return nil
	}

	h.logger.Info("Ready state toggled successfully", "userId", req.UserId, "roomId", req.RoomId)
	return nil
}

// handleChangeSeat 处理换座位
func (h *RoomHandler) handleChangeSeat(ctx context.Context, req *proto.RoomRequest, accessNodeId string, connId int64, platform string) error {
	h.logger.Info("Change seat", "userId", req.UserId, "reqId", req.ReqId, "roomId", req.RoomId, "targetSeatIndex", req.SeatIndex)

	_, err := h.roomService.ChangeSeat(ctx, room.ChangeSeatParams{
		UserId:     req.UserId,
		RoomId:     req.RoomId,
		TargetSeat: req.SeatIndex,
	})

	if err != nil {
		h.logger.Warn("Failed to change seat", "error", err, "userId", req.UserId, "roomId", req.RoomId)
		h.sendErrorResponse(ctx, accessNodeId, connId, platform, req.UserId, req.RoomId, err)
		return nil
	}

	h.logger.Info("Seat changed successfully", "userId", req.UserId, "roomId", req.RoomId, "newSeat", req.SeatIndex)
	return nil
}

// handleStartGame 处理开始游戏
func (h *RoomHandler) handleStartGame(ctx context.Context, req *proto.RoomRequest, accessNodeId string, connId int64, platform string) error {
	h.logger.Info("Start game", "userId", req.UserId, "reqId", req.ReqId, "roomId", req.RoomId, "gameType", req.GameType)

	roomInfo, err := h.roomService.StartGame(ctx, room.StartGameParams{
		UserId: req.UserId,
		RoomId: req.RoomId,
	})

	if err != nil {
		h.logger.Warn("Failed to start game", "error", err, "userId", req.UserId, "roomId", req.RoomId)
		h.sendErrorResponse(ctx, accessNodeId, connId, platform, req.UserId, req.RoomId, err)
		return nil
	}

	if err := h.gameManager.StartGame(ctx, roomInfo); err != nil {
		h.logger.Warn("Failed to initialize game", "error", err, "roomId", req.RoomId)
		h.sendErrorResponse(ctx, accessNodeId, connId, platform, req.UserId, req.RoomId, err)
		return nil
	}

	h.logger.Info("Game started successfully", "userId", req.UserId, "roomId", req.RoomId, "gameType", roomInfo.GameType)
	return nil
}

// HandleGameRequest 处理游戏请求
func (h *RoomHandler) HandleGameRequest(ctx context.Context, req *proto.GameRequest, accessNodeId string, connId int64, platform string) error {
	h.logger.Info("Game request received", "userId", req.UserId, "reqId", req.ReqId, "roomId", req.RoomId, "gameType", req.GameType)

	switch req.GameType {
	case game.GameTypeHTMahjong:
		return h.handleMahjongGame(ctx, req, accessNodeId, connId, platform)
	default:
		h.logger.Warn("Unknown game type", "gameType", req.GameType)
		return nil
	}
}

// handleMahjongGame 处理麻将游戏请求
func (h *RoomHandler) handleMahjongGame(ctx context.Context, req *proto.GameRequest, accessNodeId string, connId int64, platform string) error {
	h.logger.Debug("Handling mahjong game request", "userId", req.UserId, "roomId", req.RoomId, "payloadSize", len(req.GamePayload))
	// TODO: 实现麻将游戏逻辑
	return nil
}

// sendErrorResponse 发送错误响应
func (h *RoomHandler) sendErrorResponse(ctx context.Context, accessNodeId string, connId int64, platform string, userId int64, roomId string, err error) {
	errorResp := h.mapError(err, roomId)
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

// mapError 映射错误到错误响应
func (h *RoomHandler) mapError(err error, roomId string) map[string]interface{} {
	if err == nil {
		return map[string]interface{}{"code": "UNKNOWN_ERROR", "message": "未知错误", "roomId": roomId}
	}

	switch {
	case errors.Is(err, room.ErrRoomNotFound):
		return map[string]interface{}{"code": "ROOM_NOT_FOUND", "message": "房间不存在", "roomId": roomId}
	case errors.Is(err, room.ErrRoomFull):
		return map[string]interface{}{"code": "ROOM_FULL", "message": "房间已满", "roomId": roomId}
	case errors.Is(err, room.ErrRoomBusy):
		return map[string]interface{}{"code": "ROOM_BUSY", "message": "房间繁忙", "roomId": roomId}
	case errors.Is(err, room.ErrInvalidPassword):
		return map[string]interface{}{"code": "INVALID_PASSWORD", "message": "密码错误", "roomId": roomId}
	case errors.Is(err, room.ErrGameStarted):
		return map[string]interface{}{"code": "GAME_STARTED", "message": "游戏已开始", "roomId": roomId}
	case errors.Is(err, room.ErrAlreadyInRoom):
		return map[string]interface{}{"code": "ALREADY_IN_ROOM", "message": "已在房间中", "roomId": roomId}
	case errors.Is(err, room.ErrLockFailed):
		return map[string]interface{}{"code": "LOCK_FAILED", "message": "获取锁失败", "roomId": roomId}
	case errors.Is(err, room.ErrNotRoomHost):
		return map[string]interface{}{"code": "NOT_ROOM_HOST", "message": "不是房主", "roomId": roomId}
	case errors.Is(err, room.ErrNotAllReady):
		return map[string]interface{}{"code": "NOT_ALL_READY", "message": "玩家未全部准备", "roomId": roomId}
	case errors.Is(err, room.ErrUnsupportedGameType):
		return map[string]interface{}{"code": "UNSUPPORTED_GAME_TYPE", "message": "不支持的游戏类型", "roomId": roomId}
	case errors.Is(err, room.ErrSeatOccupied):
		return map[string]interface{}{"code": "SEAT_OCCUPIED", "message": "座位已被占用", "roomId": roomId}
	case errors.Is(err, room.ErrInvalidSeat):
		return map[string]interface{}{"code": "INVALID_SEAT", "message": "无效的座位", "roomId": roomId}
	case errors.Is(err, room.ErrNotInRoom):
		return map[string]interface{}{"code": "NOT_IN_ROOM", "message": "不在房间中", "roomId": roomId}
	case errors.Is(err, room.ErrNotEnoughPlayers):
		return map[string]interface{}{"code": "NOT_ENOUGH_PLAYERS", "message": "玩家人数不足", "roomId": roomId}
	default:
		return map[string]interface{}{"code": "OPERATION_FAILED", "message": err.Error(), "roomId": roomId}
	}
}

// parseRoomConfig 解析房间配置
func (h *RoomHandler) parseRoomConfig(configStr string) (map[string]string, error) {
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

// parseMaxPlayers 解析最大玩家数
func (h *RoomHandler) parseMaxPlayers(maxPlayersStr string) int {
	if maxPlayersStr == "" {
		return 4
	}
	if n, err := json.Number(maxPlayersStr).Int64(); err == nil && n > 0 {
		return int(n)
	}
	return 4
}
