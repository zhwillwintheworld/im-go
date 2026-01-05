package room

import (
	"context"
	"encoding/json"
	"time"

	"sudooom.im.shared/model"
	sharedRedis "sudooom.im.shared/redis"
)

// LeaveRoomParams 离开房间参数
type LeaveRoomParams struct {
	UserId       int64
	RoomId       string
	AccessNodeId string
	ConnId       int64
	Platform     string
}

// LeaveRoom 离开房间
func (s *RoomService) LeaveRoom(ctx context.Context, params LeaveRoomParams) error {
	// 1. 使用分布式锁保护房间操作
	lockKey := sharedRedis.BuildRoomLockKey(params.RoomId)
	locked, err := s.redisClient.SetNX(ctx, lockKey, "1", 5*time.Second).Result()
	if err != nil {
		s.logger.Error("Failed to acquire room lock", "error", err, "roomId", params.RoomId)
		return ErrLockFailed
	}
	if !locked {
		s.logger.Warn("Room is locked by another operation", "roomId", params.RoomId)
		return ErrRoomBusy
	}
	defer s.redisClient.Del(ctx, lockKey)

	// 2. 检查用户是否在房间用户列表中（使用 Redis）
	roomUsersKey := sharedRedis.BuildRoomUsersKey(params.RoomId)
	isInRoom, err := s.redisClient.SIsMember(ctx, roomUsersKey, params.UserId).Result()
	if err != nil {
		s.logger.Error("Failed to check if user is in room", "error", err, "roomId", params.RoomId, "userId", params.UserId)
		return err
	}

	if !isInRoom {
		s.logger.Warn("User not in room users list", "userId", params.UserId, "roomId", params.RoomId)
		// 清理可能存在的残留映射关系
		s.cleanUserRoomMapping(ctx, params.UserId, params.RoomId)
		return ErrNotInRoom
	}

	// 3. 获取房间信息
	room, err := s.GetRoom(ctx, params.RoomId)
	if err != nil {
		s.logger.Warn("Room not found", "roomId", params.RoomId, "error", err)
		// 房间不存在，清理用户映射关系
		s.cleanUserRoomMapping(ctx, params.UserId, params.RoomId)
		return ErrRoomNotFound
	}

	// 4. 根据房间状态执行不同的离开逻辑
	if room.Status == "waiting" {
		return s.leaveWaitingRoom(ctx, room, params)
	} else {
		return s.leavePlayingRoom(ctx, room, params)
	}
}

// leaveWaitingRoom 处理等待状态房间的离开逻辑
func (s *RoomService) leaveWaitingRoom(ctx context.Context, room *model.Room, params LeaveRoomParams) error {
	// 查找用户是否在玩家列表中（有座位）
	playerIndex := -1
	var leavingPlayer *model.RoomPlayer
	for i, player := range room.Players {
		if player.UserID == params.UserId {
			playerIndex = i
			leavingPlayer = &player
			break
		}
	}

	// 清理用户映射关系
	s.cleanUserRoomMapping(ctx, params.UserId, params.RoomId)

	// 如果用户没有座位（例如观战者），只需要清理映射关系即可
	if playerIndex == -1 {
		s.logger.Info("User without seat left waiting room",
			"roomId", params.RoomId,
			"userId", params.UserId)

		// 向房间所有人广播观战者离开消息（简洁消息体）
		leaveInfo := map[string]interface{}{
			"room_id": params.RoomId,
			"user_id": params.UserId,
			"event":   "spectator_left",
		}
		leaveData, _ := json.Marshal(leaveInfo)
		if err := s.BroadcastToRoom(ctx, params.RoomId, "USER_LEFT", leaveData); err != nil {
			s.logger.Warn("Failed to broadcast spectator leave event", "error", err, "roomId", params.RoomId)
		}

		return nil
	}

	// 用户有座位，需要释放座位
	isHost := leavingPlayer.IsHost
	remainingPlayers := len(room.Players) - 1

	// 从房间玩家列表中移除该玩家
	room.Players = append(room.Players[:playerIndex], room.Players[playerIndex+1:]...)

	// 如果房间没人了，删除房间
	if remainingPlayers == 0 {
		s.logger.Info("Last player left, deleting room", "roomId", params.RoomId)
		return s.deleteRoom(ctx, params.RoomId, params)
	}

	// 房间还有人，如果离开的是房主，需要转移房主
	if isHost {
		// 转移房主给第一个玩家
		room.Players[0].IsHost = true
		s.logger.Info("Host transferred",
			"roomId", params.RoomId,
			"oldHost", params.UserId,
			"newHost", room.Players[0].UserID)
	}

	// 更新房间信息
	if err := s.UpdateRoom(ctx, room); err != nil {
		s.logger.Error("Failed to update room after player left", "error", err, "roomId", params.RoomId)
		return err
	}

	// 获取更新后的房间信息
	updatedRoom, _ := s.GetRoom(ctx, params.RoomId)
	if updatedRoom != nil {
		// 向房间所有人广播离开消息
		roomInfo, _ := json.Marshal(updatedRoom)
		if err := s.BroadcastToRoom(ctx, params.RoomId, "USER_LEFT", roomInfo); err != nil {
			s.logger.Warn("Failed to broadcast leave event", "error", err, "roomId", params.RoomId)
		}
	}

	s.logger.Info("User left waiting room",
		"roomId", params.RoomId,
		"userId", params.UserId,
		"hadSeat", true,
		"remainingPlayers", remainingPlayers)

	return nil
}

// leavePlayingRoom 处理游戏中状态房间的离开逻辑
func (s *RoomService) leavePlayingRoom(ctx context.Context, room *model.Room, params LeaveRoomParams) error {
	// 只从房间用户列表中移除（不修改 room.Players，保持游戏状态）
	s.cleanUserRoomMapping(ctx, params.UserId, params.RoomId)

	// 向离开的用户发送离开确认消息
	senderLoc := model.UserLocation{
		AccessNodeId: params.AccessNodeId,
		ConnId:       params.ConnId,
		Platform:     params.Platform,
		UserId:       params.UserId,
	}

	leaveInfo := map[string]interface{}{
		"room_id": params.RoomId,
		"user_id": params.UserId,
		"message": "你已离开游戏中的房间",
	}
	leaveData, _ := json.Marshal(leaveInfo)

	if err := s.routerService.SendRoomPushToSelf(senderLoc, "LEFT_PLAYING_ROOM", params.RoomId, leaveData); err != nil {
		s.logger.Warn("Failed to send leave confirmation", "error", err, "userId", params.UserId)
	}

	s.logger.Info("User left playing room",
		"roomId", params.RoomId,
		"userId", params.UserId,
		"status", room.Status)

	return nil
}

// deleteRoom 删除房间
func (s *RoomService) deleteRoom(ctx context.Context, roomId string, params LeaveRoomParams) error {
	// 删除房间信息
	roomKey := sharedRedis.BuildRoomKey(roomId)
	if err := s.redisClient.Del(ctx, roomKey).Err(); err != nil {
		s.logger.Warn("Failed to delete room", "error", err, "roomId", roomId)
	}

	// 删除房间用户列表
	roomUsersKey := sharedRedis.BuildRoomUsersKey(roomId)
	if err := s.redisClient.Del(ctx, roomUsersKey).Err(); err != nil {
		s.logger.Warn("Failed to delete room users list", "error", err, "roomId", roomId)
	}

	// 向离开的用户发送房间已解散消息
	senderLoc := model.UserLocation{
		AccessNodeId: params.AccessNodeId,
		ConnId:       params.ConnId,
		Platform:     params.Platform,
		UserId:       params.UserId,
	}

	dismissInfo := map[string]interface{}{
		"room_id": roomId,
		"message": "房间已解散",
	}
	dismissData, _ := json.Marshal(dismissInfo)

	if err := s.routerService.SendRoomPushToSelf(senderLoc, "ROOM_DISMISSED", roomId, dismissData); err != nil {
		s.logger.Warn("Failed to send room dismissed message", "error", err, "userId", params.UserId)
	}

	s.logger.Info("Room deleted", "roomId", roomId)
	return nil
}

// cleanUserRoomMapping 清理用户房间映射关系
func (s *RoomService) cleanUserRoomMapping(ctx context.Context, userId int64, roomId string) {
	// 删除用户所在房间映射
	userRoomKey := sharedRedis.BuildUserRoomKey(userId)
	if err := s.redisClient.Del(ctx, userRoomKey).Err(); err != nil {
		s.logger.Warn("Failed to delete user room mapping", "error", err, "userId", userId)
	}

	// 从房间用户列表中移除
	roomUsersKey := sharedRedis.BuildRoomUsersKey(roomId)
	if err := s.redisClient.SRem(ctx, roomUsersKey, userId).Err(); err != nil {
		s.logger.Warn("Failed to remove user from room users list", "error", err, "roomId", roomId, "userId", userId)
	}
}
