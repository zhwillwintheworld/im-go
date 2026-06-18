package handler

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sync"

	"log/slog"

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
	// FrameHeaderSize 帧头大小：4 bytes length + 1 byte frame type
	FrameHeaderSize = 5

	// FrameTypeAuth 帧类型
	FrameTypeAuth    byte = 1 // 认证请求（AuthRequest）
	FrameTypeRequest byte = 2 // 普通请求（ClientRequest）

	// FrameTypeAuthAck 响应帧类型
	FrameTypeAuthAck  byte = 3 // 认证响应
	FrameTypeResponse byte = 4 // 普通响应（ClientResponse）

	// Buffer Pool 默认容量（4KB，适合大多数消息）
	defaultBufferCap = 4096

	// defaultMaxFrameSize 默认最大帧体长度，避免绕过配置加载时失去内存保护。
	defaultMaxFrameSize = 1 << 20
)

var ErrFrameTooLarge = errors.New("frame body too large")

type Handler struct {
	connMgr        *connection.Manager
	natsClient     *nats.Client
	redisClient    *redis.Client
	nodeID         string
	logger         *slog.Logger
	workerPool     *workerpool.Pool
	bufferPool     *sync.Pool // 消息 buffer 对象池，减少内存分配
	maxFrameSize   uint32
	roomShardCount int
}

func NewHandler(connMgr *connection.Manager, natsClient *nats.Client, redisClient *redis.Client, nodeID string, logger *slog.Logger, workerPool *workerpool.Pool, maxFrameSize int, roomShardCount int) *Handler {
	if roomShardCount <= 0 {
		roomShardCount = sharedNats.DefaultRoomShardCount
	}
	return &Handler{
		connMgr:        connMgr,
		natsClient:     natsClient,
		redisClient:    redisClient,
		nodeID:         nodeID,
		logger:         logger,
		workerPool:     workerPool,
		maxFrameSize:   normalizeMaxFrameSize(maxFrameSize),
		roomShardCount: roomShardCount,
		bufferPool: &sync.Pool{
			New: func() interface{} {
				buf := make([]byte, 0, defaultBufferCap)
				return &buf
			},
		},
	}
}

// HandleStream 处理客户端流（连接已认证）
func (h *Handler) HandleStream(ctx context.Context, conn *connection.Connection, stream *webtransport.Stream) {
	defer func(stream *webtransport.Stream) {
		if err := stream.Close(); err != nil {
			h.logger.Debug("Failed to close client stream", "error", err, "conn_id", conn.ID())
		}
	}(stream)

	conn.SetClientStream(stream)

	for {
		frameType, body, err := readFrame(stream, h.maxFrameSize)
		if err != nil {
			if errors.Is(err, ErrFrameTooLarge) {
				h.logger.Warn("Reject frame because body is too large",
					"error", err,
					"conn_id", conn.ID(),
					"frameType", frameType)
			} else if err != io.EOF {
				h.logger.Debug("Failed to read frame", "error", err, "conn_id", conn.ID())
			}
			return
		}

		conn.UpdateActive()

		// 从对象池获取buffer
		bufPtr := h.bufferPool.Get().(*[]byte)
		buf := *bufPtr

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
			defer func() {
				buf = buf[:0]
				*bufPtr = buf
				h.bufferPool.Put(bufPtr)
			}()
			h.dispatch(ctx, conn, stream, frameType, buf)
		})

		if !submitted {
			h.logger.Warn("Worker pool busy or shutting down, message dropped",
				"conn_id", conn.ID(),
				"frameType", frameType,
				"bodyLength", len(body))
			buf = buf[:0]
			*bufPtr = buf
			h.bufferPool.Put(bufPtr)
		}
	}
}

func normalizeMaxFrameSize(maxFrameSize int) uint32 {
	if maxFrameSize <= 0 {
		return defaultMaxFrameSize
	}
	if maxFrameSize > math.MaxUint32 {
		return math.MaxUint32
	}
	return uint32(maxFrameSize)
}

func readFrame(reader io.Reader, maxBodySize uint32) (byte, []byte, error) {
	header := make([]byte, FrameHeaderSize)
	if _, err := io.ReadFull(reader, header); err != nil {
		return 0, nil, err
	}

	length := binary.BigEndian.Uint32(header[:4])
	frameType := header[4]
	if maxBodySize > 0 && length > maxBodySize {
		return frameType, nil, fmt.Errorf("%w: length=%d max=%d", ErrFrameTooLarge, length, maxBodySize)
	}

	body := make([]byte, length)
	if _, err := io.ReadFull(reader, body); err != nil {
		return frameType, nil, err
	}

	return frameType, body, nil
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
		h.sendClientResponse(conn, reqID, im_protocol.ErrorCodeUNKNOWN_ERROR, "unknown request type", im_protocol.ResponsePayloadNONE, nil)
	}
}

// sendClientResponse 构建响应帧后通过连接写队列发送，保护 stream 单 writer 语义。
func (h *Handler) sendClientResponse(conn *connection.Connection, reqID string, code im_protocol.ErrorCode, msg string, payloadType im_protocol.ResponsePayload, payload []byte) {
	frame := h.buildClientResponseFrame(reqID, code, msg, payloadType, payload)
	if err := conn.Send(frame); err != nil {
		if errors.Is(err, connection.ErrConnectionBackpressure) {
			h.logger.Warn("Drop client response because connection write queue is full",
				"conn_id", conn.ID(),
				"user_id", conn.UserID(),
				"req_id", reqID,
				"payload_type", payloadType,
				"error", err)
			return
		}
		h.logger.Error("Failed to enqueue client response",
			"conn_id", conn.ID(),
			"user_id", conn.UserID(),
			"req_id", reqID,
			"payload_type", payloadType,
			"error", err)
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
	return h.publishUpstreamToSubject(sharedNats.SubjectLogicUpstream, msg)
}

func (h *Handler) publishUpstreamToSubject(subject string, msg *proto.UpstreamMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return h.natsClient.Publish(subject, data)
}

func (h *Handler) logicRoomShardSubject(roomID string, fallbackUserID int64) string {
	shard := sharedNats.ShardForRoomID(roomID, fallbackUserID, h.roomShardCount)
	return sharedNats.BuildLogicRoomShardSubject(shard)
}

func (h *Handler) logicGameShardSubject(roomID string, fallbackUserID int64) string {
	shard := sharedNats.ShardForRoomID(roomID, fallbackUserID, h.roomShardCount)
	return sharedNats.BuildLogicGameShardSubject(shard)
}
