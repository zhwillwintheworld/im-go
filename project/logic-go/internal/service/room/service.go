package room

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"sudooom.im.logic/internal/service"
	"sudooom.im.shared/model"
	sharedRedis "sudooom.im.shared/redis"
	"sudooom.im.shared/snowflake"
)

// RoomService 房间服务
type RoomService struct {
	redisClient   *redis.Client
	sfNode        *snowflake.Node
	routerService *service.RouterService
	logger        *slog.Logger
}

// NewRoomService 创建房间服务
func NewRoomService(redisClient *redis.Client, sfNode *snowflake.Node, routerService *service.RouterService) *RoomService {
	return &RoomService{
		redisClient:   redisClient,
		sfNode:        sfNode,
		routerService: routerService,
		logger:        slog.Default(),
	}
}

// getUserInfo 从 Redis 获取用户基本信息
func (s *RoomService) getUserInfo(ctx context.Context, userId int64) *model.User {
	// 直接从固定的 user:info 键获取用户信息（与平台无关）
	userInfoKey := sharedRedis.BuildUserInfoKey(userId)
	data, err := s.redisClient.Get(ctx, userInfoKey).Result()
	if err != nil {
		// 如果无法获取用户信息，返回一个基本的 User 对象
		s.logger.Warn("Failed to get user info from Redis, using default", "userId", userId, "error", err)
		return &model.User{
			UserID:   userId,
			Username: fmt.Sprintf("user_%d", userId),
			Nickname: fmt.Sprintf("玩家%d", userId),
			Avatar:   "",
		}
	}

	// 解析用户信息
	var userInfo model.User
	if err := json.Unmarshal([]byte(data), &userInfo); err != nil {
		s.logger.Error("Failed to unmarshal user info", "error", err, "userId", userId)
		return &model.User{
			UserID:   userId,
			Username: fmt.Sprintf("user_%d", userId),
			Nickname: fmt.Sprintf("玩家%d", userId),
			Avatar:   "",
		}
	}

	return &userInfo
}

// GetRouterService 获取 RouterService 实例
func (s *RoomService) GetRouterService() *service.RouterService {
	return s.routerService
}

// GetRoom 从 Redis 获取房间信息
func (s *RoomService) GetRoom(ctx context.Context, roomId string) (*model.Room, error) {
	roomKey := sharedRedis.BuildRoomKey(roomId)
	data, err := s.redisClient.Get(ctx, roomKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("room not found")
		}
		return nil, fmt.Errorf("failed to get room: %w", err)
	}

	var room model.Room
	if err := json.Unmarshal([]byte(data), &room); err != nil {
		return nil, fmt.Errorf("failed to unmarshal room: %w", err)
	}

	return &room, nil
}

// UpdateRoom 更新房间信息到 Redis
func (s *RoomService) UpdateRoom(ctx context.Context, room *model.Room) error {
	roomKey := sharedRedis.BuildRoomKey(room.RoomID)
	room.UpdatedAt = time.Now()

	roomData, err := json.Marshal(room)
	if err != nil {
		return fmt.Errorf("failed to marshal room: %w", err)
	}

	// 保持原有的 TTL
	ttl, _ := s.redisClient.TTL(ctx, roomKey).Result()
	if ttl <= 0 {
		ttl = 48 * time.Hour
	}

	if err := s.redisClient.Set(ctx, roomKey, roomData, ttl).Err(); err != nil {
		return fmt.Errorf("failed to update room: %w", err)
	}

	return nil
}

// AddPlayerToRoom 将玩家加入房间
func (s *RoomService) AddPlayerToRoom(ctx context.Context, room *model.Room, userId int64, seatIndex int32) error {
	// 获取用户信息
	userInfo := s.getUserInfo(ctx, userId)

	// 创建玩家对象
	player := model.RoomPlayer{
		UserID:    userId,
		SeatIndex: seatIndex,
		IsReady:   false,
		IsHost:    false,
		UserInfo:  userInfo,
	}

	// 添加到玩家列表
	room.Players = append(room.Players, player)

	// 更新到 Redis
	if err := s.UpdateRoom(ctx, room); err != nil {
		return err
	}

	// 将用户加入房间成员列表
	userRoomKey := sharedRedis.BuildUserRoomKey(userId)
	if err := s.redisClient.Set(ctx, userRoomKey, room.RoomID, 24*time.Hour).Err(); err != nil {
		s.logger.Warn("Failed to save user room mapping", "error", err, "userId", userId)
	}

	// 将用户加入到房间用户列表
	roomUsersKey := sharedRedis.BuildRoomUsersKey(room.RoomID)
	if err := s.redisClient.SAdd(ctx, roomUsersKey, userId).Err(); err != nil {
		s.logger.Warn("Failed to add user to room users list", "error", err, "roomId", room.RoomID, "userId", userId)
	}
	s.redisClient.Expire(ctx, roomUsersKey, 48*time.Hour)

	return nil
}

// BroadcastToRoom 向房间所有成员广播消息
func (s *RoomService) BroadcastToRoom(ctx context.Context, roomId string, event string, roomInfo []byte) error {
	// 获取房间所有用户
	roomUsersKey := sharedRedis.BuildRoomUsersKey(roomId)
	userIdStrs, err := s.redisClient.SMembers(ctx, roomUsersKey).Result()
	if err != nil {
		s.logger.Warn("Failed to get room users", "error", err, "roomId", roomId)
		return err
	}

	// 转换为 int64 数组
	userIds := make([]int64, 0, len(userIdStrs))
	for _, userIdStr := range userIdStrs {
		var userId int64
		if _, err := fmt.Sscanf(userIdStr, "%d", &userId); err != nil {
			s.logger.Warn("Invalid user id in room users", "userIdStr", userIdStr)
			continue
		}
		userIds = append(userIds, userId)
	}

	// 使用新的广播API（会并发获取所有用户的locations并推送）
	return s.routerService.SendRoomPushToUsers(ctx, userIds, event, roomId, roomInfo)
}

// CheckUserInRoom 检查用户是否在房间用户列表中（使用 Redis）
func (s *RoomService) CheckUserInRoom(ctx context.Context, roomId string, userId int64) (bool, error) {
	roomUsersKey := sharedRedis.BuildRoomUsersKey(roomId)
	isInRoom, err := s.redisClient.SIsMember(ctx, roomUsersKey, userId).Result()
	if err != nil {
		s.logger.Error("Failed to check if user is in room", "error", err, "roomId", roomId, "userId", userId)
		return false, err
	}
	return isInRoom, nil
}

// FindPlayerInRoom 在房间 Players 列表中查找用户
// 返回：玩家指针、玩家索引（-1 表示未找到）
func (s *RoomService) FindPlayerInRoom(room *model.Room, userId int64) (*model.RoomPlayer, int) {
	for i, player := range room.Players {
		if player.UserID == userId {
			return &room.Players[i], i
		}
	}
	return nil, -1
}
