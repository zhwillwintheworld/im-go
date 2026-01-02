package handler

import (
	"context"
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/quic-go/webtransport-go"
	"sudooom.im.access/internal/connection"
	im_protocol "sudooom.im.access/pkg/flatbuf/im/protocol"
)

// handleHeartbeat 处理心跳请求
func (h *Handler) handleHeartbeat(ctx context.Context, conn *connection.Connection, stream *webtransport.Stream, reqID string, _payload []byte) {
	// 刷新用户位置 TTL
	if conn.UserID() > 0 {
		if err := h.redisClient.RefreshUserLocation(ctx, conn.UserID(), conn.Platform()); err != nil {
			h.logger.Error("Failed to refresh user location", "error", err)
		}
	}
	// 构建 HeartbeatResp payload
	builder := flatbuffers.NewBuilder(64)
	im_protocol.HeartbeatRespStart(builder)
	im_protocol.HeartbeatRespAddServerTime(builder, time.Now().UnixMilli())
	respOffset := im_protocol.HeartbeatRespEnd(builder)
	builder.Finish(respOffset)
	respPayload := builder.FinishedBytes()

	h.sendClientResponse(stream, reqID, im_protocol.ErrorCodeSUCCESS, "", im_protocol.ResponsePayloadHeartbeatResp, respPayload)
}
