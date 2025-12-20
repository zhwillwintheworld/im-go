package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sudooom.im.web/internal/model"
)

var (
	ErrUserNotFound   = errors.New("user not found")
	ErrUsernameExists = errors.New("username already exists")
)

// UserRepository 用户数据访问
type UserRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository 创建用户仓库
func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// Create 创建用户
func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (id, username, password_hash, nickname, avatar, status, create_at, update_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING create_at, update_at
	`
	return r.db.QueryRow(ctx, query,
		user.ID,
		user.Username,
		user.PasswordHash,
		user.Nickname,
		user.Avatar,
		user.Status,
	).Scan(&user.CreateAt, &user.UpdateAt)
}

// GetByID 通过 ID 获取用户
func (r *UserRepository) GetByID(ctx context.Context, id int64) (*model.User, error) {
	query := `
		SELECT id, username, password_hash, nickname, avatar, status, create_at, update_at
		FROM users WHERE id = $1 AND deleted = 0
	`
	user := &model.User{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Nickname,
		&user.Avatar,
		&user.Status,
		&user.CreateAt,
		&user.UpdateAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// GetByUsername 通过用户名获取用户
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	query := `
		SELECT id, username, password_hash, nickname, avatar, status, create_at, update_at
		FROM users WHERE username = $1 AND deleted = 0
	`
	user := &model.User{}
	err := r.db.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Nickname,
		&user.Avatar,
		&user.Status,
		&user.CreateAt,
		&user.UpdateAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// ExistsByUsername 检查用户名是否存在
func (r *UserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1 AND deleted = 0)`
	err := r.db.QueryRow(ctx, query, username).Scan(&exists)
	return exists, err
}

// Update 更新用户信息
func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users SET nickname = $2, avatar = $3, update_at = NOW()
		WHERE id = $1 AND deleted = 0
	`
	result, err := r.db.Exec(ctx, query,
		user.ID,
		user.Nickname,
		user.Avatar,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// Search 搜索用户
func (r *UserRepository) Search(ctx context.Context, keyword string, limit, offset int) ([]*model.User, error) {
	query := `
		SELECT id, username, nickname, avatar, status, create_at, update_at
		FROM users
		WHERE (username ILIKE $1 OR nickname ILIKE $1) AND deleted = 0
		ORDER BY id DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, "%"+keyword+"%", limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		user := &model.User{}
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Nickname,
			&user.Avatar,
			&user.Status,
			&user.CreateAt,
			&user.UpdateAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

// Delete 逻辑删除用户
func (r *UserRepository) Delete(ctx context.Context, id int64) error {
	query := `UPDATE users SET deleted = 1, update_at = NOW() WHERE id = $1 AND deleted = 0`
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}
