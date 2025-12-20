package protocol

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/quic-go/webtransport-go"
	"sudooom.im.access/internal/connection"
	"sudooom.im.access/internal/nats"
	"sudooom.im.access/internal/redis"
	im_protocol "sudooom.im.access/pkg/flatbuf/im/protocol"
	sharedNats "sudooom.im.shared/nats"
	"sudooom.im.shared/proto"
)

const (
	// 帧头大小：4 bytes length + 1 byte frame type
	FrameHeaderSize = 5

	// 帧类型
	FrameTypeAuth    byte = 1 // 认证请求（AuthRequest）
	FrameTypeRequest byte = 2 // 普通请求（ClientRequest）

	// 响应帧类型
	FrameTypeAuthAck  byte = 3 // 认证响应
	FrameTypeResponse byte = 4 // 普通响应（ClientResponse）
)

type Handler struct {
	connMgr     *connection.Manager
	natsClient  *nats.Client
	redisClient *redis.Client
	nodeID      string
	logger      *slog.Logger
}

func NewHandler(connMgr *connection.Manager, natsClient *nats.Client, redisClient *redis.Client, nodeID string, logger *slog.Logger) *Handler {
	return &Handler{
		connMgr:     connMgr,
		natsClient:  natsClient,
		redisClient: redisClient,
		nodeID:      nodeID,
		logger:      logger,
	}
}

func (h *Handler) HandleStream(ctx context.Context, conn *connection.Connection, stream *webtransport.Stream) {
	defer func(stream *webtransport.Stream) {
		err := stream.Close()
		if err != nil {
		}
	}(stream)

	for {
		// 读取帧头：4 bytes length + 1 byte frame type
		header := make([]byte, FrameHeaderSize)
		if _, err := io.ReadFull(stream, header); err != nil {
			if err != io.EOF {
				h.logger.Debug("Failed to read header", "error", err)
			}
			return
		}

		length := binary.BigEndian.Uint32(header[:4])
		frameType := header[4]

		// 读取消息体
		body := make([]byte, length)
		if _, err := io.ReadFull(stream, body); err != nil {
			h.logger.Error("Failed to read body", "error", err)
			return
		}

		// 更新活跃时间
		conn.UpdateActive()

		// 根据帧类型分发处理
		h.dispatch(ctx, conn, stream, frameType, body)
	}
}

func (h *Handler) dispatch(ctx context.Context, conn *connection.Connection, stream *webtransport.Stream, frameType byte, body []byte) {
	switch frameType {
	case FrameTypeAuth:
		// 认证请求在正常流程中不应该出现（已通过 HandleFirstStream 处理）
		h.logger.Warn("Unexpected auth request after authentication", "conn_id", conn.ID())
	case FrameTypeRequest:
		h.handleClientRequest(ctx, conn, stream, body)
	default:
		h.logger.Warn("Unknown frame type", "frameType", frameType)
	}
}

// HandleFirstStream 处理首个数据流，必须是认证请求
// 返回 error 表示认证失败，调用方应关闭连接
func (h *Handler) HandleFirstStream(ctx context.Context, conn *connection.Connection, stream *webtransport.Stream) error {
	defer stream.Close()

	// 读取帧头
	header := make([]byte, FrameHeaderSize)
	if _, err := io.ReadFull(stream, header); err != nil {
		h.logger.Debug("Failed to read header", "error", err)
		return fmt.Errorf("failed to read header: %w", err)
	}

	length := binary.BigEndian.Uint32(header[:4])
	frameType := header[4]

	// 检查首包必须是 Auth 请求
	if frameType != FrameTypeAuth {
		h.logger.Warn("First frame must be auth request", "conn_id", conn.ID(), "frameType", frameType)
		h.sendClientResponse(stream, "", im_protocol.ErrorCodeAUTH_FAILED, "auth required", im_protocol.ResponsePayloadNONE, nil)
		return fmt.Errorf("first frame is not auth request")
	}

	// 读取消息体
	body := make([]byte, length)
	if _, err := io.ReadFull(stream, body); err != nil {
		h.logger.Error("Failed to read body", "error", err)
		return fmt.Errorf("failed to read body: %w", err)
	}

	// 更新活跃时间
	conn.UpdateActive()

	// 处理认证
	return h.handleAuth(ctx, conn, stream, body)
}

// handleAuth 处理认证请求，返回 error 表示认证失败
func (h *Handler) handleAuth(ctx context.Context, conn *connection.Connection, stream *webtransport.Stream, body []byte) error {
	h.logger.Debug("Auth request received", "conn_id", conn.ID())

	// 解析 FlatBuffers AuthRequest
	authReq := im_protocol.GetRootAsAuthRequest(body, 0)

	token := string(authReq.Token())
	deviceID := string(authReq.DeviceId())
	platform := authReq.Platform()

	// 从 Redis 获取 token 对应的用户信息
	userInfo, err := h.redisClient.GetUserInfoByToken(ctx, token)
	if err != nil {
		h.logger.Error("Failed to get user info from Redis", "error", err)
		h.sendClientResponse(stream, "", im_protocol.ErrorCodeUNKNOWN_ERROR, "internal error", im_protocol.ResponsePayloadNONE, nil)
		return fmt.Errorf("redis error: %w", err)
	}

	// 验证 token 是否存在
	if userInfo == nil {
		h.logger.Warn("Token not found", "conn_id", conn.ID())
		h.sendClientResponse(stream, "", im_protocol.ErrorCodeAUTH_FAILED, "invalid token", im_protocol.ResponsePayloadNONE, nil)
		return fmt.Errorf("token not found")
	}

	// 比对 deviceId
	if userInfo.DeviceID != deviceID {
		h.logger.Warn("DeviceID mismatch", "conn_id", conn.ID(), "expected", userInfo.DeviceID, "got", deviceID)
		h.sendClientResponse(stream, "", im_protocol.ErrorCodeAUTH_FAILED, "device mismatch", im_protocol.ResponsePayloadNONE, nil)
		return fmt.Errorf("device mismatch")
	}

	// 比对 platform（不区分大小写，FlatBuffers 返回大写，Redis 存储小写）
	if !strings.EqualFold(userInfo.Platform, platform.String()) {
		h.logger.Warn("Platform mismatch", "conn_id", conn.ID(), "expected", userInfo.Platform, "got", platform.String())
		h.sendClientResponse(stream, "", im_protocol.ErrorCodeAUTH_FAILED, "platform mismatch", im_protocol.ResponsePayloadNONE, nil)
		return fmt.Errorf("platform mismatch")
	}

	// 构建 sessInfo 使用 Redis 中的用户信息
	sessInfo := &connection.SessionInfo{
		UserID:   userInfo.UserID,
		DeviceID: deviceID,
		Platform: platform.String(),
	}
	conn.BindSession(sessInfo)
	h.connMgr.BindUser(conn.ID(), sessInfo.UserID)

	// 注册用户位置到 Redis
	if err := h.redisClient.RegisterUserLocation(ctx, sessInfo.UserID, sessInfo.Platform); err != nil {
		h.logger.Error("Failed to register user location", "error", err)
	}

	// 发送上线通知到 Logic
	h.sendUserOnlineToLogic(conn, sessInfo)

	// 发送认证成功响应
	h.sendClientResponse(stream, "", im_protocol.ErrorCodeSUCCESS, "success", im_protocol.ResponsePayloadNONE, nil)
	h.logger.Info("User authenticated", "conn_id", conn.ID(), "user_id", userInfo.UserID)

	return nil
}

// handleClientRequest 处理 ClientRequest 统一包装的请求
func (h *Handler) handleClientRequest(ctx context.Context, conn *connection.Connection, stream *webtransport.Stream, body []byte) {
	// 解析 ClientRequest
	clientReq := im_protocol.GetRootAsClientRequest(body, 0)
	reqID := string(clientReq.ReqId())
	payloadType := clientReq.PayloadType()
	payload := clientReq.PayloadBytes()

	h.logger.Debug("ClientRequest received",
		"conn_id", conn.ID(),
		"req_id", reqID,
		"payload_type", payloadType.String())

	// 根据 PayloadType 分发
	switch payloadType {
	case im_protocol.RequestPayloadHeartbeatReq:
		h.handleHeartbeat(ctx, conn, stream, reqID, payload)
	case im_protocol.RequestPayloadChatSendReq:
		h.handleChatSend(ctx, conn, stream, reqID, payload)
	case im_protocol.RequestPayloadRoomReq:
		h.handleRoomRequest(ctx, conn, stream, reqID, payload)
	case im_protocol.RequestPayloadGameReq:
		h.handleGameRequest(ctx, conn, stream, reqID, payload)
	default:
		h.logger.Warn("Unknown payload type", "payload_type", payloadType)
		h.sendClientResponse(stream, reqID, im_protocol.ErrorCodeUNKNOWN_ERROR, "unknown payload type", im_protocol.ResponsePayloadNONE, nil)
	}
}

// handleHeartbeat 处理心跳请求
func (h *Handler) handleHeartbeat(ctx context.Context, conn *connection.Connection, stream *webtransport.Stream, reqID string, payload []byte) {
	h.logger.Debug("Heartbeat received", "conn_id", conn.ID())

	// 刷新用户位置 TTL
	if conn.UserID() > 0 {
		h.redisClient.RefreshUserLocation(ctx, conn.UserID(), conn.Platform())
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

// handleChatSend 处理聊天发送请求
func (h *Handler) handleChatSend(ctx context.Context, conn *connection.Connection, stream *webtransport.Stream, reqID string, payload []byte) {
	h.logger.Debug("ChatSendReq received", "conn_id", conn.ID())

	// 解析 ChatSendReq
	chatReq := im_protocol.GetRootAsChatSendReq(payload, 0)

	// 封装上行消息到 Logic
	msg := &proto.UpstreamMessage{
		AccessNodeId: h.nodeID,
		UserMessage: &proto.UserMessage{
			FromUserId:  conn.UserID(),
			ClientMsgId: reqID,
			ToUserId:    0, // TODO: 从 chatReq.TargetId() 解析
			MsgType:     int32(chatReq.MsgType()),
			Content:     chatReq.Content(),
			Timestamp:   0, // Logic 层会处理
		},
	}

	// 根据 ChatType 设置目标
	switch chatReq.ChatType() {
	case im_protocol.ChatTypePRIVATE:
		// 私聊：targetId 是用户 ID
		// TODO: 解析 targetId 为 int64
	case im_protocol.ChatTypeGROUP:
		// 群聊：targetId 是群组 ID
		// TODO: 解析 targetId 为群组 ID
	}

	data, _ := json.Marshal(msg)
	h.natsClient.Publish(sharedNats.SubjectLogicUpstream, data)

	// 发送 ACK（实际 ACK 由 Logic 返回后再发送）
	h.logger.Debug("ChatSendReq forwarded to logic", "req_id", reqID)
}

// handleRoomRequest 处理房间请求
func (h *Handler) handleRoomRequest(ctx context.Context, conn *connection.Connection, stream *webtransport.Stream, reqID string, payload []byte) {
	h.logger.Debug("RoomReq received", "conn_id", conn.ID())

	// 解析 RoomReq
	roomReq := im_protocol.GetRootAsRoomReq(payload, 0)
	action := roomReq.Action()
	roomID := string(roomReq.RoomId())
	gameType := roomReq.GameType()

	h.logger.Debug("RoomReq details",
		"action", action.String(),
		"room_id", roomID,
		"game_type", gameType.String())

	// TODO: 转发到 Logic 处理
	// 暂时返回成功
	h.sendClientResponse(stream, reqID, im_protocol.ErrorCodeSUCCESS, "", im_protocol.ResponsePayloadRoomResp, nil)
}

// handleGameRequest 处理游戏请求
func (h *Handler) handleGameRequest(ctx context.Context, conn *connection.Connection, stream *webtransport.Stream, reqID string, payload []byte) {
	h.logger.Debug("GameReq received", "conn_id", conn.ID())

	// 解析 GameReq
	gameReq := im_protocol.GetRootAsGameReq(payload, 0)
	roomID := string(gameReq.RoomId())
	gameType := gameReq.GameType()

	h.logger.Debug("GameReq details",
		"room_id", roomID,
		"game_type", gameType.String())

	// TODO: 转发到 Logic 处理
}

func (h *Handler) sendUserOnlineToLogic(conn *connection.Connection, sessInfo *connection.SessionInfo) {
	msg := &proto.UpstreamMessage{
		AccessNodeId: h.nodeID,
		UserOnline: &proto.UserOnline{
			UserId:   sessInfo.UserID,
			ConnId:   conn.ID(),
			DeviceId: sessInfo.DeviceID,
			Platform: sessInfo.Platform,
		},
	}
	data, _ := json.Marshal(msg)
	h.natsClient.Publish(sharedNats.SubjectLogicUpstream, data)
	h.logger.Debug("Sent user online to logic", "userId", sessInfo.UserID)
}

// SendUserOfflineToLogic 发送用户下线通知
func (h *Handler) SendUserOfflineToLogic(conn *connection.Connection) {
	if conn.UserID() == 0 {
		return
	}

	msg := &proto.UpstreamMessage{
		AccessNodeId: h.nodeID,
		UserOffline: &proto.UserOffline{
			UserId: conn.UserID(),
			ConnId: conn.ID(),
		},
	}
	data, _ := json.Marshal(msg)
	h.natsClient.Publish(sharedNats.SubjectLogicUpstream, data)
	h.logger.Debug("Sent user offline to logic", "userId", conn.UserID())
}

// HandleDownstream 处理下行消息（从 Logic 推送到客户端）
func (h *Handler) HandleDownstream(data []byte) {
	h.logger.Debug("Downstream message received", "size", len(data))

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
		h.logger.Debug("Push dropped, user offline", "toUserId", pushMsg.ToUserId)
		return
	}

	// 使用 FlatBuffers 构建 ChatPush
	builder := flatbuffers.NewBuilder(512)

	msgIdOffset := builder.CreateString(fmt.Sprintf("%d", pushMsg.ServerMsgId))
	senderIdOffset := builder.CreateString(fmt.Sprintf("%d", pushMsg.FromUserId))
	targetIdOffset := builder.CreateString(fmt.Sprintf("%d", pushMsg.ToUserId))
	contentOffset := builder.CreateByteVector(pushMsg.Content)

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
		conn.Send(respFrame)
	}

	h.logger.Debug("Pushed message to users",
		"toUserId", pushMsg.ToUserId,
		"connCount", len(conns))
}

func (h *Handler) handleMessageAck(ack *proto.MessageAck) {
	conns := h.connMgr.GetByUserID(ack.ToUserId)
	if len(conns) == 0 {
		h.logger.Debug("Message ACK dropped, user offline", "userId", ack.ToUserId)
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
		conn.Send(respFrame)
	}

	h.logger.Debug("Message ACK sent to user",
		"userId", ack.ToUserId,
		"clientMsgId", ack.ClientMsgId)
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

// sendClientResponse 发送 ClientResponse 响应
func (h *Handler) sendClientResponse(stream *webtransport.Stream, reqID string, code im_protocol.ErrorCode, msg string, payloadType im_protocol.ResponsePayload, payload []byte) {
	builder := flatbuffers.NewBuilder(256)

	// 创建字符串偏移
	reqIDOffset := builder.CreateString(reqID)
	msgOffset := builder.CreateString(msg)

	// 创建 payload 向量（如果有）
	var payloadOffset flatbuffers.UOffsetT
	if len(payload) > 0 {
		payloadOffset = builder.CreateByteVector(payload)
	}

	// 构建 ClientResponse
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

	// 发送带帧头的响应
	h.sendFrame(stream, FrameTypeResponse, respBytes)
}

// sendFrame 发送带帧头的数据
func (h *Handler) sendFrame(stream *webtransport.Stream, frameType byte, body []byte) {
	header := make([]byte, FrameHeaderSize)
	binary.BigEndian.PutUint32(header[:4], uint32(len(body)))
	header[4] = frameType
	stream.Write(header)
	if len(body) > 0 {
		stream.Write(body)
	}
}
