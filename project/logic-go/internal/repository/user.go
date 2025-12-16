package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"sudooom.im.logic/internal/model"
)

// UserRepository 用户仓库
type UserRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository 创建用户仓库
func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// FindByID 根据 ID 查找用户
func (r *UserRepository) FindByID(ctx context.Context, id int64) (*model.User, error) {
	query := `
		SELECT id, username, nickname, avatar, status, created_at, updated_at
		FROM users WHERE id = $1
	`

	var user model.User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.Id,
		&user.Username,
		&user.Nickname,
		&user.Avatar,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// FindByUsername 根据用户名查找用户
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	query := `
		SELECT id, username, nickname, avatar, status, created_at, updated_at
		FROM users WHERE username = $1
	`

	var user model.User
	err := r.db.QueryRow(ctx, query, username).Scan(
		&user.Id,
		&user.Username,
		&user.Nickname,
		&user.Avatar,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}
