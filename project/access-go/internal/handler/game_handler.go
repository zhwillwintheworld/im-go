package handler

import (
	"context"

	"sudooom.im.access/internal/connection"
	im_protocol "sudooom.im.access/pkg/flatbuf/im/protocol"
	"sudooom.im.shared/proto"
)

// handleGameRequest 处理游戏请求
func (h *Handler) handleGameRequest(_ctx context.Context, conn *connection.Connection, reqID string, payload []byte) {
	// Game request processing

	// 解析 GameReq
	gameReq := im_protocol.GetRootAsGameReq(payload, 0)

	// 封装上行消息到 Logic
	msg := h.buildUpstreamMessage(conn, proto.UpstreamPayload{
		GameRequest: &proto.GameRequest{
			UserId:      conn.UserID(),
			ReqId:       reqID,
			RoomId:      string(gameReq.RoomId()),
			GameType:    gameReq.GameType().String(),
			GamePayload: gameReq.GamePayloadBytes(),
		},
	})

	if err := h.publishUpstream(msg); err != nil {
		h.logger.Error("Failed to publish game request to NATS", "error", err)
	}
	// Game request published
}
