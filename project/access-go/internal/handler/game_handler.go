package handler

import (
	"context"
	"encoding/json"

	"github.com/quic-go/webtransport-go"
	"sudooom.im.access/internal/connection"
	im_protocol "sudooom.im.access/pkg/flatbuf/im/protocol"
	sharedNats "sudooom.im.shared/nats"
	"sudooom.im.shared/proto"
)

// handleGameRequest 处理游戏请求
func (h *Handler) handleGameRequest(_ctx context.Context, conn *connection.Connection, _stream *webtransport.Stream, reqID string, payload []byte) {
	// Game request processing

	// 解析 GameReq
	gameReq := im_protocol.GetRootAsGameReq(payload, 0)

	// 封装上行消息到 Logic
	msg := &proto.UpstreamMessage{
		AccessNodeId: h.nodeID,
		GameRequest: &proto.GameRequest{
			UserId:      conn.UserID(),
			ReqId:       reqID,
			RoomId:      string(gameReq.RoomId()),
			GameType:    gameReq.GameType().String(),
			GamePayload: gameReq.GamePayloadBytes(),
		},
	}

	// Forward to logic
	data, _ := json.Marshal(msg)
	if err := h.natsClient.Publish(sharedNats.SubjectLogicUpstream, data); err != nil {
		h.logger.Error("Failed to publish game request to NATS", "error", err)
		return
	}
	// Game request published
}
