package service

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// GroupService 群组服务
type GroupService struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

// NewGroupService 创建群组服务
func NewGroupService(db *pgxpool.Pool) *GroupService {
	return &GroupService{
		db:     db,
		logger: slog.Default(),
	}
}

// GetGroupMembers 获取群成员列表
func (s *GroupService) GetGroupMembers(ctx context.Context, groupId int64) ([]int64, error) {
	query := `SELECT user_id FROM group_members WHERE group_id = $1`

	rows, err := s.db.Query(ctx, query, groupId)
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

// IsGroupMember 检查用户是否为群成员
func (s *GroupService) IsGroupMember(ctx context.Context, groupId, userId int64) (bool, error) {
	query := `SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2 LIMIT 1`

	var exists int
	err := s.db.QueryRow(ctx, query, groupId, userId).Scan(&exists)
	if err != nil {
		return false, nil // 用户不在群里
	}

	return true, nil
}

// GetGroupMemberCount 获取群成员数量
func (s *GroupService) GetGroupMemberCount(ctx context.Context, groupId int64) (int, error) {
	query := `SELECT COUNT(*) FROM group_members WHERE group_id = $1`

	var count int
	err := s.db.QueryRow(ctx, query, groupId).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}
