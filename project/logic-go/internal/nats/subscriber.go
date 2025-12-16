package nats

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/nats-io/nats.go"
	sharedNats "sudooom.im.shared/nats"
	"sudooom.im.shared/proto"
)

// MessageHandler 消息处理器接口
type MessageHandler interface {
	HandleUserMessage(ctx context.Context, msg *proto.UserMessage, accessNodeId string)
	HandleUserOnline(ctx context.Context, event *proto.UserOnline, accessNodeId string)
	HandleUserOffline(ctx context.Context, event *proto.UserOffline, accessNodeId string)
}

// MessageSubscriber 消息订阅器
type MessageSubscriber struct {
	nc           *nats.Conn
	handler      MessageHandler
	logger       *slog.Logger
	subscription *nats.Subscription
}

// NewMessageSubscriber 创建消息订阅器
func NewMessageSubscriber(nc *nats.Conn, handler MessageHandler) *MessageSubscriber {
	return &MessageSubscriber{
		nc:      nc,
		handler: handler,
		logger:  slog.Default(),
	}
}

// Start 启动订阅
func (s *MessageSubscriber) Start(ctx context.Context) error {
	// 订阅上行消息 - 使用队列组实现负载均衡
	sub, err := s.nc.QueueSubscribe(sharedNats.SubjectLogicUpstream, sharedNats.QueueGroupLogic, func(msg *nats.Msg) {
		go s.handleUpstreamMessage(ctx, msg.Data)
	})
	if err != nil {
		return err
	}

	s.subscription = sub
	s.logger.Info("NATS subscriber started", "subject", sharedNats.SubjectLogicUpstream)
	return nil
}

// handleUpstreamMessage 处理上行消息
func (s *MessageSubscriber) handleUpstreamMessage(ctx context.Context, data []byte) {
	var message proto.UpstreamMessage
	if err := json.Unmarshal(data, &message); err != nil {
		s.logger.Error("Failed to unmarshal message", "error", err)
		return
	}

	accessNodeId := message.AccessNodeId

	switch {
	case message.UserMessage != nil:
		s.handler.HandleUserMessage(ctx, message.UserMessage, accessNodeId)
	case message.UserOnline != nil:
		s.handler.HandleUserOnline(ctx, message.UserOnline, accessNodeId)
	case message.UserOffline != nil:
		s.handler.HandleUserOffline(ctx, message.UserOffline, accessNodeId)
	}
}

// Stop 停止订阅
func (s *MessageSubscriber) Stop() error {
	if s.subscription != nil {
		if err := s.subscription.Unsubscribe(); err != nil {
			return err
		}
	}
	s.logger.Info("NATS subscriber stopped")
	return nil
}
