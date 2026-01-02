package handler

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	// Added by user instruction
	// Added by user instruction
	"github.com/quic-go/webtransport-go"
	"sudooom.im.access/internal/connection"
	im_protocol "sudooom.im.access/pkg/flatbuf/im/protocol"
	"sudooom.im.shared/proto"
)

// HandleFirstStream 处理首个数据流，必须是认证请求
// 返回 error 表示认证失败，调用方应关闭连接
// 认证成功后流保持打开，调用方应继续在此流上处理后续消息
func (h *Handler) HandleFirstStream(ctx context.Context, conn *connection.Connection, stream *webtransport.Stream) error {
	// 注意：不再 defer close，因为要复用这个流

	// 读取帧头
	header := make([]byte, FrameHeaderSize)
	if _, err := io.ReadFull(stream, header); err != nil {
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
	// Auth request processing

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

	// 验证 token 是否是该用户该 platform 当前有效的 token（被下线的 token 不能连接）
	isCurrent, err := h.redisClient.IsTokenCurrent(ctx, userInfo.UserID, userInfo.Platform, token)
	if err != nil {
		h.logger.Error("Failed to check token validity", "error", err)
		h.sendClientResponse(stream, "", im_protocol.ErrorCodeUNKNOWN_ERROR, "internal error", im_protocol.ResponsePayloadNONE, nil)
		return fmt.Errorf("redis error: %w", err)
	}
	if !isCurrent {
		h.logger.Warn("Token is not current", "conn_id", conn.ID(), "user_id", userInfo.UserID)
		h.sendClientResponse(stream, "", im_protocol.ErrorCodeAUTH_FAILED, "token expired or replaced", im_protocol.ResponsePayloadNONE, nil)
		return fmt.Errorf("token is not current")
	}

	// 构建 sessInfo 使用 Redis 中的用户信息
	sessInfo := &connection.SessionInfo{
		UserID:   userInfo.UserID,
		DeviceID: deviceID,
		Platform: platform.String(),
	}
	conn.BindSession(sessInfo)

	// 绑定用户到连接管理器（按平台）
	// 如果该用户在同一平台已有连接，会返回旧连接
	oldConn := h.connMgr.BindUser(conn.ID(), sessInfo.UserID, sessInfo.Platform)
	if oldConn != nil {
		// 踢掉同平台的旧连接
		h.logger.Info("Kicking old connection on same platform",
			"user_id", sessInfo.UserID,
			"platform", sessInfo.Platform,
			"old_conn_id", oldConn.ID(),
			"new_conn_id", conn.ID())
		// 旧连接会在 server 的 defer 中自动清理
		oldConn.Close()
	}

	// 注册用户位置到 Redis（包含 connId）
	if err := h.redisClient.RegisterUserLocation(ctx, sessInfo.UserID, sessInfo.Platform, conn.ID()); err != nil {
		h.logger.Error("Failed to register user location", "error", err)
	}

	// 发送上线通知到 Logic
	h.sendUserOnlineToLogic(conn, sessInfo)

	// 发送认证成功响应
	h.sendClientResponse(stream, "", im_protocol.ErrorCodeSUCCESS, "success", im_protocol.ResponsePayloadNONE, nil)
	h.logger.Info("User authenticated", "conn_id", conn.ID(), "user_id", userInfo.UserID)

	return nil
}

// sendUserOnlineToLogic 发送用户上线事件到 Logic
func (h *Handler) sendUserOnlineToLogic(conn *connection.Connection, sessInfo *connection.SessionInfo) {
	msg := h.buildUpstreamMessage(conn, proto.UpstreamPayload{
		UserOnline: &proto.UserOnline{
			UserId:   sessInfo.UserID,
			ConnId:   conn.ID(),
			DeviceId: sessInfo.DeviceID,
			Platform: sessInfo.Platform,
		},
	})

	if err := h.publishUpstream(msg); err != nil {
		h.logger.Error("Failed to publish user online event", "error", err)
	}
}

// SendUserOfflineToLogic 发送用户下线通知
func (h *Handler) SendUserOfflineToLogic(conn *connection.Connection) {
	if conn.UserID() == 0 {
		return
	}

	// 注意：这里不能使用 buildUpstreamMessage，因为连接可能已经不存在了
	// 所以手动构建，只填充必需的字段
	msg := &proto.UpstreamMessage{
		AccessNodeId: h.nodeID,
		Payload: proto.UpstreamPayload{
			UserOffline: &proto.UserOffline{
				UserId: conn.UserID(),
				ConnId: conn.ID(),
			},
		},
	}

	if err := h.publishUpstream(msg); err != nil {
		h.logger.Error("Failed to publish user offline event", "error", err)
	}
}
