package handler

import (
	"context"
	"strconv"

	"github.com/quic-go/webtransport-go"
	"sudooom.im.access/internal/connection"
	im_protocol "sudooom.im.access/pkg/flatbuf/im/protocol"
	"sudooom.im.shared/proto"
)

// handleChatSend 处理聊天发送请求
func (h *Handler) handleChatSend(_ctx context.Context, conn *connection.Connection, _stream *webtransport.Stream, reqID string, payload []byte) {
	// Chat send request

	// 解析 ChatSendReq
	chatReq := im_protocol.GetRootAsChatSendReq(payload, 0)

	// 解析 targetId 为 int64
	targetIdStr := string(chatReq.TargetId())
	targetId, _ := strconv.ParseInt(targetIdStr, 10, 64)

	// 封装上行消息到 Logic
	msg := h.buildUpstreamMessage(conn, proto.UpstreamPayload{
		UserMessage: &proto.UserMessage{
			FromUserId:  conn.UserID(),
			ClientMsgId: reqID,
			MsgType:     int32(chatReq.MsgType()),
			Content:     chatReq.Content(),
			Timestamp:   0,
		},
	})

	// 根据 ChatType 设置目标
	switch chatReq.ChatType() {
	case im_protocol.ChatTypePRIVATE:
		msg.Payload.UserMessage.ToUserId = targetId
	case im_protocol.ChatTypeGROUP:
		msg.Payload.UserMessage.ToGroupId = targetId
	}

	// Forward to logic
	if err := h.publishUpstream(msg); err != nil {
		h.logger.Error("Failed to publish to NATS", "error", err)
		return
	}
	// Message published
}

// handleConversationRead 处理会话已读请求
func (h *Handler) handleConversationRead(conn *connection.Connection, stream *webtransport.Stream, reqID string, payload []byte) {
	// Conversation read request

	// 解析 ConversationReadReq
	readReq := im_protocol.GetRootAsConversationReadReq(payload, 0)

	// 解析 ID
	peerIdStr := string(readReq.PeerId())
	groupIdStr := string(readReq.GroupId())
	lastReadMsgIdStr := string(readReq.LastReadMsgId())

	peerId, _ := strconv.ParseInt(peerIdStr, 10, 64)
	groupId, _ := strconv.ParseInt(groupIdStr, 10, 64)
	lastReadMsgId, _ := strconv.ParseInt(lastReadMsgIdStr, 10, 64)

	// 封装上行消息到 Logic
	msg := h.buildUpstreamMessage(conn, proto.UpstreamPayload{
		ConversationRead: &proto.ConversationRead{
			UserId:        conn.UserID(),
			PeerID:        peerId,
			GroupID:       groupId,
			LastReadMsgID: lastReadMsgId,
		},
	})

	if err := h.publishUpstream(msg); err != nil {
		h.logger.Error("Failed to publish conversation read to NATS", "error", err)
		h.sendClientResponse(stream, reqID, im_protocol.ErrorCodeUNKNOWN_ERROR, "internal error", im_protocol.ResponsePayloadNONE, nil)
		return
	}

	// 返回成功
	h.sendClientResponse(stream, reqID, im_protocol.ErrorCodeSUCCESS, "", im_protocol.ResponsePayloadNONE, nil)
	// Conversation read forwarded
}
