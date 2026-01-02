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

// handleRoomRequest 处理房间请求
func (h *Handler) handleRoomRequest(_ctx context.Context, conn *connection.Connection, _stream *webtransport.Stream, reqID string, payload []byte) {
	// Room request processing

	// 解析 RoomReq
	roomReq := im_protocol.GetRootAsRoomReq(payload, 0)

	// 封装上行消息到 Logic
	msg := &proto.UpstreamMessage{
		AccessNodeId: h.nodeID,
		RoomRequest: &proto.RoomRequest{
			UserId:     conn.UserID(),
			ReqId:      reqID,
			Action:     roomReq.Action().String(),
			RoomId:     string(roomReq.RoomId()),
			GameType:   roomReq.GameType().String(),
			RoomConfig: string(roomReq.RoomConfig()),
			SeatIndex:  roomReq.TargetSeatIndex(),
		},
	}

	// Forward to logic
	data, _ := json.Marshal(msg)
	if err := h.natsClient.Publish(sharedNats.SubjectLogicUpstream, data); err != nil {
		h.logger.Error("Failed to publish room request to NATS", "error", err)
		return
	}
	// Room request published
}
