package room

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
	"sudooom.im.logic/internal/service"
	"sudooom.im.shared/model"
	sharedRedis "sudooom.im.shared/redis"
	"sudooom.im.shared/snowflake"
)

// RoomService 房间服务
// 简化为使用 RoomManager，不再直接操作 Redis
type RoomService struct {
	roomManager   *RoomManager
	redisClient   *redis.Client // 仅用于获取用户信息
	sfNode        *snowflake.Node
	routerService *service.RouterService
	logger        *slog.Logger
}

// NewRoomService 创建房间服务
func NewRoomService(
	roomManager *RoomManager,
	redisClient *redis.Client,
	sfNode *snowflake.Node,
	routerService *service.RouterService,
) *RoomService {
	return &RoomService{
		roomManager:   roomManager,
		redisClient:   redisClient,
		sfNode:        sfNode,
		routerService: routerService,
		logger:        slog.Default(),
	}
}

// getUserInfo 从 Redis 获取用户基本信息
func (s *RoomService) getUserInfo(ctx context.Context, userId int64) *model.User {
	userInfoKey := sharedRedis.BuildUserInfoKey(userId)
	data, err := s.redisClient.Get(ctx, userInfoKey).Result()
	if err != nil {
		s.logger.Warn("Failed to get user info from Redis, using default", "userId", userId, "error", err)
		return &model.User{
			UserID:   userId,
			Username: fmt.Sprintf("user_%d", userId),
			Nickname: fmt.Sprintf("玩家%d", userId),
			Avatar:   "",
		}
	}

	var user model.User
	if err := json.Unmarshal([]byte(data), &user); err != nil {
		s.logger.Warn("Failed to unmarshal user info", "userId", userId, "error", err)
		return &model.User{
			UserID:   userId,
			Username: fmt.Sprintf("user_%d", userId),
			Nickname: fmt.Sprintf("玩家%d", userId),
			Avatar:   "",
		}
	}

	return &user
}

// BroadcastToRoom 广播消息给房间所有用户
func (s *RoomService) BroadcastToRoom(ctx context.Context, roomId string, event string, data interface{}) error {
	// 获取房间快照
	r, ok := s.roomManager.Get(roomId)
	if !ok {
		return ErrRoomNotFound
	}

	snapshot := r.GetSnapshot()

	// 提取用户 ID 列表
	userIds := make([]int64, 0, len(snapshot.Players))
	for _, player := range snapshot.Players {
		userIds = append(userIds, player.UserID)
	}

	// 序列化数据
	eventData, err := json.Marshal(data)
	if err != nil {
		s.logger.Error("Failed to marshal event data", "error", err, "event", event)
		return err
	}

	// 通过 RouterService 广播
	if err := s.routerService.SendRoomPushToUsers(ctx, userIds, event, roomId, eventData); err != nil {
		s.logger.Warn("Failed to broadcast to room", "error", err, "roomId", roomId, "event", event)
		return err
	}

	return nil
}

// GetRoom 获取房间信息
func (s *RoomService) GetRoom(ctx context.Context, roomId string) (*model.Room, error) {
	r, ok := s.roomManager.Get(roomId)
	if !ok {
		return nil, ErrRoomNotFound
	}

	return r.GetSnapshot(), nil
}
