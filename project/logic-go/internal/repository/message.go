package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"sudooom.im.logic/internal/model"
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
		INSERT INTO messages (client_msg_id, from_user_id, to_user_id, to_group_id, msg_type, content, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	var id int64
	err := r.db.QueryRow(ctx, query,
		msg.ClientMsgId,
		msg.FromUserId,
		msg.ToUserId,
		msg.ToGroupId,
		msg.MsgType,
		msg.Content,
		msg.Status,
		msg.CreatedAt,
	).Scan(&id)

	return id, err
}

// FindByID 根据 ID 查找消息
func (r *MessageRepository) FindByID(ctx context.Context, id int64) (*model.Message, error) {
	query := `
		SELECT id, client_msg_id, from_user_id, to_user_id, to_group_id, msg_type, content, status, created_at
		FROM messages WHERE id = $1
	`

	var msg model.Message
	err := r.db.QueryRow(ctx, query, id).Scan(
		&msg.Id,
		&msg.ClientMsgId,
		&msg.FromUserId,
		&msg.ToUserId,
		&msg.ToGroupId,
		&msg.MsgType,
		&msg.Content,
		&msg.Status,
		&msg.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &msg, nil
}

// UpdateStatus 更新消息状态
func (r *MessageRepository) UpdateStatus(ctx context.Context, id int64, status int) error {
	query := `UPDATE messages SET status = $2 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id, status)
	return err
}
