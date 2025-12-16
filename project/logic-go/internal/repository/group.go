package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"sudooom.im.logic/internal/model"
)

// GroupRepository 群组仓库
type GroupRepository struct {
	db *pgxpool.Pool
}

// NewGroupRepository 创建群组仓库
func NewGroupRepository(db *pgxpool.Pool) *GroupRepository {
	return &GroupRepository{db: db}
}

// FindByID 根据 ID 查找群组
func (r *GroupRepository) FindByID(ctx context.Context, id int64) (*model.Group, error) {
	query := `
		SELECT id, name, owner_id, avatar, description, member_count, status, created_at, updated_at
		FROM groups WHERE id = $1
	`

	var group model.Group
	err := r.db.QueryRow(ctx, query, id).Scan(
		&group.Id,
		&group.Name,
		&group.OwnerId,
		&group.Avatar,
		&group.Description,
		&group.MemberCount,
		&group.Status,
		&group.CreatedAt,
		&group.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &group, nil
}

// GetMembers 获取群成员
func (r *GroupRepository) GetMembers(ctx context.Context, groupId int64) ([]int64, error) {
	query := `SELECT user_id FROM group_members WHERE group_id = $1`

	rows, err := r.db.Query(ctx, query, groupId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []int64
	for rows.Next() {
		var userId int64
		if err := rows.Scan(&userId); err != nil {
			continue
		}
		members = append(members, userId)
	}

	return members, nil
}

// IsMember 检查用户是否为群成员
func (r *GroupRepository) IsMember(ctx context.Context, groupId, userId int64) (bool, error) {
	query := `SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2 LIMIT 1`

	var exists int
	err := r.db.QueryRow(ctx, query, groupId, userId).Scan(&exists)
	if err != nil {
		return false, nil
	}

	return true, nil
}
