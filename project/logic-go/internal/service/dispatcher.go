package service

import (
	"log/slog"
	"time"

	"sudooom.im.logic/internal/nats"
	sharedModel "sudooom.im.shared/model"
	"sudooom.im.shared/proto"
)

// DispatcherService 消息分发服务
type DispatcherService struct {
	publisher *nats.MessagePublisher
	logger    *slog.Logger
}

// NewDispatcherService 创建消息分发服务
func NewDispatcherService(publisher *nats.MessagePublisher) *DispatcherService {
	return &DispatcherService{
		publisher: publisher,
		logger:    slog.Default(),
	}
}

// buildDownstreamMessage 构建下行消息（辅助方法，减少重复代码）
func (s *DispatcherService) buildDownstreamMessage(userId int64, connId int64, platform string, payload proto.DownstreamPayload) *proto.DownstreamMessage {
	return &proto.DownstreamMessage{
		UserId:   userId,
		ConnId:   connId,
		Platform: platform,
		Payload:  payload,
	}
}

// DispatchPushMessage 分发推送消息到用户
func (s *DispatcherService) DispatchPushMessage(userId int64, locations []sharedModel.UserLocation, msg *proto.UserMessage, serverMsgId int64) error {
	for _, loc := range locations {
		downstreamMsg := s.buildDownstreamMessage(userId, loc.ConnId, loc.Platform, proto.DownstreamPayload{
			PushMessage: &proto.PushMessage{
				ServerMsgId: serverMsgId,
				FromUserId:  msg.FromUserId,
				ToUserId:    msg.ToUserId,
				ToGroupId:   msg.ToGroupId,
				MsgType:     msg.MsgType,
				Content:     msg.Content,
				Timestamp:   time.Now().UnixMilli(),
			},
		})
		if err := s.publisher.PublishToAccess(loc.AccessNodeId, downstreamMsg); err != nil {
			s.logger.Warn("Failed to dispatch push message",
				"userId", userId,
				"platform", loc.Platform,
				"accessNodeId", loc.AccessNodeId,
				"error", err)
			// 继续推送到其他设备，不中断
		}
	}
	return nil
}

// DispatchAck 分发 ACK 消息
func (s *DispatcherService) DispatchAck(userId int64, locations []sharedModel.UserLocation, clientMsgId string, serverMsgId int64) error {
	for _, loc := range locations {
		ackMsg := s.buildDownstreamMessage(userId, loc.ConnId, loc.Platform, proto.DownstreamPayload{
			MessageAck: &proto.MessageAck{
				ClientMsgId: clientMsgId,
				ServerMsgId: serverMsgId,
				ToUserId:    userId,
				Timestamp:   time.Now().UnixMilli(),
			},
		})
		if err := s.publisher.PublishToAccess(loc.AccessNodeId, ackMsg); err != nil {
			s.logger.Warn("Failed to dispatch ack",
				"userId", userId,
				"accessNodeId", loc.AccessNodeId,
				"error", err)
		}
	}
	return nil
}

// DispatchAckDirect 直接分发 ACK 到指定 Access 节点（使用 connId）
func (s *DispatcherService) DispatchAckDirect(accessNodeId string, connId int64, userId int64, clientMsgId string, serverMsgId int64) error {
	ackMsg := &proto.DownstreamMessage{
		UserId: userId,
		ConnId: connId,
		Payload: proto.DownstreamPayload{
			MessageAck: &proto.MessageAck{
				ClientMsgId: clientMsgId,
				ServerMsgId: serverMsgId,
				ToUserId:    userId,
				Timestamp:   time.Now().UnixMilli(),
			},
		},
	}
	if err := s.publisher.PublishToAccess(accessNodeId, ackMsg); err != nil {
		s.logger.Warn("Failed to dispatch ack direct",
			"userId", userId,
			"accessNodeId", accessNodeId,
			"error", err)
		return err
	}
	return nil
}

// DispatchRoomPushDirect 直接分发房间推送（使用 connId）
func (s *DispatcherService) DispatchRoomPushDirect(accessNodeId string, connId int64, userId int64, event string, roomId string, roomInfo []byte) error {
	roomPushMsg := s.buildDownstreamMessage(userId, connId, "", proto.DownstreamPayload{
		RoomPush: &proto.RoomPush{
			Event:    event,
			RoomId:   roomId,
			UserId:   userId,
			RoomInfo: roomInfo,
			ToUserId: userId,
		},
	})
	return s.publisher.PublishToAccess(accessNodeId, roomPushMsg)
}

// DispatchRoomPushToLocations 分发房间推送到指定 locations
func (s *DispatcherService) DispatchRoomPushToLocations(userId int64, locations []sharedModel.UserLocation, event string, roomId string, roomInfo []byte) error {
	for _, loc := range locations {
		roomPushMsg := s.buildDownstreamMessage(userId, loc.ConnId, loc.Platform, proto.DownstreamPayload{
			RoomPush: &proto.RoomPush{
				Event:    event,
				RoomId:   roomId,
				UserId:   userId,
				RoomInfo: roomInfo,
				ToUserId: userId,
			},
		})
		if err := s.publisher.PublishToAccess(loc.AccessNodeId, roomPushMsg); err != nil {
			s.logger.Warn("Failed to dispatch room push",
				"userId", userId,
				"platform", loc.Platform,
				"accessNodeId", loc.AccessNodeId,
				"error", err)
			// 继续推送到其他设备
		}
	}
	return nil
}

// DispatchGamePushDirect 直接分发游戏推送（使用 connId）
func (s *DispatcherService) DispatchGamePushDirect(accessNodeId string, connId int64, userId int64, roomId string, gameType string, gamePayload []byte) error {
	gamePushMsg := s.buildDownstreamMessage(userId, connId, "", proto.DownstreamPayload{
		GamePush: &proto.GamePush{
			RoomId:      roomId,
			GameType:    gameType,
			GamePayload: gamePayload,
			ToUserId:    userId,
		},
	})
	return s.publisher.PublishToAccess(accessNodeId, gamePushMsg)
}

// DispatchGamePushToLocations 分发游戏推送到指定 locations
func (s *DispatcherService) DispatchGamePushToLocations(userId int64, locations []sharedModel.UserLocation, roomId string, gameType string, gamePayload []byte) error {
	for _, loc := range locations {
		gamePushMsg := s.buildDownstreamMessage(userId, loc.ConnId, loc.Platform, proto.DownstreamPayload{
			GamePush: &proto.GamePush{
				RoomId:      roomId,
				GameType:    gameType,
				GamePayload: gamePayload,
				ToUserId:    userId,
			},
		})
		if err := s.publisher.PublishToAccess(loc.AccessNodeId, gamePushMsg); err != nil {
			s.logger.Warn("Failed to dispatch game push",
				"userId", userId,
				"platform", loc.Platform,
				"accessNodeId", loc.AccessNodeId,
				"error", err)
			// 继续推送到其他设备
		}
	}
	return nil
}
