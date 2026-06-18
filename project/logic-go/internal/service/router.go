package service

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	sharedModel "sudooom.im.shared/model"
	"sudooom.im.shared/proto"
)

const (
	defaultFanoutWorkerCount        = 8
	defaultFanoutBufferSize         = 1024
	defaultLargeGroupThreshold      = 100
	defaultLocationQueryConcurrency = 8
	defaultLocationBatchSize        = 256
	defaultDispatchConcurrency      = 32
)

var ErrFanoutQueueFull = errors.New("fanout queue full")

type RouterConfig struct {
	FanoutWorkerCount        int
	FanoutBufferSize         int
	LargeGroupThreshold      int
	LocationQueryConcurrency int
	LocationBatchSize        int
	DispatchConcurrency      int
}

type fanoutTask struct {
	userIds []int64
	payload proto.DownstreamPayload
}

// RouterService 路由服务（编排层）
type RouterService struct {
	locationService   *LocationService
	dispatcherService *DispatcherService
	config            RouterConfig
	fanoutQueue       chan fanoutTask
	stopChan          chan struct{}
	stopOnce          sync.Once
	wg                sync.WaitGroup
	fanoutDropped     atomic.Int64
	dispatch          func(userId int64, locations []sharedModel.UserLocation, payload proto.DownstreamPayload) error
	logger            *slog.Logger
}

// NewRouterService 创建路由服务
func NewRouterService(locationService *LocationService, dispatcherService *DispatcherService) *RouterService {
	return NewRouterServiceWithConfig(locationService, dispatcherService, RouterConfig{})
}

// NewRouterServiceWithConfig 创建带 fan-out 配置的路由服务。
func NewRouterServiceWithConfig(locationService *LocationService, dispatcherService *DispatcherService, cfg RouterConfig) *RouterService {
	cfg = normalizeRouterConfig(cfg)
	s := &RouterService{
		locationService:   locationService,
		dispatcherService: dispatcherService,
		config:            cfg,
		fanoutQueue:       make(chan fanoutTask, cfg.FanoutBufferSize),
		stopChan:          make(chan struct{}),
		logger:            slog.Default(),
	}
	if dispatcherService != nil {
		s.dispatch = dispatcherService.Dispatch
	} else {
		s.dispatch = func(userId int64, locations []sharedModel.UserLocation, payload proto.DownstreamPayload) error {
			return nil
		}
	}
	s.startFanoutWorkers()
	return s
}

func normalizeRouterConfig(cfg RouterConfig) RouterConfig {
	if cfg.FanoutWorkerCount <= 0 {
		cfg.FanoutWorkerCount = defaultFanoutWorkerCount
	}
	if cfg.FanoutBufferSize <= 0 {
		cfg.FanoutBufferSize = defaultFanoutBufferSize
	}
	if cfg.LargeGroupThreshold <= 0 {
		cfg.LargeGroupThreshold = defaultLargeGroupThreshold
	}
	if cfg.LocationQueryConcurrency <= 0 {
		cfg.LocationQueryConcurrency = defaultLocationQueryConcurrency
	}
	if cfg.LocationBatchSize <= 0 {
		cfg.LocationBatchSize = defaultLocationBatchSize
	}
	if cfg.DispatchConcurrency <= 0 {
		cfg.DispatchConcurrency = defaultDispatchConcurrency
	}
	return cfg
}

func (s *RouterService) startFanoutWorkers() {
	for i := 0; i < s.config.FanoutWorkerCount; i++ {
		workerID := i
		s.wg.Add(1)
		go s.fanoutWorker(workerID)
	}
}

func (s *RouterService) fanoutWorker(workerID int) {
	defer s.wg.Done()

	for {
		select {
		case task := <-s.fanoutQueue:
			if err := s.routePayloadToMultiple(context.Background(), task.userIds, task.payload); err != nil {
				s.logger.Warn("Failed to process fanout task", "workerId", workerID, "error", err)
			}
		case <-s.stopChan:
			s.logger.Info("Fanout worker stopped", "workerId", workerID)
			return
		}
	}
}

// Stop 停止 fan-out worker。
func (s *RouterService) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopChan)
	})
	s.wg.Wait()
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
	if len(userIds) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make([]userLocationResult, 0, len(userIds))
	locationBatchSize := s.config.LocationBatchSize
	if locationBatchSize <= 0 {
		locationBatchSize = defaultLocationBatchSize
	}
	locationQueryConcurrency := s.config.LocationQueryConcurrency
	if locationQueryConcurrency <= 0 {
		locationQueryConcurrency = defaultLocationQueryConcurrency
	}
	chunks := chunkInt64s(userIds, locationBatchSize)
	sem := make(chan struct{}, locationQueryConcurrency)

	for _, chunk := range chunks {
		chunkUserIds := chunk
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() {
				<-sem
			}()

			locationsByUser, err := s.locationService.GetUsersLocations(ctx, chunkUserIds)
			if err != nil {
				s.logger.Warn("Failed to get user locations", "count", len(chunkUserIds), "error", err)
				return
			}

			chunkResults := make([]userLocationResult, 0, len(chunkUserIds))
			for _, uid := range chunkUserIds {
				locs := locationsByUser[uid]
				if len(locs) > 0 {
					chunkResults = append(chunkResults, userLocationResult{userId: uid, locations: locs})
				}
			}
			if len(chunkResults) > 0 {
				mu.Lock()
				results = append(results, chunkResults...)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	return results
}

func chunkInt64s(values []int64, size int) [][]int64 {
	if len(values) == 0 {
		return nil
	}
	if size <= 0 || size >= len(values) {
		return [][]int64{values}
	}

	chunks := make([][]int64, 0, (len(values)+size-1)/size)
	for start := 0; start < len(values); start += size {
		end := start + size
		if end > len(values) {
			end = len(values)
		}
		chunks = append(chunks, values[start:end])
	}
	return chunks
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
func (s *RouterService) SendAckToUserDirect(accessNodeId string, connId int64, userId int64, clientMsgId string, serverMsgId int64, status proto.MessageAckStatus) error {
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
			Status:      status,
		},
	}
	return s.dispatch(userId, locations, payload)
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
	return s.dispatch(userId, otherLocations, payload)
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
	return s.dispatch(userId, locations, payload)
}

// RouteToMultiple 批量路由消息（群消息）- 并行处理
func (s *RouterService) RouteToMultiple(ctx context.Context, userIds []int64, msg *proto.UserMessage, serverMsgId int64) error {
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

	if len(userIds) >= s.config.LargeGroupThreshold {
		return s.enqueueFanout(userIds, payload)
	}

	return s.routePayloadToMultiple(ctx, userIds, payload)
}

func (s *RouterService) enqueueFanout(userIds []int64, payload proto.DownstreamPayload) error {
	task := fanoutTask{
		userIds: cloneInt64s(userIds),
		payload: cloneDownstreamPayload(payload),
	}
	select {
	case s.fanoutQueue <- task:
		return nil
	default:
		s.fanoutDropped.Add(1)
		s.logger.Warn("Fanout queue full, drop task", "userCount", len(userIds))
		return ErrFanoutQueueFull
	}
}

type RouterStats struct {
	FanoutQueueSize     int
	FanoutQueueCapacity int
	FanoutDroppedCount  int64
}

// Stats 返回 RouterService 当前轻量观测快照。
func (s *RouterService) Stats() RouterStats {
	return RouterStats{
		FanoutQueueSize:     len(s.fanoutQueue),
		FanoutQueueCapacity: cap(s.fanoutQueue),
		FanoutDroppedCount:  s.fanoutDropped.Load(),
	}
}

func (s *RouterService) routePayloadToMultiple(ctx context.Context, userIds []int64, payload proto.DownstreamPayload) error {
	allUserLocations := s.fetchMultipleUserLocations(ctx, userIds)
	s.dispatchUserLocations(allUserLocations, payload)
	return nil
}

func (s *RouterService) dispatchUserLocations(allUserLocations []userLocationResult, payload proto.DownstreamPayload) {
	if len(allUserLocations) == 0 {
		return
	}

	var wg sync.WaitGroup
	dispatchConcurrency := s.config.DispatchConcurrency
	if dispatchConcurrency <= 0 {
		dispatchConcurrency = defaultDispatchConcurrency
	}
	sem := make(chan struct{}, dispatchConcurrency)
	for _, ul := range allUserLocations {
		ul := ul
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() {
				<-sem
			}()
			if err := s.dispatch(ul.userId, ul.locations, payload); err != nil {
				s.logger.Warn("Failed to dispatch message to user", "userId", ul.userId, "error", err)
			}
		}()
	}
	wg.Wait()
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
			return s.dispatch(senderLoc.UserId, []sharedModel.UserLocation{senderLoc}, payload)
		},
		func(otherLocations []sharedModel.UserLocation) error {
			return s.dispatch(senderLoc.UserId, otherLocations, payload)
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
		s.dispatchUserLocations([]userLocationResult{ul}, payload)
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
			return s.dispatch(senderLoc.UserId, []sharedModel.UserLocation{senderLoc}, payload)
		},
		func(otherLocations []sharedModel.UserLocation) error {
			return s.dispatch(senderLoc.UserId, otherLocations, payload)
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
		s.dispatchUserLocations([]userLocationResult{ul}, payload)
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

func cloneDownstreamPayload(payload proto.DownstreamPayload) proto.DownstreamPayload {
	cloned := payload
	if payload.MessageAck != nil {
		messageAck := *payload.MessageAck
		cloned.MessageAck = &messageAck
	}
	if payload.PushMessage != nil {
		pushMessage := *payload.PushMessage
		pushMessage.Content = cloneBytes(payload.PushMessage.Content)
		cloned.PushMessage = &pushMessage
	}
	if payload.RoomPush != nil {
		roomPush := *payload.RoomPush
		roomPush.RoomInfo = cloneBytes(payload.RoomPush.RoomInfo)
		cloned.RoomPush = &roomPush
	}
	if payload.GamePush != nil {
		gamePush := *payload.GamePush
		gamePush.GamePayload = cloneBytes(payload.GamePush.GamePayload)
		cloned.GamePush = &gamePush
	}
	return cloned
}

func cloneBytes(values []byte) []byte {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]byte, len(values))
	copy(cloned, values)
	return cloned
}
