package room

import (
	"context"
	"encoding/json"
	"time"

	"sudooom.im.shared/model"
	sharedRedis "sudooom.im.shared/redis"
)

// ChangeSeatParams 换座位参数
type ChangeSeatParams struct {
	UserId     int64
	RoomId     string
	TargetSeat int32 // 目标座位索引
}

// ChangeSeat 换座位
func (s *RoomService) ChangeSeat(ctx context.Context, params ChangeSeatParams) error {
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

	// 4. 检查房间状态（只有等待状态的房间才能换座位）
	if room.Status != "waiting" {
		s.logger.Warn("Cannot change seat in non-waiting room", "roomId", params.RoomId, "status", room.Status)
		return ErrGameStarted
	}

	// 5. 检查目标座位是否被占用（-1 表示放弃座位，不需要检查）
	if params.TargetSeat != -1 && s.isSeatOccupied(room, params.TargetSeat) {
		s.logger.Warn("Target seat is occupied",
			"roomId", params.RoomId,
			"targetSeat", params.TargetSeat)
		return ErrSeatOccupied
	}

	// 6. 查找用户是否已有座位
	player, playerIndex := s.FindPlayerInRoom(room, params.UserId)

	// 7. 根据目标座位和当前状态执行不同操作
	if params.TargetSeat == -1 {
		// 用户想放弃座位成为观战者
		if player != nil && playerIndex != -1 {
			// 用户当前有座位，从 Players 列表中移除
			s.logger.Info("User giving up seat to become spectator",
				"userId", params.UserId,
				"roomId", params.RoomId,
				"oldSeat", room.Players[playerIndex].SeatIndex)

			// 从 Players 列表中移除
			room.Players = append(room.Players[:playerIndex], room.Players[playerIndex+1:]...)
		} else {
			// 用户已经是观战者，无需处理
			s.logger.Info("User is already a spectator",
				"userId", params.UserId,
				"roomId", params.RoomId)
			return nil
		}
	} else {
		// 用户想要坐到指定座位
		if player != nil && playerIndex != -1 {
			// 用户已有座位，更新座位索引
			s.logger.Info("User changing seat",
				"userId", params.UserId,
				"roomId", params.RoomId,
				"oldSeat", room.Players[playerIndex].SeatIndex,
				"newSeat", params.TargetSeat)

			room.Players[playerIndex].SeatIndex = params.TargetSeat
		} else {
			// 用户没有座位（观战者），加入 Players 列表
			s.logger.Info("User taking seat from spectator",
				"userId", params.UserId,
				"roomId", params.RoomId,
				"seat", params.TargetSeat)

			// 获取用户信息
			userInfo := s.getUserInfo(ctx, params.UserId)

			// 创建玩家对象
			newPlayer := model.RoomPlayer{
				UserID:    params.UserId,
				SeatIndex: params.TargetSeat,
				IsReady:   false,
				IsHost:    false,
				UserInfo:  userInfo,
			}

			// 添加到玩家列表
			room.Players = append(room.Players, newPlayer)
		}
	}

	// 8. 更新房间信息到 Redis
	if err := s.UpdateRoom(ctx, room); err != nil {
		s.logger.Error("Failed to update room after seat change", "error", err, "roomId", params.RoomId)
		return err
	}

	// 9. 获取更新后的房间信息并广播
	updatedRoom, _ := s.GetRoom(ctx, params.RoomId)
	if updatedRoom != nil {
		roomInfo, _ := json.Marshal(updatedRoom)
		if err := s.BroadcastToRoom(ctx, params.RoomId, "SEAT_CHANGED", roomInfo); err != nil {
			s.logger.Warn("Failed to broadcast seat change", "error", err, "roomId", params.RoomId)
		}
	}

	return nil
}
