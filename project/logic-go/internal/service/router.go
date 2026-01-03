package service

import (
	"context"
	"log/slog"
	"sync"

	sharedModel "sudooom.im.shared/model"
	"sudooom.im.shared/proto"
)

// RouterService 路由服务（编排层）
type RouterService struct {
	locationService   *LocationService
	dispatcherService *DispatcherService
	logger            *slog.Logger
}

// NewRouterService 创建路由服务
func NewRouterService(locationService *LocationService, dispatcherService *DispatcherService) *RouterService {
	return &RouterService{
		locationService:   locationService,
		dispatcherService: dispatcherService,
		logger:            slog.Default(),
	}
}

// SendAckToUser 发送 ACK 给用户
func (s *RouterService) SendAckToUser(ctx context.Context, userId int64, clientMsgId string, serverMsgId int64) error {
	// 1. 查询用户位置
	locations, err := s.locationService.GetUserLocations(ctx, userId)
	if err != nil {
		return err
	}

	// 2. 分发 ACK
	return s.dispatcherService.DispatchAck(userId, locations, clientMsgId, serverMsgId)
}

// SendAckToUserDirect 直接发送 ACK 到指定的 Access 节点（用于回复发送者，使用 connId 避免查询）
func (s *RouterService) SendAckToUserDirect(ctx context.Context, accessNodeId string, connId int64, userId int64, clientMsgId string, serverMsgId int64) error {
	return s.dispatcherService.DispatchAckDirect(accessNodeId, connId, userId, clientMsgId, serverMsgId)
}

// SyncToSenderOtherDevices 同步消息给发送者的其他设备（多端同步）
func (s *RouterService) SyncToSenderOtherDevices(ctx context.Context, excludePlatform string, userId int64, msg *proto.UserMessage, serverMsgId int64) error {
	// 1. 查询用户所有设备位置
	locations, err := s.locationService.GetUserLocations(ctx, userId)
	if err != nil {
		return err
	}

	// 2. 过滤排除平台
	otherLocations := make([]sharedModel.UserLocation, 0, len(locations))
	for _, loc := range locations {
		if loc.Platform != excludePlatform {
			otherLocations = append(otherLocations, loc)
		}
	}

	// 3. 分发到其他设备
	return s.dispatcherService.DispatchPushMessage(userId, otherLocations, msg, serverMsgId)
}

// RouteMessage 路由消息到用户
func (s *RouterService) RouteMessage(ctx context.Context, userId int64, msg *proto.UserMessage, serverMsgId int64) error {
	// 1. 查询用户位置
	locations, err := s.locationService.GetUserLocations(ctx, userId)
	if err != nil {
		return err
	}

	if len(locations) == 0 {
		s.logger.Debug("User is offline", "userId", userId)
		return nil
	}

	// 2. 分发消息
	return s.dispatcherService.DispatchPushMessage(userId, locations, msg, serverMsgId)
}

// RouteToMultiple 批量路由消息（群消息）- 并行处理
func (s *RouterService) RouteToMultiple(ctx context.Context, userIds []int64, msg *proto.UserMessage, serverMsgId int64) error {
	// 1. 并发获取所有用户位置
	var wg sync.WaitGroup
	var mu sync.Mutex
	type userLoc struct {
		userId    int64
		locations []sharedModel.UserLocation
	}
	allUserLocations := make([]userLoc, 0, len(userIds))

	for _, userId := range userIds {
		wg.Add(1)
		go func(uid int64) {
			defer wg.Done()
			locs, err := s.locationService.GetUserLocations(ctx, uid)
			if err != nil {
				s.logger.Warn("Failed to get user locations", "userId", uid, "error", err)
				return
			}
			if len(locs) > 0 {
				mu.Lock()
				allUserLocations = append(allUserLocations, userLoc{userId: uid, locations: locs})
				mu.Unlock()
			}
		}(userId)
	}
	wg.Wait()

	// 2. 并发分发消息
	for _, ul := range allUserLocations {
		if err := s.dispatcherService.DispatchPushMessage(ul.userId, ul.locations, msg, serverMsgId); err != nil {
			s.logger.Warn("Failed to dispatch message to user", "userId", ul.userId, "error", err)
		}
	}

	return nil
}

// SendRoomPushToUser 发送房间推送到用户
func (s *RouterService) SendRoomPushToUser(ctx context.Context, accessNodeId string, userId int64, event string, roomId string, roomInfo []byte) error {
	return s.dispatcherService.DispatchRoomPush(accessNodeId, userId, event, roomId, roomInfo)
}

// SendGamePushToUser 发送游戏推送到用户
func (s *RouterService) SendGamePushToUser(ctx context.Context, accessNodeId string, userId int64, roomId string, gameType string, gamePayload []byte) error {
	return s.dispatcherService.DispatchGamePush(accessNodeId, userId, roomId, gameType, gamePayload)
}

// InvalidateUserCache 代理到 LocationService
func (s *RouterService) InvalidateUserCache(userId int64) {
	s.locationService.InvalidateCache(userId)
}
