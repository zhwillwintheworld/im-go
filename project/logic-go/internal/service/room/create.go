package room

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"sudooom.im.shared/model"
	"sudooom.im.shared/proto"
	sharedRedis "sudooom.im.shared/redis"
)

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
	roomKey := sharedRedis.BuildRoomKey(roomID)
	roomData, err := json.Marshal(room)
	if err != nil {
		s.logger.Error("Failed to marshal room data", "error", err)
		return nil, fmt.Errorf("failed to marshal room: %w", err)
	}

	// 设置过期时间为 48 小时
	if err := s.redisClient.Set(ctx, roomKey, roomData, 48*time.Hour).Err(); err != nil {
		s.logger.Error("Failed to save room to Redis", "error", err, "roomId", roomID)
		return nil, fmt.Errorf("failed to save room: %w", err)
	}

	// 6. 将用户加入房间成员列表（用于快速查询用户所在房间）
	userRoomKey := sharedRedis.BuildUserRoomKey(req.UserId)
	if err := s.redisClient.Set(ctx, userRoomKey, roomID, 24*time.Hour).Err(); err != nil {
		s.logger.Warn("Failed to save user room mapping", "error", err, "userId", req.UserId)
	}

	// 7. 将用户加入到房间用户列表（使用 Set 存储房间内的所有用户）
	roomUsersKey := sharedRedis.BuildRoomUsersKey(roomID)
	if err := s.redisClient.SAdd(ctx, roomUsersKey, req.UserId).Err(); err != nil {
		s.logger.Warn("Failed to add user to room users list", "error", err, "roomId", roomID, "userId", req.UserId)
	}
	// 设置过期时间与房间一致
	s.redisClient.Expire(ctx, roomUsersKey, 48*time.Hour)
	s.logger.Info("Room created successfully",
		"roomId", roomID,
		"roomName", room.RoomName,
		"creatorId", req.UserId,
		"gameType", req.GameType)

	return room, nil
}

// SendRoomCreatedResponse 发送房间创建成功响应
func (s *RoomService) SendRoomCreatedResponse(ctx context.Context, room *model.Room, accessNodeId string, connId int64, platform string) error {
	// 将 Room 转换为 JSON
	roomInfo, err := json.Marshal(room)
	if err != nil {
		return fmt.Errorf("failed to marshal room info: %w", err)
	}

	// 构造 sender location
	senderLoc := model.UserLocation{
		AccessNodeId: accessNodeId,
		ConnId:       connId,
		Platform:     platform,
		UserId:       room.CreatorID,
	}

	// 发送给自己（快速响应 + 多端同步）
	return s.routerService.SendRoomPushToSelf(senderLoc, "ROOM_CREATED", room.RoomID, roomInfo)
}
