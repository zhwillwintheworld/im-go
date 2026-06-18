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
	HandleRoomRequest(ctx context.Context, req *proto.RoomRequest, accessNodeId string, connId int64, platform string)
	HandleGameRequest(ctx context.Context, req *proto.GameRequest, accessNodeId string, connId int64, platform string)
}

// SubscriberConfig Worker Pool 配置
type SubscriberConfig struct {
	WorkerCount     int // 普通消息 Worker 数量
	BufferSize      int // 普通消息缓冲区大小
	RoomWorkerCount int // room/game Worker 数量
	RoomBufferSize  int // room/game 消息缓冲区大小
	RoomShardCount  int // room/game shard 总数
	RoomShardIndex  int // 当前节点负责的 room/game shard
}

// MessageSubscriber 消息订阅器
type MessageSubscriber struct {
	nc                 *nats.Conn
	handler            MessageHandler
	logger             *slog.Logger
	subscription       *nats.Subscription
	shardSubscriptions []*nats.Subscription
	config             SubscriberConfig
	msgChan            chan *nats.Msg
	shardMsgChan       chan *nats.Msg
	wg                 sync.WaitGroup
	cancelFunc         context.CancelFunc
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
	if config.RoomWorkerCount <= 0 {
		config.RoomWorkerCount = 32
	}
	if config.RoomBufferSize <= 0 {
		config.RoomBufferSize = 5000
	}
	if config.RoomShardCount <= 0 {
		config.RoomShardCount = sharedNats.DefaultRoomShardCount
	}
	if config.RoomShardIndex < 0 || config.RoomShardIndex >= config.RoomShardCount {
		config.RoomShardIndex = 0
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
	s.shardMsgChan = make(chan *nats.Msg, s.config.RoomBufferSize)

	// 创建可取消的上下文
	workerCtx, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel

	// 启动 Worker Pool
	for i := 0; i < s.config.WorkerCount; i++ {
		s.wg.Add(1)
		go s.worker(workerCtx, s.msgChan)
	}

	for i := 0; i < s.config.RoomWorkerCount; i++ {
		s.wg.Add(1)
		go s.worker(workerCtx, s.shardMsgChan)
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

	if err := s.subscribeRoomGameShards(); err != nil {
		if unsubscribeErr := sub.Unsubscribe(); unsubscribeErr != nil {
			s.logger.Error("Failed to unsubscribe upstream after shard subscribe error", "error", unsubscribeErr)
		}
		cancel()
		return err
	}

	s.logger.Info("NATS subscriber started",
		"subject", sharedNats.SubjectLogicUpstream,
		"workerCount", s.config.WorkerCount,
		"bufferSize", s.config.BufferSize,
		"roomWorkerCount", s.config.RoomWorkerCount,
		"roomBufferSize", s.config.RoomBufferSize,
		"roomShardCount", s.config.RoomShardCount,
		"roomShardIndex", s.config.RoomShardIndex,
	)
	return nil
}

// worker 工作协程
func (s *MessageSubscriber) worker(ctx context.Context, msgChan <-chan *nats.Msg) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-msgChan:
			if !ok {
				return
			}
			s.handleUpstreamMessage(ctx, msg.Data)
		}
	}
}

func (s *MessageSubscriber) subscribeRoomGameShards() error {
	for _, subject := range s.roomGameShardSubjects() {
		sub, err := s.nc.Subscribe(subject, func(msg *nats.Msg) {
			select {
			case s.shardMsgChan <- msg:
			default:
				s.logger.Warn("Room/game message buffer full, dropping message",
					"subject", msg.Subject,
					"bufferSize", s.config.RoomBufferSize)
			}
		})
		if err != nil {
			return err
		}
		s.shardSubscriptions = append(s.shardSubscriptions, sub)
	}
	return nil
}

func (s *MessageSubscriber) roomGameShardSubjects() []string {
	shard := s.config.RoomShardIndex
	return []string{
		sharedNats.BuildLogicRoomShardSubject(shard),
		sharedNats.BuildLogicGameShardSubject(shard),
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
		s.handler.HandleRoomRequest(ctx, message.Payload.RoomRequest, accessNodeId, message.ConnId, platform)
	case message.Payload.GameRequest != nil:
		s.handler.HandleGameRequest(ctx, message.Payload.GameRequest, accessNodeId, message.ConnId, platform)
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
	for _, sub := range s.shardSubscriptions {
		if err := sub.Unsubscribe(); err != nil {
			s.logger.Error("Failed to unsubscribe shard", "error", err)
		}
	}

	// 关闭消息通道
	if s.msgChan != nil {
		close(s.msgChan)
	}
	if s.shardMsgChan != nil {
		close(s.shardMsgChan)
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
