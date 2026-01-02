package handler

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"sync"
	"time"

	"log/slog"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/quic-go/webtransport-go"
	"sudooom.im.access/internal/connection"
	"sudooom.im.access/internal/nats"
	"sudooom.im.access/internal/redis"
	"sudooom.im.access/internal/workerpool"
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

	// Buffer Pool 默认容量（4KB，适合大多数消息）
	defaultBufferCap = 4096
)

type Handler struct {
	connMgr     *connection.Manager
	natsClient  *nats.Client
	redisClient *redis.Client
	nodeID      string
	logger      *slog.Logger
	workerPool  *workerpool.Pool
	bufferPool  *sync.Pool // 消息 buffer 对象池，减少内存分配
}

func NewHandler(connMgr *connection.Manager, natsClient *nats.Client, redisClient *redis.Client, nodeID string, logger *slog.Logger, workerPool *workerpool.Pool) *Handler {
	return &Handler{
		connMgr:     connMgr,
		natsClient:  natsClient,
		redisClient: redisClient,
		nodeID:      nodeID,
		logger:      logger,
		workerPool:  workerPool,
		bufferPool: &sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, defaultBufferCap)
			},
		},
	}
}

// HandleStream 处理客户端流（连接已认证）
func (h *Handler) HandleStream(ctx context.Context, conn *connection.Connection, stream *webtransport.Stream) {
	defer func(stream *webtransport.Stream) {
		err := stream.Close()
		if err != nil {
		}
	}(stream)

	conn.SetClientStream(stream)

	for {
		header := make([]byte, FrameHeaderSize)
		if _, err := io.ReadFull(stream, header); err != nil {
			if err != io.EOF {
			}
			return
		}

		length := binary.BigEndian.Uint32(header[:4])
		frameType := header[4]

		body := make([]byte, length)
		if _, err := io.ReadFull(stream, body); err != nil {
			h.logger.Error("Failed to read body", "error", err)
			return
		}

		conn.UpdateActive()

		// 从对象池获取buffer
		buf := h.bufferPool.Get().([]byte)

		// 检查容量
		if cap(buf) < len(body) {
			buf = make([]byte, len(body))
		} else {
			buf = buf[:len(body)]
		}

		// 复制数据
		copy(buf, body)

		// 异步提交到 Worker Pool，避免阻塞消息读取循环
		submitted := h.workerPool.Submit(func() {
			defer h.bufferPool.Put(buf[:0]) // 处理完后归还到对象池
			h.dispatch(ctx, conn, stream, frameType, buf)
		})

		if !submitted {
			h.logger.Warn("Worker pool is shutting down, message dropped")
			h.bufferPool.Put(buf[:0]) // 如果提交失败，手动归还buffer
		}
	}
}

// dispatch 根据帧类型分发处理
func (h *Handler) dispatch(ctx context.Context, conn *connection.Connection, stream *webtransport.Stream, frameType byte, body []byte) {
	switch frameType {
	case FrameTypeAuth:
		h.logger.Warn("Unexpected auth request after authentication", "conn_id", conn.ID())
	case FrameTypeRequest:
		h.handleClientRequest(ctx, conn, stream, body)
	default:
		h.logger.Warn("Unknown frame type", "frameType", frameType)
	}
}

// handleClientRequest 处理客户端请求
func (h *Handler) handleClientRequest(ctx context.Context, conn *connection.Connection, stream *webtransport.Stream, body []byte) {
	clientReq := im_protocol.GetRootAsClientRequest(body, 0)

	reqID := string(clientReq.ReqId())
	payloadType := clientReq.PayloadType()

	payload := clientReq.PayloadBytes()

	// 根据 Payload 类型分发
	switch payloadType {
	case im_protocol.RequestPayloadChatSendReq:
		h.handleChatSend(ctx, conn, stream, reqID, payload)
	case im_protocol.RequestPayloadHeartbeatReq:
		h.handleHeartbeat(ctx, conn, stream, reqID, payload)
	case im_protocol.RequestPayloadConversationReadReq:
		h.handleConversationRead(conn, stream, reqID, payload)
	case im_protocol.RequestPayloadRoomReq:
		h.handleRoomRequest(ctx, conn, reqID, payload)
	case im_protocol.RequestPayloadGameReq:
		h.handleGameRequest(ctx, conn, reqID, payload)
	default:
		h.logger.Warn("Unknown payload type", "payloadType", payloadType)
		h.sendClientResponse(stream, reqID, im_protocol.ErrorCodeUNKNOWN_ERROR, "unknown request type", im_protocol.ResponsePayloadNONE, nil)
	}
}

// sendClientResponse 发送响应给客户端
func (h *Handler) sendClientResponse(stream *webtransport.Stream, reqID string, code im_protocol.ErrorCode, msg string, payloadType im_protocol.ResponsePayload, payload []byte) {
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

	// 发送带帧头的响应
	h.sendFrame(stream, FrameTypeResponse, respBytes)
}

// sendFrame 发送带帧头的数据
func (h *Handler) sendFrame(stream *webtransport.Stream, frameType byte, body []byte) {
	header := make([]byte, FrameHeaderSize)
	binary.BigEndian.PutUint32(header[:4], uint32(len(body)))
	header[4] = frameType
	_, err := stream.Write(header)
	if err != nil {
		return
	}
	if len(body) > 0 {
		_, err := stream.Write(body)
		if err != nil {
			return
		}
	}
}

// buildUpstreamMessage 构建上行消息（辅助方法，减少重复代码）
func (h *Handler) buildUpstreamMessage(conn *connection.Connection, payload proto.UpstreamPayload) *proto.UpstreamMessage {
	return &proto.UpstreamMessage{
		AccessNodeId: h.nodeID,
		ConnId:       conn.ID(),
		Platform:     conn.Platform(),
		Payload:      payload,
	}
}

// publishUpstream 发布上行消息到 Logic（辅助方法）
func (h *Handler) publishUpstream(msg *proto.UpstreamMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return h.natsClient.Publish(sharedNats.SubjectLogicUpstream, data)
}
