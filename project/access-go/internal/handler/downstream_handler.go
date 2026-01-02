package handler

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
	"sudooom.im.access/internal/connection"
	im_protocol "sudooom.im.access/pkg/flatbuf/im/protocol"
	"sudooom.im.shared/proto"
)

// HandleDownstream 处理下行消息（从 Logic 推送到客户端）
func (h *Handler) HandleDownstream(data []byte) {
	// Downstream message processing

	// 解析下行消息
	var msg proto.DownstreamMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		h.logger.Error("Failed to unmarshal downstream message", "error", err)
		return
	}

	// 优先使用 ConnId 直接路由
	if msg.ConnId > 0 {
		conn := h.connMgr.Get(msg.ConnId)
		if conn == nil {
			h.logger.Warn("Connection not found for downstream message", "connId", msg.ConnId)
			return
		}
		h.sendToConnection(conn, &msg)
		return
	}

	// UserId 是必需的
	if msg.UserId == 0 {
		h.logger.Warn("No userId found in downstream message")
		return
	}

	// 使用 Platform 路由到指定平台
	if msg.Platform != "" {
		conn := h.connMgr.GetByUserIDAndPlatform(msg.UserId, msg.Platform)
		if conn != nil {
			h.sendToConnection(conn, &msg)
		}
		return
	}

	// 没有 Platform，推送到所有平台
	conns := h.connMgr.GetByUserID(msg.UserId)
	for _, conn := range conns {
		h.sendToConnection(conn, &msg)
	}
}

// sendToConnection 发送消息到指定连接
func (h *Handler) sendToConnection(conn *connection.Connection, msg *proto.DownstreamMessage) {
	if msg.Payload.PushMessage != nil {
		h.handlePushMessage(conn, msg.Payload.PushMessage)
	} else if msg.Payload.MessageAck != nil {
		h.handleMessageAck(conn, msg.Payload.MessageAck)
	}
}

func (h *Handler) handlePushMessage(conn *connection.Connection, pushMsg *proto.PushMessage) {
	// 使用 FlatBuffers 构建 ChatPush
	builder := flatbuffers.NewBuilder(512)

	msgIdOffset := builder.CreateString(fmt.Sprintf("%d", pushMsg.ServerMsgId))
	senderIdOffset := builder.CreateString(fmt.Sprintf("%d", pushMsg.FromUserId))
	targetIdOffset := builder.CreateString(fmt.Sprintf("%d", pushMsg.ToUserId))
	contentOffset := builder.CreateString(string(pushMsg.Content)) // content 是 string 类型

	im_protocol.ChatPushStart(builder)
	im_protocol.ChatPushAddMsgId(builder, msgIdOffset)
	im_protocol.ChatPushAddSenderId(builder, senderIdOffset)
	im_protocol.ChatPushAddChatType(builder, im_protocol.ChatTypePRIVATE)
	im_protocol.ChatPushAddTargetId(builder, targetIdOffset)
	im_protocol.ChatPushAddMsgType(builder, im_protocol.MsgType(pushMsg.MsgType))
	im_protocol.ChatPushAddContent(builder, contentOffset)
	im_protocol.ChatPushAddSendTime(builder, pushMsg.Timestamp)
	chatPushOffset := im_protocol.ChatPushEnd(builder)
	builder.Finish(chatPushOffset)

	payload := builder.FinishedBytes()

	// 构建 ClientResponse 并发送
	respFrame := h.buildClientResponseFrame("", im_protocol.ErrorCodeSUCCESS, "", im_protocol.ResponsePayloadChatPush, payload)
	err := conn.Send(respFrame)
	if err != nil {
		h.logger.Error("Failed to send push message to user", "userId", conn.UserID(), "error", err)
	}
}

func (h *Handler) handleMessageAck(conn *connection.Connection, ack *proto.MessageAck) {
	// 使用 FlatBuffers 构建 ChatSendAck
	builder := flatbuffers.NewBuilder(128)

	msgIdOffset := builder.CreateString(fmt.Sprintf("%d", ack.ServerMsgId))

	im_protocol.ChatSendAckStart(builder)
	im_protocol.ChatSendAckAddMsgId(builder, msgIdOffset)
	im_protocol.ChatSendAckAddSendTime(builder, ack.Timestamp)
	ackOffset := im_protocol.ChatSendAckEnd(builder)
	builder.Finish(ackOffset)

	payload := builder.FinishedBytes()

	// 构建 ClientResponse 并发送
	// reqId 使用 ClientMsgId，让客户端可以关联请求
	respFrame := h.buildClientResponseFrame(ack.ClientMsgId, im_protocol.ErrorCodeSUCCESS, "", im_protocol.ResponsePayloadChatSendAck, payload)
	if err := conn.Send(respFrame); err != nil {
		h.logger.Error("Failed to send ACK to user", "userId", conn.UserID(), "error", err)
	}
}

// buildClientResponseFrame 构建完整的 ClientResponse 帧（用于 conn.Send 推送）
func (h *Handler) buildClientResponseFrame(reqID string, code im_protocol.ErrorCode, msg string, payloadType im_protocol.ResponsePayload, payload []byte) []byte {
	builder := flatbuffers.NewBuilder(256 + len(payload))

	reqIDOffset := builder.CreateString(reqID)
	msgOffset := builder.CreateString(msg)

	var payloadOffset flatbuffers.UOffsetT
	if len(payload) > 0 {
		payloadOffset = builder.CreateByteVector(payload)
	}

	im_protocol.ClientResponseStart(builder)
	im_protocol.ClientResponseAddReqId(builder, reqIDOffset)
	im_protocol.ClientResponseAddTimestamp(builder, time.Now().UnixMilli())
	im_protocol.ClientResponseAddCode(builder, code)
	im_protocol.ClientResponseAddMsg(builder, msgOffset)
	im_protocol.ClientResponseAddPayloadType(builder, payloadType)
	if len(payload) > 0 {
		im_protocol.ClientResponseAddPayload(builder, payloadOffset)
	}
	respOffset := im_protocol.ClientResponseEnd(builder)
	builder.Finish(respOffset)

	respBytes := builder.FinishedBytes()

	// 构建帧：header + body
	frame := make([]byte, FrameHeaderSize+len(respBytes))
	binary.BigEndian.PutUint32(frame[:4], uint32(len(respBytes)))
	frame[4] = FrameTypeResponse
	copy(frame[FrameHeaderSize:], respBytes)
	return frame
}
