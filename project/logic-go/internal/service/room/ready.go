package room

import (
	"context"
	"encoding/json"
	"time"

	sharedRedis "sudooom.im.shared/redis"
)

// ToggleReadyParams 切换准备状态参数
type ToggleReadyParams struct {
	UserId int64
	RoomId string
}

// ToggleReady 切换玩家准备状态
func (s *RoomService) ToggleReady(ctx context.Context, params ToggleReadyParams) error {
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

	// 2. 检查用户是否在房间中
	isInRoom, err := s.CheckUserInRoom(ctx, params.RoomId, params.UserId)
	if err != nil {
		return err
	}
	if !isInRoom {
		s.logger.Warn("User not in room", "userId", params.UserId, "roomId", params.RoomId)
		return ErrNotInRoom
	}

	// 3. 获取房间信息
	room, err := s.GetRoom(ctx, params.RoomId)
	if err != nil {
		s.logger.Warn("Room not found", "roomId", params.RoomId, "error", err)
		return ErrRoomNotFound
	}

	// 4. 检查房间状态（只有等待状态的房间才能准备）
	if room.Status != "waiting" {
		s.logger.Warn("Cannot ready in non-waiting room", "roomId", params.RoomId, "status", room.Status)
		return ErrGameStarted
	}

	// 5. 查找玩家（只有有座位的玩家才能准备）
	player, playerIndex := s.FindPlayerInRoom(room, params.UserId)
	if player == nil || playerIndex == -1 {
		s.logger.Warn("User has no seat in room", "userId", params.UserId, "roomId", params.RoomId)
		return ErrNotInRoom // 或者可以定义一个新的错误 ErrNoSeat
	}

	// 6. 切换准备状态
	oldReadyState := room.Players[playerIndex].IsReady
	room.Players[playerIndex].IsReady = !oldReadyState

	s.logger.Info("Player toggled ready state",
		"userId", params.UserId,
		"roomId", params.RoomId,
		"oldState", oldReadyState,
		"newState", room.Players[playerIndex].IsReady)

	// 7. 更新房间信息到 Redis
	if err := s.UpdateRoom(ctx, room); err != nil {
		s.logger.Error("Failed to update room after ready toggle", "error", err, "roomId", params.RoomId)
		return err
	}

	// 8. 获取更新后的房间信息并广播
	updatedRoom, _ := s.GetRoom(ctx, params.RoomId)
	if updatedRoom != nil {
		roomInfo, _ := json.Marshal(updatedRoom)
		if err := s.BroadcastToRoom(ctx, params.RoomId, "PLAYER_READY_CHANGED", roomInfo); err != nil {
			s.logger.Warn("Failed to broadcast ready state change", "error", err, "roomId", params.RoomId)
		}
	}

	return nil
}
