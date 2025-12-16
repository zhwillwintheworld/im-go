package protocol

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"log/slog"

	"github.com/quic-go/webtransport-go"
	"sudooom.im.access/internal/connection"
	"sudooom.im.access/internal/nats"
	"sudooom.im.access/internal/redis"
	im_protocol "sudooom.im.access/pkg/flatbuf/im/protocol"
	sharedNats "sudooom.im.shared/nats"
	"sudooom.im.shared/proto"
)

const (
	HeaderSize = 6 // 4 bytes length + 2 bytes msg type

	// 消息类型
	MsgTypeHeartbeat  uint16 = 0
	MsgTypeAuth       uint16 = 1
	MsgTypeAuthAck    uint16 = 2
	MsgTypeMessage    uint16 = 10
	MsgTypeMessageAck uint16 = 11
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
	defer stream.Close()

	for {
		// 读取消息头
		header := make([]byte, HeaderSize)
		if _, err := io.ReadFull(stream, header); err != nil {
			if err != io.EOF {
				h.logger.Debug("Failed to read header", "error", err)
			}
			return
		}

		length := binary.BigEndian.Uint32(header[:4])
		msgType := binary.BigEndian.Uint16(header[4:6])

		// 读取消息体
		body := make([]byte, length)
		if _, err := io.ReadFull(stream, body); err != nil {
			h.logger.Error("Failed to read body", "error", err)
			return
		}

		// 更新活跃时间
		conn.UpdateActive()

		// 处理消息
		h.dispatch(ctx, conn, stream, msgType, body)
	}
}

func (h *Handler) dispatch(ctx context.Context, conn *connection.Connection, stream *webtransport.Stream, msgType uint16, body []byte) {
	switch msgType {
	case MsgTypeHeartbeat:
		h.handleHeartbeat(ctx, conn, stream)
	case MsgTypeAuth:
		h.handleAuth(ctx, conn, stream, body)
	default:
		// 转发到 Logic 层
		h.forwardToLogic(conn, msgType, body)
	}
}

func (h *Handler) handleHeartbeat(ctx context.Context, conn *connection.Connection, stream *webtransport.Stream) {
	h.logger.Debug("Heartbeat received", "conn_id", conn.ID())

	// 刷新用户位置 TTL
	if conn.UserID() > 0 {
		h.redisClient.RefreshUserLocation(ctx, conn.UserID())
	}

	// 回复心跳响应
	h.sendResponse(stream, MsgTypeHeartbeat, nil)
}

func (h *Handler) handleAuth(ctx context.Context, conn *connection.Connection, stream *webtransport.Stream, body []byte) {
	h.logger.Debug("Auth request received", "conn_id", conn.ID())

	// 解析 FlatBuffers 消息
	authReq := im_protocol.GetRootAsAuthRequest(body, 0)

	token := string(authReq.Token())
	deviceID := string(authReq.DeviceId())
	platform := authReq.Platform()
	// appVersion := string(authReq.AppVersion())

	// TODO: 验证 token
	// 这里简化处理，直接绑定用户
	sessInfo := &connection.SessionInfo{
		UserID:   1, // TODO: 从 token 解析，此处可以使用 token
		DeviceID: deviceID,
		Platform: platform.String(),
	}
	// 临时使用 token 避免未使用错误
	_ = token
	conn.BindSession(sessInfo)
	h.connMgr.BindUser(conn.ID(), sessInfo.UserID)

	// 注册用户位置到 Redis
	if err := h.redisClient.RegisterUserLocation(ctx, sessInfo.UserID, conn.ID(), sessInfo.DeviceID, sessInfo.Platform); err != nil {
		h.logger.Error("Failed to register user location", "error", err)
	}

	// 发送上线通知到 Logic
	h.sendUserOnlineToLogic(conn, sessInfo)

	// 回复认证成功
	response := map[string]interface{}{
		"code":    0,
		"user_id": sessInfo.UserID,
		"message": "success",
	}
	respData, _ := json.Marshal(response)
	h.sendResponse(stream, MsgTypeAuthAck, respData)
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

// ClientMessage 客户端发送的消息结构
type ClientMessage struct {
	ClientMsgId string `json:"clientMsgId"`
	ToUserId    int64  `json:"toUserId"`
	ToGroupId   int64  `json:"toGroupId"`
	Content     string `json:"content"`
}

func (h *Handler) forwardToLogic(conn *connection.Connection, msgType uint16, body []byte) {
	// 解析客户端消息
	var clientMsg ClientMessage
	if err := json.Unmarshal(body, &clientMsg); err != nil {
		h.logger.Error("Failed to unmarshal client message", "error", err)
		return
	}

	// 封装上行消息
	msg := &proto.UpstreamMessage{
		AccessNodeId: h.nodeID,
		UserMessage: &proto.UserMessage{
			FromUserId:  conn.UserID(),
			ClientMsgId: clientMsg.ClientMsgId,
			ToUserId:    clientMsg.ToUserId,
			ToGroupId:   clientMsg.ToGroupId,
			MsgType:     int32(msgType),
			Content:     []byte(clientMsg.Content),
			Timestamp:   0, // Logic 层会处理
		},
	}
	data, _ := json.Marshal(msg)
	h.natsClient.Publish(sharedNats.SubjectLogicUpstream, data)
}

func (h *Handler) HandleDownstream(data []byte) {
	h.logger.Debug("Downstream message received", "size", len(data))

	// 解析下行消息
	var msg proto.DownstreamMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		h.logger.Error("Failed to unmarshal downstream message", "error", err)
		return
	}

	if msg.Payload.PushMessage != nil {
		// 查找用户连接并发送
		conns := h.connMgr.GetByUserID(msg.Payload.PushMessage.ToUserId)
		for _, conn := range conns {
			// 构建消息帧: 直接发送内容？还是需要包装？
			// 这里假设客户端希望收到原始 PushMessage 结构或仅 Content
			// 为了简单，我们发送整个 PushMessage 的 JSON
			// 但协议定义的是 binary frame header + body.
			// MsgTypeMessage = 10.
			// 让我们将 PushMessage 序列化后作为 Body 发送
			respData, _ := json.Marshal(msg.Payload.PushMessage)
			frame := h.buildMessageFrame(MsgTypeMessage, respData)
			conn.Send(frame)
		}
		h.logger.Debug("Pushed message to users",
			"toUserId", msg.Payload.PushMessage.ToUserId,
			"connCount", len(conns))
	}

	if msg.Payload.MessageAck != nil {
		// 处理消息 ACK
		ack := msg.Payload.MessageAck
		conns := h.connMgr.GetByUserID(ack.ToUserId)
		for _, conn := range conns {
			// 构建 ACK 帧
			// 使用 MsgTypeMessageAck (11)
			respData, _ := json.Marshal(ack)
			frame := h.buildMessageFrame(MsgTypeMessageAck, respData)
			conn.Send(frame)
		}

		if len(conns) > 0 {
			h.logger.Debug("Message ACK sent to user",
				"userId", ack.ToUserId,
				"clientMsgId", ack.ClientMsgId)
		} else {
			h.logger.Debug("Message ACK dropped, user offline",
				"userId", ack.ToUserId)
		}
	}
}

func (h *Handler) buildMessageFrame(msgType uint16, body []byte) []byte {
	frame := make([]byte, HeaderSize+len(body))
	binary.BigEndian.PutUint32(frame[:4], uint32(len(body)))
	binary.BigEndian.PutUint16(frame[4:6], msgType)
	copy(frame[HeaderSize:], body)
	return frame
}

func (h *Handler) sendResponse(stream *webtransport.Stream, msgType uint16, body []byte) {
	header := make([]byte, HeaderSize)
	binary.BigEndian.PutUint32(header[:4], uint32(len(body)))
	binary.BigEndian.PutUint16(header[4:6], msgType)
	stream.Write(header)
	if len(body) > 0 {
		stream.Write(body)
	}
}
