package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sudooom.im.web/internal/model"
)

var (
	ErrGroupNotFound       = errors.New("group not found")
	ErrGroupMemberNotFound = errors.New("group member not found")
	ErrAlreadyGroupMember  = errors.New("already group member")
)

// GroupRepository 群组数据访问
type GroupRepository struct {
	db *pgxpool.Pool
}

// NewGroupRepository 创建群组仓库
func NewGroupRepository(db *pgxpool.Pool) *GroupRepository {
	return &GroupRepository{db: db}
}

// Create 创建群组
func (r *GroupRepository) Create(ctx context.Context, group *model.Group) error {
	query := `
		INSERT INTO groups (id, name, owner_id, avatar, description, max_members, status, create_at, update_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING create_at, update_at
	`
	return r.db.QueryRow(ctx, query,
		group.ID,
		group.Name,
		group.OwnerID,
		group.Avatar,
		group.Description,
		group.MaxMembers,
		group.Status,
	).Scan(&group.CreateAt, &group.UpdateAt)
}

// GetByID 通过 ID 获取群组
func (r *GroupRepository) GetByID(ctx context.Context, id int64) (*model.Group, error) {
	query := `
		SELECT id, name, owner_id, avatar, description, max_members, status, create_at, update_at
		FROM groups WHERE id = $1 AND deleted = 0
	`
	group := &model.Group{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&group.ID,
		&group.Name,
		&group.OwnerID,
		&group.Avatar,
		&group.Description,
		&group.MaxMembers,
		&group.Status,
		&group.CreateAt,
		&group.UpdateAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}
	return group, nil
}

// Update 更新群组信息
func (r *GroupRepository) Update(ctx context.Context, group *model.Group) error {
	query := `
		UPDATE groups SET name = $2, avatar = $3, description = $4, update_at = NOW()
		WHERE id = $1 AND deleted = 0
	`
	result, err := r.db.Exec(ctx, query,
		group.ID,
		group.Name,
		group.Avatar,
		group.Description,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrGroupNotFound
	}
	return nil
}

// Delete 逻辑删除群组
func (r *GroupRepository) Delete(ctx context.Context, id int64) error {
	query := `UPDATE groups SET deleted = 1, status = 1, update_at = NOW() WHERE id = $1 AND deleted = 0`
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrGroupNotFound
	}
	return nil
}

// GetMemberCount 获取群成员数量
func (r *GroupRepository) GetMemberCount(ctx context.Context, groupID int64) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM group_members WHERE group_id = $1 AND deleted = 0`
	err := r.db.QueryRow(ctx, query, groupID).Scan(&count)
	return count, err
}

// GetUserGroups 获取用户加入的群组列表
func (r *GroupRepository) GetUserGroups(ctx context.Context, userID int64) ([]*model.GroupWithMemberCount, error) {
	query := `
		SELECT g.id, g.name, g.owner_id, g.avatar, g.description, g.max_members, g.status, g.create_at, g.update_at,
		       (SELECT COUNT(*) FROM group_members gm WHERE gm.group_id = g.id AND gm.deleted = 0) as member_count
		FROM groups g
		JOIN group_members m ON g.id = m.group_id
		WHERE m.user_id = $1 AND g.deleted = 0 AND m.deleted = 0
		ORDER BY g.create_at DESC
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []*model.GroupWithMemberCount
	for rows.Next() {
		g := &model.GroupWithMemberCount{}
		err := rows.Scan(
			&g.ID,
			&g.Name,
			&g.OwnerID,
			&g.Avatar,
			&g.Description,
			&g.MaxMembers,
			&g.Status,
			&g.CreateAt,
			&g.UpdateAt,
			&g.MemberCount,
		)
		if err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, nil
}

// AddMember 添加群成员
func (r *GroupRepository) AddMember(ctx context.Context, member *model.GroupMember) error {
	query := `
		INSERT INTO group_members (id, group_id, user_id, role, nickname, create_at, update_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (group_id, user_id) DO NOTHING
		RETURNING create_at, update_at
	`
	err := r.db.QueryRow(ctx, query,
		member.ID,
		member.GroupID,
		member.UserID,
		member.Role,
		member.Nickname,
	).Scan(&member.CreateAt, &member.UpdateAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAlreadyGroupMember
		}
		return err
	}
	return nil
}

// RemoveMember 移除群成员（逻辑删除）
func (r *GroupRepository) RemoveMember(ctx context.Context, groupID, userID int64) error {
	query := `UPDATE group_members SET deleted = 1, update_at = NOW() WHERE group_id = $1 AND user_id = $2 AND deleted = 0`
	result, err := r.db.Exec(ctx, query, groupID, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrGroupMemberNotFound
	}
	return nil
}

// GetMember 获取群成员信息
func (r *GroupRepository) GetMember(ctx context.Context, groupID, userID int64) (*model.GroupMember, error) {
	query := `
		SELECT id, group_id, user_id, role, nickname, create_at, update_at
		FROM group_members WHERE group_id = $1 AND user_id = $2 AND deleted = 0
	`
	member := &model.GroupMember{}
	err := r.db.QueryRow(ctx, query, groupID, userID).Scan(
		&member.ID,
		&member.GroupID,
		&member.UserID,
		&member.Role,
		&member.Nickname,
		&member.CreateAt,
		&member.UpdateAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGroupMemberNotFound
		}
		return nil, err
	}
	return member, nil
}

// GetMembers 获取群成员列表
func (r *GroupRepository) GetMembers(ctx context.Context, groupID int64) ([]*model.GroupMemberWithUser, error) {
	query := `
		SELECT gm.id, gm.group_id, gm.user_id, gm.role, gm.nickname, gm.create_at, gm.update_at,
		       u.username, u.nickname as user_nickname, u.avatar
		FROM group_members gm
		JOIN users u ON gm.user_id = u.id
		WHERE gm.group_id = $1 AND gm.deleted = 0 AND u.deleted = 0
		ORDER BY gm.role DESC, gm.create_at ASC
	`
	rows, err := r.db.Query(ctx, query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*model.GroupMemberWithUser
	for rows.Next() {
		m := &model.GroupMemberWithUser{}
		var userNickname string
		err := rows.Scan(
			&m.ID,
			&m.GroupID,
			&m.UserID,
			&m.Role,
			&m.Nickname,
			&m.CreateAt,
			&m.UpdateAt,
			&m.Username,
			&userNickname,
			&m.Avatar,
		)
		if err != nil {
			return nil, err
		}
		// 注意：Nickname 字段是群内昵称，需要额外存储用户昵称
		members = append(members, m)
	}
	return members, nil
}

// UpdateMemberRole 更新群成员角色
func (r *GroupRepository) UpdateMemberRole(ctx context.Context, groupID, userID int64, role int) error {
	query := `UPDATE group_members SET role = $3, update_at = NOW() WHERE group_id = $1 AND user_id = $2 AND deleted = 0`
	result, err := r.db.Exec(ctx, query, groupID, userID, role)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrGroupMemberNotFound
	}
	return nil
}

// UpdateMemberNickname 更新群成员昵称
func (r *GroupRepository) UpdateMemberNickname(ctx context.Context, groupID, userID int64, nickname string) error {
	query := `UPDATE group_members SET nickname = $3, update_at = NOW() WHERE group_id = $1 AND user_id = $2 AND deleted = 0`
	result, err := r.db.Exec(ctx, query, groupID, userID, nickname)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrGroupMemberNotFound
	}
	return nil
}

// IsMember 检查是否为群成员
func (r *GroupRepository) IsMember(ctx context.Context, groupID, userID int64) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2 AND deleted = 0)`
	err := r.db.QueryRow(ctx, query, groupID, userID).Scan(&exists)
	return exists, err
}

// GetGroupMemberIDs 获取群所有成员ID
func (r *GroupRepository) GetGroupMemberIDs(ctx context.Context, groupID int64) ([]int64, error) {
	query := `SELECT user_id FROM group_members WHERE group_id = $1 AND deleted = 0`
	rows, err := r.db.Query(ctx, query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userIDs []int64
	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}
	return userIDs, nil
}
