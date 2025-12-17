package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sudooom.im.logic/internal/model"
)

var (
	ErrMessageNotFound = errors.New("message not found")
)

// MessageRepository 消息仓库
type MessageRepository struct {
	db *pgxpool.Pool
}

// NewMessageRepository 创建消息仓库
func NewMessageRepository(db *pgxpool.Pool) *MessageRepository {
	return &MessageRepository{db: db}
}

// Create 创建消息
func (r *MessageRepository) Create(ctx context.Context, msg *model.Message) (int64, error) {
	query := `
		INSERT INTO messages (object_code, client_msg_id, from_user_id, to_user_id, to_group_id, msg_type, content, status, create_at, update_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		RETURNING id
	`

	var id int64
	err := r.db.QueryRow(ctx, query,
		msg.ObjectCode,
		msg.ClientMsgId,
		msg.FromUserId,
		msg.ToUserId,
		msg.ToGroupId,
		msg.MsgType,
		msg.Content,
		msg.Status,
	).Scan(&id)

	return id, err
}

// FindByID 根据 ID 查找消息
func (r *MessageRepository) FindByID(ctx context.Context, id int64) (*model.Message, error) {
	query := `
		SELECT id, object_code, client_msg_id, from_user_id, to_user_id, to_group_id, msg_type, content, status, create_at, update_at
		FROM messages WHERE id = $1 AND deleted = 0
	`

	var msg model.Message
	err := r.db.QueryRow(ctx, query, id).Scan(
		&msg.Id,
		&msg.ObjectCode,
		&msg.ClientMsgId,
		&msg.FromUserId,
		&msg.ToUserId,
		&msg.ToGroupId,
		&msg.MsgType,
		&msg.Content,
		&msg.Status,
		&msg.CreateAt,
		&msg.UpdateAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}

	return &msg, nil
}

// FindByObjectCode 根据 ObjectCode 查找消息
func (r *MessageRepository) FindByObjectCode(ctx context.Context, objectCode string) (*model.Message, error) {
	query := `
		SELECT id, object_code, client_msg_id, from_user_id, to_user_id, to_group_id, msg_type, content, status, create_at, update_at
		FROM messages WHERE object_code = $1 AND deleted = 0
	`

	var msg model.Message
	err := r.db.QueryRow(ctx, query, objectCode).Scan(
		&msg.Id,
		&msg.ObjectCode,
		&msg.ClientMsgId,
		&msg.FromUserId,
		&msg.ToUserId,
		&msg.ToGroupId,
		&msg.MsgType,
		&msg.Content,
		&msg.Status,
		&msg.CreateAt,
		&msg.UpdateAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}

	return &msg, nil
}

// UpdateStatus 更新消息状态
func (r *MessageRepository) UpdateStatus(ctx context.Context, id int64, status int) error {
	query := `UPDATE messages SET status = $2, update_at = NOW() WHERE id = $1 AND deleted = 0`
	result, err := r.db.Exec(ctx, query, id, status)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrMessageNotFound
	}
	return nil
}

// CreateOfflineMessage 创建离线消息
func (r *MessageRepository) CreateOfflineMessage(ctx context.Context, offline *model.OfflineMessage) error {
	query := `
		INSERT INTO offline_messages (object_code, user_id, message_id, create_at, update_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (user_id, message_id) DO NOTHING
	`
	_, err := r.db.Exec(ctx, query, offline.ObjectCode, offline.UserId, offline.MessageId)
	return err
}

// GetOfflineMessages 获取用户离线消息
func (r *MessageRepository) GetOfflineMessages(ctx context.Context, userID int64, limit int) ([]*model.Message, error) {
	query := `
		SELECT m.id, m.object_code, m.client_msg_id, m.from_user_id, m.to_user_id, m.to_group_id, m.msg_type, m.content, m.status, m.create_at, m.update_at
		FROM offline_messages om
		JOIN messages m ON om.message_id = m.id
		WHERE om.user_id = $1 AND om.deleted = 0 AND m.deleted = 0
		ORDER BY om.create_at ASC
		LIMIT $2
	`
	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*model.Message
	for rows.Next() {
		msg := &model.Message{}
		err := rows.Scan(
			&msg.Id,
			&msg.ObjectCode,
			&msg.ClientMsgId,
			&msg.FromUserId,
			&msg.ToUserId,
			&msg.ToGroupId,
			&msg.MsgType,
			&msg.Content,
			&msg.Status,
			&msg.CreateAt,
			&msg.UpdateAt,
		)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

// DeleteOfflineMessage 删除离线消息（逻辑删除）
func (r *MessageRepository) DeleteOfflineMessage(ctx context.Context, userID, messageID int64) error {
	query := `UPDATE offline_messages SET deleted = 1, update_at = NOW() WHERE user_id = $1 AND message_id = $2 AND deleted = 0`
	_, err := r.db.Exec(ctx, query, userID, messageID)
	return err
}

// DeleteAllOfflineMessages 删除用户所有离线消息（逻辑删除）
func (r *MessageRepository) DeleteAllOfflineMessages(ctx context.Context, userID int64) error {
	query := `UPDATE offline_messages SET deleted = 1, update_at = NOW() WHERE user_id = $1 AND deleted = 0`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}
