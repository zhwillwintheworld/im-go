package room

import (
	"context"
	"encoding/json"
	"time"

	"sudooom.im.shared/model"
	sharedRedis "sudooom.im.shared/redis"
)

// JoinRoomParams 加入房间参数
type JoinRoomParams struct {
	UserId       int64
	RoomId       string
	Password     string // 房间密码（私密房间需要）
	SeatIndex    int32  // 期望的座位索引（-1表示不指定）
	AccessNodeId string
	ConnId       int64
	Platform     string
}

// JoinRoom 加入房间
func (s *RoomService) JoinRoom(ctx context.Context, params JoinRoomParams) (*model.Room, error) {
	// 1. 使用分布式锁保护房间操作
	lockKey := sharedRedis.BuildRoomLockKey(params.RoomId)
	locked, err := s.redisClient.SetNX(ctx, lockKey, "1", 5*time.Second).Result()
	if err != nil {
		s.logger.Error("Failed to acquire room lock", "error", err, "roomId", params.RoomId)
		return nil, ErrLockFailed
	}
	if !locked {
		s.logger.Warn("Room is locked by another operation", "roomId", params.RoomId)
		return nil, ErrRoomBusy
	}
	defer s.redisClient.Del(ctx, lockKey)

	// 2. 获取房间信息
	room, err := s.GetRoom(ctx, params.RoomId)
	if err != nil {
		s.logger.Warn("Room not found", "roomId", params.RoomId, "error", err)
		return nil, ErrRoomNotFound
	}

	// 3. 校验加入条件
	if err := s.validateJoinConditions(room, params); err != nil {
		return nil, err
	}

	// 4. 判断是新加入还是重新加入
	isRejoin := false
	if room.Status != "waiting" {
		// 游戏中状态，这是重新加入
		isRejoin = true
	}

	var seatIndex int32
	var eventName string

	if isRejoin {
		// 重新加入：只需要恢复用户映射关系，不修改 Players 列表
		s.logger.Info("User rejoining room",
			"roomId", params.RoomId,
			"userId", params.UserId,
			"status", room.Status)

		// 恢复用户-房间映射
		userRoomKey := sharedRedis.BuildUserRoomKey(params.UserId)
		if err := s.redisClient.Set(ctx, userRoomKey, room.RoomID, 24*time.Hour).Err(); err != nil {
			s.logger.Warn("Failed to save user room mapping", "error", err, "userId", params.UserId)
		}

		// 将用户重新加入到房间用户列表
		roomUsersKey := sharedRedis.BuildRoomUsersKey(room.RoomID)
		if err := s.redisClient.SAdd(ctx, roomUsersKey, params.UserId).Err(); err != nil {
			s.logger.Warn("Failed to add user to room users list", "error", err, "roomId", room.RoomID, "userId", params.UserId)
		}
		s.redisClient.Expire(ctx, roomUsersKey, 24*time.Hour)

		eventName = "USER_REJOINED"
	} else {
		// 新加入：需要分配座位并添加到 Players 列表
		seatIndex = s.allocateSeat(room, params.SeatIndex)

		if err := s.AddPlayerToRoom(ctx, room, params.UserId, seatIndex); err != nil {
			s.logger.Error("Failed to add player to room",
				"error", err,
				"roomId", params.RoomId,
				"userId", params.UserId)
			return nil, err
		}

		s.logger.Info("User joined room successfully",
			"roomId", params.RoomId,
			"userId", params.UserId,
			"seatIndex", seatIndex)

		eventName = "USER_JOINED"
	}

	// 5. 获取更新后的房间信息
	updatedRoom, _ := s.GetRoom(ctx, params.RoomId)
	if updatedRoom != nil {
		// 6. 向房间所有人广播消息
		roomInfo, _ := json.Marshal(updatedRoom)
		if err := s.BroadcastToRoom(ctx, params.RoomId, eventName, roomInfo); err != nil {
			s.logger.Warn("Failed to broadcast event", "error", err, "roomId", params.RoomId, "event", eventName)
		}
	}

	return updatedRoom, nil
}

// validateJoinConditions 校验加入房间的条件
func (s *RoomService) validateJoinConditions(room *model.Room, params JoinRoomParams) error {
	// 1. 检查房间类型和密码
	if room.RoomType == "private" && room.RoomPassword != "" {
		if room.RoomPassword != params.Password {
			s.logger.Warn("Invalid room password",
				"roomId", room.RoomID,
				"userId", params.UserId)
			return ErrInvalidPassword
		}
	}

	// 2. 根据房间状态进行不同的校验
	if room.Status == "waiting" {
		// 等待状态：正常的加入逻辑
		return s.validateJoinWaitingRoom(room, params)
	} else {
		// 游戏中/已结束：只允许重新加入（曾经在房间但离开了）
		return s.validateRejoinPlayingRoom(room, params)
	}
}

// validateJoinWaitingRoom 校验加入等待中的房间
func (s *RoomService) validateJoinWaitingRoom(room *model.Room, params JoinRoomParams) error {
	// 1. 检查房间是否已满
	if len(room.Players) >= room.MaxPlayers {
		s.logger.Warn("Room is full",
			"roomId", room.RoomID,
			"maxPlayers", room.MaxPlayers,
			"currentPlayers", len(room.Players))
		return ErrRoomFull
	}

	// 2. 检查用户是否已在房间中
	for _, player := range room.Players {
		if player.UserID == params.UserId {
			s.logger.Warn("User already in room",
				"roomId", room.RoomID,
				"userId", params.UserId)
			return ErrAlreadyInRoom
		}
	}

	return nil
}

// validateRejoinPlayingRoom 校验重新加入游戏中的房间
func (s *RoomService) validateRejoinPlayingRoom(room *model.Room, params JoinRoomParams) error {
	// 检查用户是否在玩家列表中（只有在玩家列表中的用户才能重新加入）
	isPlayer := false
	for _, player := range room.Players {
		if player.UserID == params.UserId {
			isPlayer = true
			break
		}
	}

	if !isPlayer {
		s.logger.Warn("User is not a player in this game",
			"roomId", room.RoomID,
			"userId", params.UserId,
			"status", room.Status)
		return ErrGameStarted // 不是游戏玩家，不允许加入
	}

	// 是游戏玩家，允许重新加入（重新连接）
	s.logger.Info("Player rejoining game",
		"roomId", room.RoomID,
		"userId", params.UserId,
		"status", room.Status)

	return nil
}

// allocateSeat 分配座位
func (s *RoomService) allocateSeat(room *model.Room, requestedSeat int32) int32 {
	if room.GameType == "HT_MAHJONG" {
		// 麻将游戏：座位 0-3，如果请求的座位已占用或无效，则自动分配
		if requestedSeat < 0 || requestedSeat > 3 {
			// 未指定座位或指定无效，自动分配
			return s.findAvailableSeat(room, 0, 3)
		}

		// 检查请求的座位是否被占用
		if s.isSeatOccupied(room, requestedSeat) {
			// 座位被占用，自动分配
			return s.findAvailableSeat(room, 0, 3)
		}

		// 请求的座位可用
		return requestedSeat
	}

	// 其他游戏：查找可用座位，考虑用户退出导致的座位空缺
	maxSeat := int32(room.MaxPlayers - 1)
	if requestedSeat >= 0 && requestedSeat <= maxSeat && !s.isSeatOccupied(room, requestedSeat) {
		// 请求的座位有效且可用
		return requestedSeat
	}

	// 查找第一个可用座位
	return s.findAvailableSeat(room, 0, maxSeat)
}

// isSeatOccupied 检查座位是否被占用
func (s *RoomService) isSeatOccupied(room *model.Room, seatIndex int32) bool {
	for _, player := range room.Players {
		if player.SeatIndex == seatIndex {
			return true
		}
	}
	return false
}

// findAvailableSeat 查找可用的座位（在指定范围内）
// 返回 -1 表示指定范围内无可用座位
func (s *RoomService) findAvailableSeat(room *model.Room, minSeat, maxSeat int32) int32 {
	for i := minSeat; i <= maxSeat; i++ {
		if !s.isSeatOccupied(room, i) {
			return i
		}
	}

	// 如果 0-3 座位都满了，分配为观战者
	if minSeat == 0 && maxSeat == 3 {
		return int32(len(room.Players))
	}

	return -1
}
