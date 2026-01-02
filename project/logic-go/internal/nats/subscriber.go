package nats

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/nats-io/nats.go"
	sharedNats "sudooom.im.shared/nats"
	"sudooom.im.shared/proto"
)

// MessageHandler 消息处理器接口
type MessageHandler interface {
	HandleUserMessage(ctx context.Context, msg *proto.UserMessage, accessNodeId string, connId int64, platform string)
	HandleUserOnline(ctx context.Context, event *proto.UserOnline, accessNodeId string)
	HandleUserOffline(ctx context.Context, event *proto.UserOffline, accessNodeId string)
	HandleConversationRead(ctx context.Context, event *proto.ConversationRead)
	HandleRoomRequest(ctx context.Context, req *proto.RoomRequest, accessNodeId string)
	HandleGameRequest(ctx context.Context, req *proto.GameRequest, accessNodeId string)
}

// SubscriberConfig Worker Pool 配置
type SubscriberConfig struct {
	WorkerCount int // Worker 数量
	BufferSize  int // 消息缓冲区大小
}

// MessageSubscriber 消息订阅器
type MessageSubscriber struct {
	nc           *nats.Conn
	handler      MessageHandler
	logger       *slog.Logger
	subscription *nats.Subscription
	config       SubscriberConfig
	msgChan      chan *nats.Msg
	wg           sync.WaitGroup
	cancelFunc   context.CancelFunc
}

// NewMessageSubscriber 创建消息订阅器
func NewMessageSubscriber(nc *nats.Conn, handler MessageHandler, config SubscriberConfig) *MessageSubscriber {
	// 设置默认值
	if config.WorkerCount <= 0 {
		config.WorkerCount = 100
	}
	if config.BufferSize <= 0 {
		config.BufferSize = 10000
	}

	return &MessageSubscriber{
		nc:      nc,
		handler: handler,
		logger:  slog.Default(),
		config:  config,
	}
}

// Start 启动订阅
func (s *MessageSubscriber) Start(ctx context.Context) error {
	// 创建带缓冲的消息通道
	s.msgChan = make(chan *nats.Msg, s.config.BufferSize)

	// 创建可取消的上下文
	workerCtx, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel

	// 启动 Worker Pool
	for i := 0; i < s.config.WorkerCount; i++ {
		s.wg.Add(1)
		go s.worker(workerCtx)
	}

	// 订阅上行消息 - 使用队列组实现负载均衡
	sub, err := s.nc.QueueSubscribe(sharedNats.SubjectLogicUpstream, sharedNats.QueueGroupLogic, func(msg *nats.Msg) {
		select {
		case s.msgChan <- msg:
			// 消息入队成功
		default:
			// 缓冲区满，记录警告
			s.logger.Warn("Message buffer full, dropping message", "bufferSize", s.config.BufferSize)
		}
	})
	if err != nil {
		cancel()
		return err
	}

	s.subscription = sub
	s.logger.Info("NATS subscriber started",
		"subject", sharedNats.SubjectLogicUpstream,
		"workerCount", s.config.WorkerCount,
		"bufferSize", s.config.BufferSize,
	)
	return nil
}

// worker 工作协程
func (s *MessageSubscriber) worker(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-s.msgChan:
			if !ok {
				return
			}
			s.handleUpstreamMessage(ctx, msg.Data)
		}
	}
}

// handleUpstreamMessage 处理上行消息
func (s *MessageSubscriber) handleUpstreamMessage(ctx context.Context, data []byte) {
	var message proto.UpstreamMessage
	s.logger.Info("Received message", "subject", sharedNats.SubjectLogicUpstream)
	if err := json.Unmarshal(data, &message); err != nil {
		s.logger.Error("Failed to unmarshal message", "error", err)
		return
	}

	accessNodeId := message.AccessNodeId
	platform := message.Platform

	switch {
	case message.Payload.UserMessage != nil:
		s.handler.HandleUserMessage(ctx, message.Payload.UserMessage, accessNodeId, message.ConnId, platform)
	case message.Payload.UserOnline != nil:
		s.handler.HandleUserOnline(ctx, message.Payload.UserOnline, accessNodeId)
	case message.Payload.UserOffline != nil:
		s.handler.HandleUserOffline(ctx, message.Payload.UserOffline, accessNodeId)
	case message.Payload.ConversationRead != nil:
		s.handler.HandleConversationRead(ctx, message.Payload.ConversationRead)
	case message.Payload.RoomRequest != nil:
		s.handler.HandleRoomRequest(ctx, message.Payload.RoomRequest, accessNodeId)
	case message.Payload.GameRequest != nil:
		s.handler.HandleGameRequest(ctx, message.Payload.GameRequest, accessNodeId)
	}
}

// Stop 停止订阅
func (s *MessageSubscriber) Stop() error {
	// 取消 worker 上下文
	if s.cancelFunc != nil {
		s.cancelFunc()
	}

	// 取消订阅
	if s.subscription != nil {
		if err := s.subscription.Unsubscribe(); err != nil {
			s.logger.Error("Failed to unsubscribe", "error", err)
		}
	}

	// 关闭消息通道
	if s.msgChan != nil {
		close(s.msgChan)
	}

	// 等待所有 worker 完成
	s.wg.Wait()

	s.logger.Info("NATS subscriber stopped")
	return nil
}

// GetBufferUsage 获取缓冲区使用情况（用于监控）
func (s *MessageSubscriber) GetBufferUsage() (current int, capacity int) {
	if s.msgChan == nil {
		return 0, 0
	}
	return len(s.msgChan), cap(s.msgChan)
}
