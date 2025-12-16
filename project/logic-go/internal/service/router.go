package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"sudooom.im.logic/internal/nats"
	sharedModel "sudooom.im.shared/model"
	"sudooom.im.shared/proto"
	sharedRedis "sudooom.im.shared/redis"
)

// MessagePublisher 消息发布器接口
type MessagePublisher interface {
	PublishToAccess(accessNodeId string, message *proto.DownstreamMessage) error
}

// RouterService 路由服务
type RouterService struct {
	redisClient *redis.Client
	publisher   *nats.MessagePublisher
	logger      *slog.Logger
}

// NewRouterService 创建路由服务
func NewRouterService(redisClient *redis.Client, publisher *nats.MessagePublisher) *RouterService {
	return &RouterService{
		redisClient: redisClient,
		publisher:   publisher,
		logger:      slog.Default(),
	}
}

// GetUserLocations 获取用户所在的 Access 节点
func (s *RouterService) GetUserLocations(ctx context.Context, userId int64) ([]sharedModel.UserLocation, error) {
	key := sharedRedis.BuildUserLocationKey(userId)

	entries, err := s.redisClient.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	locations := make([]sharedModel.UserLocation, 0, len(entries))
	for _, value := range entries {
		loc, err := sharedRedis.ParseUserLocation(value)
		if err != nil {
			continue
		}
		locations = append(locations, *loc)
	}

	return locations, nil
}

// SendAckToUser 发送 ACK 给用户
func (s *RouterService) SendAckToUser(ctx context.Context, userId int64, clientMsgId string, serverMsgId int64) error {
	locations, err := s.GetUserLocations(ctx, userId)
	if err != nil {
		return err
	}

	for _, loc := range locations {
		ackMsg := &proto.DownstreamMessage{
			Payload: proto.DownstreamPayload{
				MessageAck: &proto.MessageAck{
					ClientMsgId: clientMsgId,
					ServerMsgId: serverMsgId,
					ToUserId:    userId,
					Timestamp:   time.Now().UnixMilli(),
				},
			},
		}
		if err := s.publisher.PublishToAccess(loc.AccessNodeId, ackMsg); err != nil {
			s.logger.Warn("Failed to send ack to user", "userId", userId, "accessNodeId", loc.AccessNodeId, "error", err)
		}
	}

	return nil
}

// RouteMessage 路由消息到用户
func (s *RouterService) RouteMessage(ctx context.Context, userId int64, msg *proto.UserMessage, serverMsgId int64) error {
	locations, err := s.GetUserLocations(ctx, userId)
	if err != nil {
		return err
	}

	if len(locations) == 0 {
		s.logger.Debug("User is offline, saving to offline storage", "userId", userId)
		// TODO: offlineMessageService.Save(userId, message)
		return nil
	}

	// 按 Access 节点分组并行推送
	nodeLocations := make(map[string][]sharedModel.UserLocation)
	for _, loc := range locations {
		nodeLocations[loc.AccessNodeId] = append(nodeLocations[loc.AccessNodeId], loc)
	}

	var wg sync.WaitGroup
	for accessNodeId := range nodeLocations {
		wg.Add(1)
		go func(nodeId string) {
			defer wg.Done()
			downstreamMsg := &proto.DownstreamMessage{
				Payload: proto.DownstreamPayload{
					PushMessage: &proto.PushMessage{
						ServerMsgId: serverMsgId,
						FromUserId:  msg.FromUserId,
						ToUserId:    msg.ToUserId,
						ToGroupId:   msg.ToGroupId,
						MsgType:     msg.MsgType,
						Content:     msg.Content,
						Timestamp:   time.Now().UnixMilli(),
					},
				},
			}
			if err := s.publisher.PublishToAccess(nodeId, downstreamMsg); err != nil {
				s.logger.Warn("Failed to route message to access node",
					"accessNodeId", nodeId,
					"error", err)
			}
		}(accessNodeId)
	}
	wg.Wait()

	return nil
}

// RouteToMultiple 批量路由消息（群消息）- 并行处理
func (s *RouterService) RouteToMultiple(ctx context.Context, userIds []int64, msg *proto.UserMessage, serverMsgId int64) error {
	// 并行获取所有用户位置
	type userLoc struct {
		userId    int64
		locations []sharedModel.UserLocation
	}

	results := make(chan userLoc, len(userIds))
	var wg sync.WaitGroup

	for _, userId := range userIds {
		wg.Add(1)
		go func(uid int64) {
			defer wg.Done()
			locs, _ := s.GetUserLocations(ctx, uid)
			results <- userLoc{userId: uid, locations: locs}
		}(userId)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// 按 Access 节点分组
	nodeToUsers := make(map[string][]int64)
	for result := range results {
		for _, loc := range result.locations {
			nodeToUsers[loc.AccessNodeId] = append(nodeToUsers[loc.AccessNodeId], result.userId)
		}
	}

	// 并行发送
	var sendWg sync.WaitGroup
	for accessNodeId, users := range nodeToUsers {
		sendWg.Add(1)
		go func(nodeId string, targetUsers []int64) {
			defer sendWg.Done()
			for range targetUsers {
				downstreamMsg := &proto.DownstreamMessage{
					Payload: proto.DownstreamPayload{
						PushMessage: &proto.PushMessage{
							ServerMsgId: serverMsgId,
							FromUserId:  msg.FromUserId,
							ToUserId:    msg.ToUserId,
							ToGroupId:   msg.ToGroupId,
							MsgType:     msg.MsgType,
							Content:     msg.Content,
							Timestamp:   time.Now().UnixMilli(),
						},
					},
				}
				s.publisher.PublishToAccess(nodeId, downstreamMsg)
			}
		}(accessNodeId, users)
	}
	sendWg.Wait()

	return nil
}
