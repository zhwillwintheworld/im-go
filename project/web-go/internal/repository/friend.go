package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sudooom.im.web/internal/model"
)

var (
	ErrFriendRequestNotFound = errors.New("friend request not found")
	ErrAlreadyFriends        = errors.New("already friends")
	ErrRequestPending        = errors.New("friend request pending")
)

// FriendRepository 好友数据访问
type FriendRepository struct {
	db *pgxpool.Pool
}

// NewFriendRepository 创建好友仓库
func NewFriendRepository(db *pgxpool.Pool) *FriendRepository {
	return &FriendRepository{db: db}
}

// CreateRequest 创建好友请求
func (r *FriendRepository) CreateRequest(ctx context.Context, request *model.FriendRequest) error {
	query := `
		INSERT INTO friend_requests (from_user_id, to_user_id, message, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(ctx, query,
		request.FromUserID,
		request.ToUserID,
		request.Message,
		request.Status,
	).Scan(&request.ID, &request.CreatedAt, &request.UpdatedAt)
}

// GetRequestByID 通过 ID 获取好友请求
func (r *FriendRepository) GetRequestByID(ctx context.Context, id int64) (*model.FriendRequest, error) {
	query := `
		SELECT id, from_user_id, to_user_id, message, status, created_at, updated_at
		FROM friend_requests WHERE id = $1
	`
	req := &model.FriendRequest{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&req.ID,
		&req.FromUserID,
		&req.ToUserID,
		&req.Message,
		&req.Status,
		&req.CreatedAt,
		&req.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrFriendRequestNotFound
		}
		return nil, err
	}
	return req, nil
}

// GetPendingRequest 获取待处理的好友请求
func (r *FriendRepository) GetPendingRequest(ctx context.Context, fromUserID, toUserID int64) (*model.FriendRequest, error) {
	query := `
		SELECT id, from_user_id, to_user_id, message, status, created_at, updated_at
		FROM friend_requests
		WHERE from_user_id = $1 AND to_user_id = $2 AND status = $3
	`
	req := &model.FriendRequest{}
	err := r.db.QueryRow(ctx, query, fromUserID, toUserID, model.FriendRequestPending).Scan(
		&req.ID,
		&req.FromUserID,
		&req.ToUserID,
		&req.Message,
		&req.Status,
		&req.CreatedAt,
		&req.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return req, nil
}

// UpdateRequestStatus 更新好友请求状态
func (r *FriendRepository) UpdateRequestStatus(ctx context.Context, id int64, status int) error {
	query := `UPDATE friend_requests SET status = $2, updated_at = NOW() WHERE id = $1`
	result, err := r.db.Exec(ctx, query, id, status)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrFriendRequestNotFound
	}
	return nil
}

// GetPendingRequestsForUser 获取用户待处理的好友请求
func (r *FriendRepository) GetPendingRequestsForUser(ctx context.Context, userID int64) ([]*model.FriendRequestWithUser, error) {
	query := `
		SELECT fr.id, fr.from_user_id, fr.to_user_id, fr.message, fr.status, fr.created_at, fr.updated_at,
		       u.username, u.nickname, u.avatar
		FROM friend_requests fr
		JOIN users u ON fr.from_user_id = u.id
		WHERE fr.to_user_id = $1 AND fr.status = $2
		ORDER BY fr.created_at DESC
	`
	rows, err := r.db.Query(ctx, query, userID, model.FriendRequestPending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []*model.FriendRequestWithUser
	for rows.Next() {
		req := &model.FriendRequestWithUser{}
		err := rows.Scan(
			&req.ID,
			&req.FromUserID,
			&req.ToUserID,
			&req.Message,
			&req.Status,
			&req.CreatedAt,
			&req.UpdatedAt,
			&req.FromUsername,
			&req.FromNickname,
			&req.FromAvatar,
		)
		if err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}
	return requests, nil
}

// CreateFriendship 创建好友关系（双向）
func (r *FriendRepository) CreateFriendship(ctx context.Context, userID, friendID int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `INSERT INTO friends (user_id, friend_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`

	// 添加双向好友关系
	if _, err := tx.Exec(ctx, query, userID, friendID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, query, friendID, userID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// DeleteFriendship 删除好友关系（双向）
func (r *FriendRepository) DeleteFriendship(ctx context.Context, userID, friendID int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `DELETE FROM friends WHERE user_id = $1 AND friend_id = $2`

	if _, err := tx.Exec(ctx, query, userID, friendID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, query, friendID, userID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// IsFriend 检查是否为好友
func (r *FriendRepository) IsFriend(ctx context.Context, userID, friendID int64) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM friends WHERE user_id = $1 AND friend_id = $2)`
	err := r.db.QueryRow(ctx, query, userID, friendID).Scan(&exists)
	return exists, err
}

// GetFriends 获取好友列表
func (r *FriendRepository) GetFriends(ctx context.Context, userID int64) ([]*model.FriendWithUser, error) {
	query := `
		SELECT f.id, f.user_id, f.friend_id, f.remark, f.created_at,
		       u.username, u.nickname, u.avatar
		FROM friends f
		JOIN users u ON f.friend_id = u.id
		WHERE f.user_id = $1
		ORDER BY f.created_at DESC
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var friends []*model.FriendWithUser
	for rows.Next() {
		f := &model.FriendWithUser{}
		err := rows.Scan(
			&f.ID,
			&f.UserID,
			&f.FriendID,
			&f.Remark,
			&f.CreatedAt,
			&f.Username,
			&f.Nickname,
			&f.Avatar,
		)
		if err != nil {
			return nil, err
		}
		friends = append(friends, f)
	}
	return friends, nil
}
