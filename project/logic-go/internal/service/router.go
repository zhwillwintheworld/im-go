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

// AllPlatforms 支持的所有平台列表
var AllPlatforms = []string{"android", "ios", "web", "desktop", "wechat"}

// GetUserLocations 获取用户所在的所有 Access 节点（遍历所有平台）
func (s *RouterService) GetUserLocations(ctx context.Context, userId int64) ([]sharedModel.UserLocation, error) {
	locations := make([]sharedModel.UserLocation, 0, len(AllPlatforms))

	// 构建所有平台的 key
	keys := make([]string, len(AllPlatforms))
	for i, platform := range AllPlatforms {
		keys[i] = sharedRedis.BuildUserLocationKeyWithPlatform(userId, platform)
	}

	// 批量获取
	results, err := s.redisClient.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	for i, result := range results {
		if result == nil {
			continue
		}
		accessNodeId, ok := result.(string)
		if !ok || accessNodeId == "" {
			continue
		}
		locations = append(locations, sharedModel.UserLocation{
			UserId:       userId,
			AccessNodeId: accessNodeId,
			Platform:     AllPlatforms[i],
		})
	}

	return locations, nil
}

// GetUserLocationsByPlatforms 获取用户在指定平台的位置
func (s *RouterService) GetUserLocationsByPlatforms(ctx context.Context, userId int64, platforms []string) ([]sharedModel.UserLocation, error) {
	if len(platforms) == 0 {
		return nil, nil
	}

	locations := make([]sharedModel.UserLocation, 0, len(platforms))

	keys := make([]string, len(platforms))
	for i, platform := range platforms {
		keys[i] = sharedRedis.BuildUserLocationKeyWithPlatform(userId, platform)
	}

	results, err := s.redisClient.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	for i, result := range results {
		if result == nil {
			continue
		}
		accessNodeId, ok := result.(string)
		if !ok || accessNodeId == "" {
			continue
		}
		locations = append(locations, sharedModel.UserLocation{
			UserId:       userId,
			AccessNodeId: accessNodeId,
			Platform:     platforms[i],
		})
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

// SendAckToUserDirect 直接发送 ACK 到指定的 Access 节点（用于回复发送者，避免 Redis 查询）
func (s *RouterService) SendAckToUserDirect(ctx context.Context, accessNodeId string, userId int64, clientMsgId string, serverMsgId int64) error {
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
	if err := s.publisher.PublishToAccess(accessNodeId, ackMsg); err != nil {
		s.logger.Warn("Failed to send ack to user", "userId", userId, "accessNodeId", accessNodeId, "error", err)
		return err
	}
	return nil
}

// SyncToSenderOtherDevices 同步消息给发送者的其他设备（多端同步）
// excludePlatform: 发送消息的平台，排除该平台
func (s *RouterService) SyncToSenderOtherDevices(ctx context.Context, excludePlatform string, userId int64, msg *proto.UserMessage, serverMsgId int64) error {
	// 获取除发送平台外的其他平台
	otherPlatforms := make([]string, 0, len(AllPlatforms)-1)
	for _, p := range AllPlatforms {
		if p != excludePlatform {
			otherPlatforms = append(otherPlatforms, p)
		}
	}

	// 获取其他平台的位置
	locations, err := s.GetUserLocationsByPlatforms(ctx, userId, otherPlatforms)
	if err != nil {
		return err
	}

	// 没有其他设备在线
	if len(locations) == 0 {
		return nil
	}

	// 发送给其他设备
	for _, loc := range locations {
		syncMsg := &proto.DownstreamMessage{
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
		if err := s.publisher.PublishToAccess(loc.AccessNodeId, syncMsg); err != nil {
			s.logger.Warn("Failed to sync message to sender's other device",
				"userId", userId,
				"platform", loc.Platform,
				"accessNodeId", loc.AccessNodeId,
				"error", err,
			)
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
		s.logger.Debug("User is offline", "userId", userId)
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
