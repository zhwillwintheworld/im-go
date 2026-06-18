package service

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultGroupMemberCacheTTL = 500 * time.Millisecond

type cachedGroupMembers struct {
	Members   []int64
	ExpiresAt time.Time
}

// GroupService 群组服务
type GroupService struct {
	db             *pgxpool.Pool
	logger         *slog.Logger
	memberCache    sync.Map
	memberCacheTTL time.Duration
}

// NewGroupService 创建群组服务
func NewGroupService(db *pgxpool.Pool) *GroupService {
	return NewGroupServiceWithCacheTTL(db, defaultGroupMemberCacheTTL)
}

// NewGroupServiceWithCacheTTL 创建带群成员短缓存的群组服务
func NewGroupServiceWithCacheTTL(db *pgxpool.Pool, cacheTTL time.Duration) *GroupService {
	if cacheTTL <= 0 {
		cacheTTL = defaultGroupMemberCacheTTL
	}
	return &GroupService{
		db:             db,
		logger:         slog.Default(),
		memberCacheTTL: cacheTTL,
	}
}

// GetGroupMembers 获取群成员列表
func (s *GroupService) GetGroupMembers(ctx context.Context, groupId int64) ([]int64, error) {
	if cached, ok := s.memberCache.Load(groupId); ok {
		entry := cached.(*cachedGroupMembers)
		if time.Now().Before(entry.ExpiresAt) {
			return cloneInt64s(entry.Members), nil
		}
		s.memberCache.Delete(groupId)
	}

	if s.db == nil {
		return nil, errors.New("group service db is nil")
	}

	query := `SELECT user_id FROM group_members WHERE group_id = $1 AND deleted = 0`

	rows, err := s.db.Query(ctx, query, groupId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []int64
	for rows.Next() {
		var userId int64
		if err := rows.Scan(&userId); err != nil {
			s.logger.Warn("Failed to scan group member", "groupId", groupId, "error", err)
			continue
		}
		members = append(members, userId)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	s.memberCache.Store(groupId, &cachedGroupMembers{
		Members:   cloneInt64s(members),
		ExpiresAt: time.Now().Add(s.memberCacheTTL),
	})

	return members, nil
}

// IsGroupMember 检查用户是否为群成员
func (s *GroupService) IsGroupMember(ctx context.Context, groupId, userId int64) (bool, error) {
	query := `SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2 AND deleted = 0 LIMIT 1`

	var exists int
	err := s.db.QueryRow(ctx, query, groupId, userId).Scan(&exists)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// GetGroupMemberCount 获取群成员数量
func (s *GroupService) GetGroupMemberCount(ctx context.Context, groupId int64) (int, error) {
	query := `SELECT COUNT(*) FROM group_members WHERE group_id = $1 AND deleted = 0`

	var count int
	err := s.db.QueryRow(ctx, query, groupId).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// InvalidateGroupMemberCache 失效指定群成员缓存。
func (s *GroupService) InvalidateGroupMemberCache(groupId int64) {
	s.memberCache.Delete(groupId)
}

func cloneInt64s(values []int64) []int64 {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]int64, len(values))
	copy(cloned, values)
	return cloned
}
