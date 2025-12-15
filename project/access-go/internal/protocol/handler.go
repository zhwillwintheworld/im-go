package protocol

import (
	"context"
	"encoding/binary"
	"io"

	"github.com/example/im-access/internal/connection"
	"github.com/example/im-access/internal/nats"
	"github.com/quic-go/quic-go"
	"go.uber.org/zap"
)

const (
	HeaderSize = 6 // 4 bytes length + 2 bytes msg type
)

type Handler struct {
	connMgr    *connection.Manager
	natsClient *nats.Client
	logger     *zap.Logger
}

func NewHandler(connMgr *connection.Manager, natsClient *nats.Client, logger *zap.Logger) *Handler {
	return &Handler{
		connMgr:    connMgr,
		natsClient: natsClient,
		logger:     logger,
	}
}

func (h *Handler) HandleStream(ctx context.Context, conn *connection.Connection, stream quic.Stream) {
	defer stream.Close()

	for {
		// 读取消息头
		header := make([]byte, HeaderSize)
		if _, err := io.ReadFull(stream, header); err != nil {
			if err != io.EOF {
				h.logger.Debug("Failed to read header", zap.Error(err))
			}
			return
		}

		length := binary.BigEndian.Uint32(header[:4])
		msgType := binary.BigEndian.Uint16(header[4:6])

		// 读取消息体
		body := make([]byte, length)
		if _, err := io.ReadFull(stream, body); err != nil {
			h.logger.Error("Failed to read body", zap.Error(err))
			return
		}

		// 更新活跃时间
		conn.UpdateActive()

		// 处理消息
		h.dispatch(ctx, conn, msgType, body)
	}
}

func (h *Handler) dispatch(ctx context.Context, conn *connection.Connection, msgType uint16, body []byte) {
	switch msgType {
	case 0: // Heartbeat
		h.handleHeartbeat(conn)
	case 1: // Auth
		h.handleAuth(conn, body)
	default:
		// 转发到 Logic 层
		h.forwardToLogic(conn, msgType, body)
	}
}

func (h *Handler) handleHeartbeat(conn *connection.Connection) {
	// TODO: 回复心跳响应
	h.logger.Debug("Heartbeat received", zap.Int64("conn_id", conn.ID()))
}

func (h *Handler) handleAuth(conn *connection.Connection, body []byte) {
	// TODO: 解析 FlatBuffers 认证请求，验证 token
	h.logger.Debug("Auth request received", zap.Int64("conn_id", conn.ID()))
}

func (h *Handler) forwardToLogic(conn *connection.Connection, msgType uint16, body []byte) {
	// TODO: 封装消息并发送到 NATS
	h.natsClient.Publish("im.logic.upstream", body)
}

func (h *Handler) HandleDownstream(data []byte) {
	// TODO: 解析下行消息，找到目标连接并发送
	h.logger.Debug("Downstream message received", zap.Int("size", len(data)))
}
