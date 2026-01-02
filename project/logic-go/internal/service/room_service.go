package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"sudooom.im.shared/model"
	"sudooom.im.shared/proto"
	sharedRedis "sudooom.im.shared/redis"
	"sudooom.im.shared/snowflake"
)

// RoomService 房间服务
type RoomService struct {
	redisClient   *redis.Client
	sfNode        *snowflake.Node
	routerService *RouterService
	logger        *slog.Logger
}

// NewRoomService 创建房间服务
func NewRoomService(redisClient *redis.Client, sfNode *snowflake.Node, routerService *RouterService) *RoomService {
	return &RoomService{
		redisClient:   redisClient,
		sfNode:        sfNode,
		routerService: routerService,
		logger:        slog.Default(),
	}
}

// CreateRoom 创建房间
func (s *RoomService) CreateRoom(ctx context.Context, req *proto.RoomRequest, accessNodeId string) (*model.Room, error) {
	// 1. 解析房间配置
	var config model.RoomConfig
	if req.RoomConfig != "" {
		if err := json.Unmarshal([]byte(req.RoomConfig), &config); err != nil {
			s.logger.Error("Failed to parse room config", "error", err, "roomConfig", req.RoomConfig)
			return nil, fmt.Errorf("invalid room config: %w", err)
		}
	} else {
		// 使用默认配置
		config = model.RoomConfig{
			RoomName:   "默认房间",
			RoomType:   "public",
			MaxPlayers: 4,
		}
	}

	// 2. 生成雪花ID作为房间ID
	roomID := s.sfNode.Generate().String()

	// 3. 获取创建者的用户信息
	userInfo := s.getUserInfo(ctx, req.UserId)

	// 4. 创建房间对象
	now := time.Now()
	room := &model.Room{
		RoomID:       roomID,
		RoomName:     config.RoomName,
		RoomPassword: config.RoomPassword,
		RoomType:     config.RoomType,
		MaxPlayers:   config.MaxPlayers,
		GameType:     req.GameType,
		GameSettings: config.GameSettings,
		Extension:    config.Extension,
		CreatorID:    req.UserId,
		CreatedAt:    now,
		UpdatedAt:    now,
		Status:       "waiting", // 初始状态为等待
		Players: []model.RoomPlayer{
			{
				UserID:    req.UserId,
				SeatIndex: 0, // 创建者默认0号座位
				IsReady:   false,
				IsHost:    true, // 创建者是房主
				UserInfo:  userInfo,
			},
		},
	}

	// 5. 存储到 Redis
	roomKey := fmt.Sprintf("room:%s", roomID)
	roomData, err := json.Marshal(room)
	if err != nil {
		s.logger.Error("Failed to marshal room data", "error", err)
		return nil, fmt.Errorf("failed to marshal room: %w", err)
	}

	// 设置过期时间为 24 小时
	if err := s.redisClient.Set(ctx, roomKey, roomData, 24*time.Hour).Err(); err != nil {
		s.logger.Error("Failed to save room to Redis", "error", err, "roomId", roomID)
		return nil, fmt.Errorf("failed to save room: %w", err)
	}

	// 6. 将用户加入房间成员列表（用于快速查询用户所在房间）
	userRoomKey := fmt.Sprintf("user_room:%d", req.UserId)
	if err := s.redisClient.Set(ctx, userRoomKey, roomID, 24*time.Hour).Err(); err != nil {
		s.logger.Warn("Failed to save user room mapping", "error", err, "userId", req.UserId)
	}

	s.logger.Info("Room created successfully",
		"roomId", roomID,
		"roomName", room.RoomName,
		"creatorId", req.UserId,
		"gameType", req.GameType)

	return room, nil
}

// SendRoomCreatedResponse 发送房间创建成功响应
func (s *RoomService) SendRoomCreatedResponse(ctx context.Context, room *model.Room, accessNodeId string) error {
	// 将 Room 转换为 JSON
	roomInfo, err := json.Marshal(room)
	if err != nil {
		return fmt.Errorf("failed to marshal room info: %w", err)
	}

	// 发送 RoomPush 到 Access
	return s.routerService.SendRoomPushToUser(ctx, accessNodeId, room.CreatorID, "ROOM_CREATED", room.RoomID, roomInfo)
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
