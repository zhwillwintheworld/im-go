package room

import (
	"context"
	"encoding/json"
	"log/slog"

	"sudooom.im.logic/internal/service"
	"sudooom.im.shared/model"
	"sudooom.im.shared/snowflake"
)

// RoomService 房间服务
// 注意：房间数据在内存中（RoomManager），用户信息从 UserInfoProvider 获取
type RoomService struct {
	roomManager      *RoomManager
	userInfoProvider UserInfoProvider // 用户信息提供者（可以是 Redis、数据库或其他）
	sfNode           *snowflake.Node
	routerService    *service.RouterService
	logger           *slog.Logger
}

// NewRoomService 创建房间服务
func NewRoomService(
	roomManager *RoomManager,
	userInfoProvider UserInfoProvider,
	sfNode *snowflake.Node,
	routerService *service.RouterService,
) *RoomService {
	return &RoomService{
		roomManager:      roomManager,
		userInfoProvider: userInfoProvider,
		sfNode:           sfNode,
		routerService:    routerService,
		logger:           slog.Default(),
	}
}

// getUserInfo 获取用户信息（委托给 UserInfoProvider）
func (s *RoomService) getUserInfo(ctx context.Context, userId int64) *model.User {
	return s.userInfoProvider.GetUserInfo(ctx, userId)
}

// BroadcastToRoom 广播消息给房间所有用户
func (s *RoomService) BroadcastToRoom(ctx context.Context, roomId string, event string, data interface{}) error {
	// 获取房间快照
	r, ok := s.roomManager.Get(roomId)
	if !ok {
		return ErrRoomNotFound
	}

	snapshot := r.CopyRoomInfo()

	// 提取用户 ID 列表
	userIds := make([]int64, 0, len(snapshot.Players))
	for _, player := range snapshot.Players {
		userIds = append(userIds, player.UserID)
	}

	// 序列化数据
	eventData, err := json.Marshal(data)
	if err != nil {
		s.logger.Error("Failed to marshal event data", "error", err, "event", event)
		return err
	}

	// 通过 RouterService 广播
	if err := s.routerService.SendRoomPushToUsers(ctx, userIds, event, roomId, eventData); err != nil {
		s.logger.Warn("Failed to broadcast to room", "error", err, "roomId", roomId, "event", event)
		return err
	}

	return nil
}

// GetRoom 获取房间信息
func (s *RoomService) GetRoom(ctx context.Context, roomId string) (*model.Room, error) {
	r, ok := s.roomManager.Get(roomId)
	if !ok {
		return nil, ErrRoomNotFound
	}

	return r.CopyRoomInfo(), nil
}
