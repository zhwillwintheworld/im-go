package service

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"sudooom.im.logic/internal/model"
	sharedRedis "sudooom.im.shared/redis"
)

// ConversationService 会话服务（基于 Redis）
type ConversationService struct {
	redisClient *redis.Client
	logger      *slog.Logger
}

// NewConversationService 创建会话服务
func NewConversationService(redisClient *redis.Client) *ConversationService {
	return &ConversationService{
		redisClient: redisClient,
		logger:      slog.Default(),
	}
}

// UpdateConversationForSender 更新发送者的会话（发消息时）
func (s *ConversationService) UpdateConversationForSender(ctx context.Context, userId, peerId, groupId, msgId int64) error {
	now := time.Now().UnixMilli()

	var convKey, member string
	if peerId > 0 {
		convKey = sharedRedis.BuildConversationPeerKey(userId, peerId)
		member = sharedRedis.BuildConversationPeerMember(peerId)
	} else {
		convKey = sharedRedis.BuildConversationGroupKey(userId, groupId)
		member = sharedRedis.BuildConversationGroupMember(groupId)
	}
	idxKey := sharedRedis.BuildConversationIndexKey(userId)

	pipe := s.redisClient.Pipeline()
	pipe.HSet(ctx, convKey, "last_msg_id", msgId, "update_at", now)
	pipe.ZAdd(ctx, idxKey, redis.Z{Score: float64(now), Member: member})
	_, err := pipe.Exec(ctx)

	return err
}

// UpdateConversationForReceiver 更新接收者的会话（收到消息时）
func (s *ConversationService) UpdateConversationForReceiver(ctx context.Context, userId, peerId, groupId, msgId int64) error {
	now := time.Now().UnixMilli()

	var convKey, member string
	if peerId > 0 {
		convKey = sharedRedis.BuildConversationPeerKey(userId, peerId)
		member = sharedRedis.BuildConversationPeerMember(peerId)
	} else {
		convKey = sharedRedis.BuildConversationGroupKey(userId, groupId)
		member = sharedRedis.BuildConversationGroupMember(groupId)
	}
	idxKey := sharedRedis.BuildConversationIndexKey(userId)

	pipe := s.redisClient.Pipeline()
	pipe.HSet(ctx, convKey, "last_msg_id", msgId, "update_at", now)
	pipe.HIncrBy(ctx, convKey, "unread_count", 1)
	pipe.ZAdd(ctx, idxKey, redis.Z{Score: float64(now), Member: member})
	_, err := pipe.Exec(ctx)

	return err
}

// UpdateConversationForGroupMembers 批量更新群成员会话
func (s *ConversationService) UpdateConversationForGroupMembers(ctx context.Context, memberIds []int64, senderId, groupId, msgId int64) error {
	now := time.Now().UnixMilli()
	member := sharedRedis.BuildConversationGroupMember(groupId)

	pipe := s.redisClient.Pipeline()
	for _, userId := range memberIds {
		convKey := sharedRedis.BuildConversationGroupKey(userId, groupId)
		idxKey := sharedRedis.BuildConversationIndexKey(userId)

		pipe.HSet(ctx, convKey, "last_msg_id", msgId, "update_at", now)
		if userId != senderId {
			pipe.HIncrBy(ctx, convKey, "unread_count", 1)
		}
		pipe.ZAdd(ctx, idxKey, redis.Z{Score: float64(now), Member: member})
	}
	_, err := pipe.Exec(ctx)

	return err
}

// MarkRead 标记会话已读
func (s *ConversationService) MarkRead(ctx context.Context, userId, peerId, groupId, lastReadMsgId int64) error {
	var convKey string
	if peerId > 0 {
		convKey = sharedRedis.BuildConversationPeerKey(userId, peerId)
	} else {
		convKey = sharedRedis.BuildConversationGroupKey(userId, groupId)
	}

	return s.redisClient.HSet(ctx, convKey, "unread_count", 0, "last_read_msg_id", lastReadMsgId).Err()
}

// GetUserConversations 获取用户会话列表
func (s *ConversationService) GetUserConversations(ctx context.Context, userId int64, offset, limit int64) ([]model.Conversation, error) {
	idxKey := sharedRedis.BuildConversationIndexKey(userId)

	// 获取会话索引（按更新时间倒序）
	members, err := s.redisClient.ZRevRange(ctx, idxKey, offset, offset+limit-1).Result()
	if err != nil {
		return nil, err
	}

	if len(members) == 0 {
		return []model.Conversation{}, nil
	}

	// Pipeline 批量获取会话详情
	pipe := s.redisClient.Pipeline()
	cmds := make([]*redis.MapStringStringCmd, len(members))

	for i, m := range members {
		var convKey string
		peerId, groupId := s.parseMember(m)
		if peerId > 0 {
			convKey = sharedRedis.BuildConversationPeerKey(userId, peerId)
		} else {
			convKey = sharedRedis.BuildConversationGroupKey(userId, groupId)
		}
		cmds[i] = pipe.HGetAll(ctx, convKey)
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	// 解析结果
	conversations := make([]model.Conversation, 0, len(members))
	for i, cmd := range cmds {
		data, err := cmd.Result()
		if err != nil || len(data) == 0 {
			continue
		}

		peerId, groupId := s.parseMember(members[i])
		conv := model.Conversation{
			PeerID:        peerId,
			GroupID:       groupId,
			LastMsgID:     s.parseInt64(data["last_msg_id"]),
			LastReadMsgID: s.parseInt64(data["last_read_msg_id"]),
			UnreadCount:   int(s.parseInt64(data["unread_count"])),
			IsPinned:      data["is_pinned"] == "1",
			IsMuted:       data["is_muted"] == "1",
			UpdateAt:      s.parseInt64(data["update_at"]),
		}
		conversations = append(conversations, conv)
	}

	return conversations, nil
}

// parseMember 解析 member，返回 peerId, groupId
func (s *ConversationService) parseMember(member string) (peerId, groupId int64) {
	if len(member) < 3 {
		return 0, 0
	}
	id, _ := strconv.ParseInt(member[2:], 10, 64)
	if member[0] == 'p' {
		return id, 0
	}
	return 0, id
}

func (s *ConversationService) parseInt64(str string) int64 {
	v, _ := strconv.ParseInt(str, 10, 64)
	return v
}

// GetTotalUnreadCount 获取用户总未读数
func (s *ConversationService) GetTotalUnreadCount(ctx context.Context, userId int64) (int64, error) {
	idxKey := sharedRedis.BuildConversationIndexKey(userId)

	members, err := s.redisClient.ZRange(ctx, idxKey, 0, -1).Result()
	if err != nil {
		return 0, err
	}

	if len(members) == 0 {
		return 0, nil
	}

	// Pipeline 批量获取未读数
	pipe := s.redisClient.Pipeline()
	cmds := make([]*redis.StringCmd, len(members))

	for i, m := range members {
		var convKey string
		peerId, groupId := s.parseMember(m)
		if peerId > 0 {
			convKey = sharedRedis.BuildConversationPeerKey(userId, peerId)
		} else {
			convKey = sharedRedis.BuildConversationGroupKey(userId, groupId)
		}
		cmds[i] = pipe.HGet(ctx, convKey, "unread_count")
	}

	_, _ = pipe.Exec(ctx)

	var total int64
	for _, cmd := range cmds {
		count, err := cmd.Int64()
		if err == nil {
			total += count
		}
	}

	return total, nil
}
