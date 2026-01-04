package service

import (
	"log/slog"

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

// Dispatch 通用分发方法：将 payload 分发到指定的用户设备位置
// 这是 dispatcher 的核心底层功能，调用方负责构建 payload
func (s *DispatcherService) Dispatch(userId int64, locations []sharedModel.UserLocation, payload proto.DownstreamPayload) error {
	for _, loc := range locations {
		downstreamMsg := s.buildDownstreamMessage(userId, loc.ConnId, loc.Platform, payload)
		if err := s.publisher.PublishToAccess(loc.AccessNodeId, downstreamMsg); err != nil {
			s.logger.Warn("Failed to dispatch message",
				"userId", userId,
				"platform", loc.Platform,
				"accessNodeId", loc.AccessNodeId,
				"error", err)
			// 继续推送到其他设备，不中断
		}
	}
	return nil
}
