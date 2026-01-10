package handler

import (
	"context"
	"fmt"
	"log/slog"

	"sudooom.im.logic/internal/game"
	"sudooom.im.shared/proto"
)

// GameHandler 游戏请求处理器
type GameHandler struct {
	gameService *game.GameService
	logger      *slog.Logger
}

// NewGameHandler 创建游戏请求处理器
func NewGameHandler(gameService *game.GameService) *GameHandler {
	return &GameHandler{
		gameService: gameService,
		logger:      slog.Default(),
	}
}

// Handle 处理游戏请求
func (h *GameHandler) Handle(ctx context.Context, req *proto.GameRequest, accessNodeId string, connId int64, platform string) error {
	h.logger.Info("Game request received",
		"userId", req.UserId,
		"reqId", req.ReqId,
		"roomId", req.RoomId,
		"gameType", req.GameType,
		"accessNodeId", accessNodeId)

	// 根据游戏类型分发
	switch req.GameType {
	case "HT_MAHJONG":
		return h.handleMahjongGame(ctx, req, accessNodeId, connId, platform)
	default:
		h.logger.Warn("Unknown game type", "gameType", req.GameType)
		return fmt.Errorf("unknown game type: %s", req.GameType)
	}
}

// handleMahjongGame 处理麻将游戏请求
func (h *GameHandler) handleMahjongGame(ctx context.Context, req *proto.GameRequest, accessNodeId string, connId int64, platform string) error {
	h.logger.Debug("Handling mahjong game request",
		"userId", req.UserId,
		"roomId", req.RoomId,
		"payloadSize", len(req.GamePayload))

	// TODO: 实现麻将游戏逻辑
	// 1. 解析 GamePayload (FlatBuffers MahjongReq)
	// 2. 处理游戏动作 (出牌、碰、杠、胡等)
	// 3. 更新游戏状态
	// 4. 广播游戏状态给房间所有玩家

	return nil
}
