package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

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

// filterOtherPlatformLocations 过滤排除指定平台的设备位置
func (s *RouterService) filterOtherPlatformLocations(locations []sharedModel.UserLocation, excludePlatform string) []sharedModel.UserLocation {
	otherLocations := make([]sharedModel.UserLocation, 0, len(locations))
	for _, loc := range locations {
		if loc.Platform != excludePlatform {
			otherLocations = append(otherLocations, loc)
		}
	}
	return otherLocations
}

// userLocationResult 用户位置查询结果
type userLocationResult struct {
	userId    int64
	locations []sharedModel.UserLocation
}

// fetchMultipleUserLocations 并发获取多个用户的位置信息
func (s *RouterService) fetchMultipleUserLocations(ctx context.Context, userIds []int64) []userLocationResult {
	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make([]userLocationResult, 0, len(userIds))

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
				results = append(results, userLocationResult{userId: uid, locations: locs})
				mu.Unlock()
			}
		}(userId)
	}
	wg.Wait()

	return results
}

// dispatchToSelfAndOtherDevices 通用方法：快速响应发起者并同步给其他设备
// dispatchDirect: 直接分发的回调函数
// dispatchToLocations: 分发到多个位置的回调函数
func (s *RouterService) dispatchToSelfAndOtherDevices(
	senderLoc sharedModel.UserLocation,
	dispatchDirect func() error,
	dispatchToLocations func([]sharedModel.UserLocation) error,
) error {
	// 1. 快速回复发起者
	if err := dispatchDirect(); err != nil {
		s.logger.Warn("Failed to send direct response", "userId", senderLoc.UserId, "error", err)
	}

	// 2. 同步给发起者的其他终端
	ctx := context.Background()
	locations, err := s.locationService.GetUserLocations(ctx, senderLoc.UserId)
	if err != nil {
		s.logger.Warn("Failed to get user locations for sync", "userId", senderLoc.UserId, "error", err)
		return nil // 不阻塞主流程
	}

	// 过滤排除当前平台
	otherLocations := s.filterOtherPlatformLocations(locations, senderLoc.Platform)

	// 分发到其他设备
	if len(otherLocations) > 0 {
		if err := dispatchToLocations(otherLocations); err != nil {
			s.logger.Warn("Failed to sync to other devices", "userId", senderLoc.UserId, "error", err)
		}
	}

	return nil
}

// SendAckToUserDirect 直接发送 ACK 到指定的 Access 节点（用于回复发送者，使用 connId 避免查询）
func (s *RouterService) SendAckToUserDirect(accessNodeId string, connId int64, userId int64, clientMsgId string, serverMsgId int64) error {
	// 构造单个 location 作为数组使用通用 Dispatch
	locations := []sharedModel.UserLocation{{
		AccessNodeId: accessNodeId,
		ConnId:       connId,
		UserId:       userId,
	}}
	payload := proto.DownstreamPayload{
		MessageAck: &proto.MessageAck{
			ClientMsgId: clientMsgId,
			ServerMsgId: serverMsgId,
			ToUserId:    userId,
			Timestamp:   time.Now().UnixMilli(),
		},
	}
	return s.dispatcherService.Dispatch(userId, locations, payload)
}

// SyncToSenderOtherDevices 同步消息给发送者的其他设备（多端同步）
func (s *RouterService) SyncToSenderOtherDevices(ctx context.Context, excludePlatform string, userId int64, msg *proto.UserMessage, serverMsgId int64) error {
	// 1. 查询用户所有设备位置
	locations, err := s.locationService.GetUserLocations(ctx, userId)
	if err != nil {
		return err
	}

	// 2. 过滤排除平台并分发到其他设备
	otherLocations := s.filterOtherPlatformLocations(locations, excludePlatform)
	payload := proto.DownstreamPayload{
		PushMessage: &proto.PushMessage{
			ServerMsgId: serverMsgId,
			FromUserId:  msg.FromUserId,
			ToUserId:    msg.ToUserId,
			ToGroupId:   msg.ToGroupId,
			MsgType:     msg.MsgType,
			Content:     msg.Content,
			Timestamp:   time.Now().UnixMilli(),
		},
	}
	return s.dispatcherService.Dispatch(userId, otherLocations, payload)
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
	payload := proto.DownstreamPayload{
		PushMessage: &proto.PushMessage{
			ServerMsgId: serverMsgId,
			FromUserId:  msg.FromUserId,
			ToUserId:    msg.ToUserId,
			ToGroupId:   msg.ToGroupId,
			MsgType:     msg.MsgType,
			Content:     msg.Content,
			Timestamp:   time.Now().UnixMilli(),
		},
	}
	return s.dispatcherService.Dispatch(userId, locations, payload)
}

// RouteToMultiple 批量路由消息（群消息）- 并行处理
func (s *RouterService) RouteToMultiple(ctx context.Context, userIds []int64, msg *proto.UserMessage, serverMsgId int64) error {
	// 1. 并发获取所有用户位置
	allUserLocations := s.fetchMultipleUserLocations(ctx, userIds)

	// 2. 分发消息
	payload := proto.DownstreamPayload{
		PushMessage: &proto.PushMessage{
			ServerMsgId: serverMsgId,
			FromUserId:  msg.FromUserId,
			ToUserId:    msg.ToUserId,
			ToGroupId:   msg.ToGroupId,
			MsgType:     msg.MsgType,
			Content:     msg.Content,
			Timestamp:   time.Now().UnixMilli(),
		},
	}
	for _, ul := range allUserLocations {
		if err := s.dispatcherService.Dispatch(ul.userId, ul.locations, payload); err != nil {
			s.logger.Warn("Failed to dispatch message to user", "userId", ul.userId, "error", err)
		}
	}

	return nil
}

// SendRoomPushToSelf 发送房间推送给自己（快速响应+多端同步）
func (s *RouterService) SendRoomPushToSelf(senderLoc sharedModel.UserLocation, event string, roomId string, roomInfo []byte) error {
	payload := proto.DownstreamPayload{
		RoomPush: &proto.RoomPush{
			Event:    event,
			RoomId:   roomId,
			UserId:   senderLoc.UserId,
			RoomInfo: roomInfo,
			ToUserId: senderLoc.UserId,
		},
	}
	return s.dispatchToSelfAndOtherDevices(
		senderLoc,
		func() error {
			return s.dispatcherService.Dispatch(senderLoc.UserId, []sharedModel.UserLocation{senderLoc}, payload)
		},
		func(otherLocations []sharedModel.UserLocation) error {
			return s.dispatcherService.Dispatch(senderLoc.UserId, otherLocations, payload)
		},
	)
}

// SendRoomPushToUsers 发送房间推送给多个用户（全量推送）
func (s *RouterService) SendRoomPushToUsers(ctx context.Context, userIds []int64, event string, roomId string, roomInfo []byte) error {
	// 1. 并发获取所有用户位置
	allUserLocations := s.fetchMultipleUserLocations(ctx, userIds)

	// 2. 分发到所有用户的所有设备
	for _, ul := range allUserLocations {
		payload := proto.DownstreamPayload{
			RoomPush: &proto.RoomPush{
				Event:    event,
				RoomId:   roomId,
				UserId:   ul.userId,
				RoomInfo: roomInfo,
				ToUserId: ul.userId,
			},
		}
		if err := s.dispatcherService.Dispatch(ul.userId, ul.locations, payload); err != nil {
			s.logger.Warn("Failed to dispatch room push to user", "userId", ul.userId, "error", err)
		}
	}

	return nil
}

// SendGamePushToSelf 发送游戏推送给自己（快速响应+多端同步）
func (s *RouterService) SendGamePushToSelf(senderLoc sharedModel.UserLocation, roomId string, gameType string, gamePayload []byte) error {
	payload := proto.DownstreamPayload{
		GamePush: &proto.GamePush{
			RoomId:      roomId,
			GameType:    gameType,
			GamePayload: gamePayload,
			ToUserId:    senderLoc.UserId,
		},
	}
	return s.dispatchToSelfAndOtherDevices(
		senderLoc,
		func() error {
			return s.dispatcherService.Dispatch(senderLoc.UserId, []sharedModel.UserLocation{senderLoc}, payload)
		},
		func(otherLocations []sharedModel.UserLocation) error {
			return s.dispatcherService.Dispatch(senderLoc.UserId, otherLocations, payload)
		},
	)
}

// SendGamePushToUsers 发送游戏推送给多个用户（全量推送）
func (s *RouterService) SendGamePushToUsers(ctx context.Context, userIds []int64, roomId string, gameType string, gamePayload []byte) error {
	// 1. 并发获取所有用户位置
	allUserLocations := s.fetchMultipleUserLocations(ctx, userIds)

	// 2. 分发到所有用户的所有设备
	for _, ul := range allUserLocations {
		payload := proto.DownstreamPayload{
			GamePush: &proto.GamePush{
				RoomId:      roomId,
				GameType:    gameType,
				GamePayload: gamePayload,
				ToUserId:    ul.userId,
			},
		}
		if err := s.dispatcherService.Dispatch(ul.userId, ul.locations, payload); err != nil {
			s.logger.Warn("Failed to dispatch game push to user", "userId", ul.userId, "error", err)
		}
	}

	return nil
}

// InvalidateUserCache 代理到 LocationService
func (s *RouterService) InvalidateUserCache(userId int64) {
	s.locationService.InvalidateCache(userId)
}

// GetLocationService 获取 LocationService（用于 room 包接口）
func (s *RouterService) GetLocationService() *LocationService {
	return s.locationService
}
