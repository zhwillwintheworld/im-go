package handler

import (
	"context"
	"encoding/binary"
	"io"
	"time"

	"log/slog"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/quic-go/webtransport-go"
	"sudooom.im.access/internal/connection"
	"sudooom.im.access/internal/nats"
	"sudooom.im.access/internal/redis"
	"sudooom.im.access/internal/workerpool"
	im_protocol "sudooom.im.access/pkg/flatbuf/im/protocol"
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
	workerPool  *workerpool.Pool
}

func NewHandler(connMgr *connection.Manager, natsClient *nats.Client, redisClient *redis.Client, nodeID string, logger *slog.Logger, workerPool *workerpool.Pool) *Handler {
	return &Handler{
		connMgr:     connMgr,
		natsClient:  natsClient,
		redisClient: redisClient,
		nodeID:      nodeID,
		logger:      logger,
		workerPool:  workerPool,
	}
}

func (h *Handler) HandleStream(ctx context.Context, conn *connection.Connection, stream *webtransport.Stream) {
	defer func(stream *webtransport.Stream) {
		err := stream.Close()
		if err != nil {
		}
	}(stream)

	// 设置客户端流，用于发送消息 ACK
	conn.SetClientStream(stream)

	for {
		// 读取帧头：4 bytes length + 1 byte frame type
		header := make([]byte, FrameHeaderSize)
		if _, err := io.ReadFull(stream, header); err != nil {
			if err != io.EOF {
				// Silent error handling
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

		// 根据帧类型分发处理（异步，避免阻塞后续消息）
		// 复制 body，因为原 slice 会在下一次循环中被重用
		bodyCopy := make([]byte, len(body))
		copy(bodyCopy, body)

		// 异步提交到 Worker Pool，避免阻塞消息读取循环
		submitted := h.workerPool.Submit(func() {
			h.dispatch(ctx, conn, stream, frameType, bodyCopy)
		})

		if !submitted {
			h.logger.Warn("Worker pool is shutting down, message dropped")
		}
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

// handleClientRequest 处理 ClientRequest 统一包装的请求
func (h *Handler) handleClientRequest(ctx context.Context, conn *connection.Connection, stream *webtransport.Stream, body []byte) {
	// 解析 ClientRequest
	clientReq := im_protocol.GetRootAsClientRequest(body, 0)
	reqID := string(clientReq.ReqId())
	payloadType := clientReq.PayloadType()
	payload := clientReq.PayloadBytes()
	// Request processing
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
	case im_protocol.RequestPayloadConversationReadReq:
		h.handleConversationRead(conn, stream, reqID, payload)
	default:
		h.logger.Warn("Unknown payload type", "payload_type", payloadType)
		h.sendClientResponse(stream, reqID, im_protocol.ErrorCodeUNKNOWN_ERROR, "unknown payload type", im_protocol.ResponsePayloadNONE, nil)
	}
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
