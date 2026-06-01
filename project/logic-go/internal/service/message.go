package service

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"sudooom.im.shared/proto"
	"sudooom.im.shared/snowflake"
)

// MessageService 消息服务
type MessageService struct {
	db     *pgxpool.Pool
	sf     *snowflake.Node
	logger *slog.Logger
}

// NewMessageService 创建消息服务
func NewMessageService(db *pgxpool.Pool, sf *snowflake.Node) *MessageService {
	return &MessageService{
		db:     db,
		sf:     sf,
		logger: slog.Default(),
	}
}

// SaveMessage 保存消息
func (s *MessageService) SaveMessage(ctx context.Context, msg *proto.UserMessage) (int64, error) {
	serverMsgId := s.sf.Generate().Int64()
	objectCode := strconv.FormatInt(serverMsgId, 10)
	query := `
		INSERT INTO messages (id, object_code, client_msg_id, from_user_id, to_user_id, to_group_id, msg_type, content, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
		RETURNING id
	`

	err := s.db.QueryRow(ctx, query,
		serverMsgId,
		objectCode,
		msg.ClientMsgId,
		msg.FromUserId,
		msg.ToUserId,
		msg.ToGroupId,
		msg.MsgType,
		msg.Content,
		0, // status: 正常
		time.Now(),
	).Scan(&serverMsgId)

	if err != nil {
		s.logger.Error("Failed to save message", "error", err)
		return 0, err
	}

	s.logger.Debug("Message saved",
		"serverMsgId", serverMsgId,
		"clientMsgId", msg.ClientMsgId,
		"fromUserId", msg.FromUserId)

	return serverMsgId, nil
}

// GetMessage 获取消息
func (s *MessageService) GetMessage(ctx context.Context, msgId int64) (*proto.PushMessage, error) {
	query := `
		SELECT id, from_user_id, to_user_id, to_group_id, msg_type, content, created_at
		FROM messages WHERE id = $1
	`

	var msg proto.PushMessage
	var createdAt time.Time
	err := s.db.QueryRow(ctx, query, msgId).Scan(
		&msg.ServerMsgId,
		&msg.FromUserId,
		&msg.ToUserId,
		&msg.ToGroupId,
		&msg.MsgType,
		&msg.Content,
		&createdAt,
	)

	if err != nil {
		return nil, err
	}

	msg.Timestamp = createdAt.UnixMilli()
	return &msg, nil
}

// UpdateMessageStatus 更新消息状态
func (s *MessageService) UpdateMessageStatus(ctx context.Context, msgId int64, status int) error {
	query := `UPDATE messages SET status = $2, updated_at = NOW() WHERE id = $1`
	_, err := s.db.Exec(ctx, query, msgId, status)
	return err
}
