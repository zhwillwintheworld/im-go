package handler

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
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

	if msg.Payload.PushMessage != nil {
		h.handlePushMessage(msg.Payload.PushMessage)
	}

	if msg.Payload.MessageAck != nil {
		h.handleMessageAck(msg.Payload.MessageAck)
	}
}

func (h *Handler) handlePushMessage(pushMsg *proto.PushMessage) {
	conns := h.connMgr.GetByUserID(pushMsg.ToUserId)
	if len(conns) == 0 {
		// User offline, push dropped
		return
	}

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
	for _, conn := range conns {
		respFrame := h.buildClientResponseFrame("", im_protocol.ErrorCodeSUCCESS, "", im_protocol.ResponsePayloadChatPush, payload)
		err := conn.Send(respFrame)
		if err != nil {
			h.logger.Error("Failed to send push message to user", "userId", conn.UserID(), "error", err)
			continue // 继续发送给其他连接
		}
	}

	// Message pushed
}

func (h *Handler) handleMessageAck(ack *proto.MessageAck) {
	conns := h.connMgr.GetByUserID(ack.ToUserId)
	if len(conns) == 0 {
		// User offline, ACK dropped
		return
	}

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
	for _, conn := range conns {
		respFrame := h.buildClientResponseFrame(ack.ClientMsgId, im_protocol.ErrorCodeSUCCESS, "", im_protocol.ResponsePayloadChatSendAck, payload)
		if err := conn.Send(respFrame); err != nil {
			h.logger.Error("Failed to send ACK to user", "userId", conn.UserID(), "error", err)
			continue // 继续发送给其他连接
		}
	}

	// ACK sent
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
