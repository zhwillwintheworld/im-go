package room

import (
	"context"

	"sudooom.im.shared/model"
)

// ============================================================================
// 房间操作参数定义
// ============================================================================

// CreateRoomParams 创建房间参数
type CreateRoomParams struct {
	UserId       int64
	RoomName     string
	RoomType     string
	RoomPassword string
	MaxPlayers   int
	GameType     string
	GameSettings map[string]string
}

// JoinRoomParams 加入房间参数
type JoinRoomParams struct {
	UserId   int64
	RoomId   string
	Password string
}

// LeaveRoomParams 离开房间参数
type LeaveRoomParams struct {
	UserId int64
	RoomId string
}

// ReadyRoomParams 准备参数
type ReadyRoomParams struct {
	UserId int64
	RoomId string
}

// ChangeSeatParams 换座位参数
type ChangeSeatParams struct {
	UserId     int64
	RoomId     string
	TargetSeat int32
}

// StartGameParams 开始游戏参数
type StartGameParams struct {
	UserId int64
	RoomId string
}

// ============================================================================
// 房间操作方法
// ============================================================================

// CreateRoom 创建房间
func (s *RoomService) CreateRoom(ctx context.Context, params CreateRoomParams) (*model.Room, error) {
	// 生成房间 ID
	roomId := s.sfNode.Generate().String()

	// 构建房间配置
	config := &model.RoomConfig{
		RoomName:     params.RoomName,
		RoomPassword: params.RoomPassword,
		RoomType:     params.RoomType,
		MaxPlayers:   params.MaxPlayers,
		GameSettings: params.GameSettings,
	}

	// 创建房间（通过 RoomManager）
	room := s.roomManager.GetOrCreate(roomId, params.UserId, config, params.GameType)

	// 获取策略
	strategy, err := s.getGameTypeStrategy(params.GameType)
	if err != nil {
		return nil, err
	}

	seatIndex, err := strategy.AllocateSeat(room.roomInfo)
	if err != nil {
		s.logger.Error("Failed to allocate seat for creator", "error", err, "roomId", roomId)
		return nil, err
	}

	// 获取用户信息
	userInfo := s.getUserInfo(ctx, params.UserId)

	// 房主自动加入房间
	if err := room.Join(params.UserId, seatIndex, userInfo); err != nil {
		s.logger.Error("Failed to join room as creator", "error", err, "roomId", roomId)
		return nil, err
	}

	s.logger.Info("Room created", "roomId", roomId, "creator", params.UserId)

	return room.CopyRoomInfo(), nil
}

// JoinRoom 加入房间
func (s *RoomService) JoinRoom(ctx context.Context, params JoinRoomParams) (*model.Room, error) {
	// 获取房间
	room, ok := s.roomManager.Get(params.RoomId)
	if !ok {
		return nil, ErrRoomNotFound
	}

	// 获取策略（直接访问 roomInfo，避免不必要的拷贝）
	strategy, err := s.getGameTypeStrategy(room.roomInfo.GameType)
	if err != nil {
		return nil, err
	}

	// 使用策略自动分配座位
	seatIndex, err := strategy.AllocateSeat(room.roomInfo)
	if err != nil {
		s.logger.Warn("Failed to allocate seat", "error", err, "userId", params.UserId, "roomId", params.RoomId)
		return nil, err
	}

	// 获取用户信息
	userInfo := s.getUserInfo(ctx, params.UserId)

	// 加入房间
	if err := room.Join(params.UserId, seatIndex, userInfo); err != nil {
		s.logger.Warn("Failed to join room", "error", err, "userId", params.UserId, "roomId", params.RoomId)
		return nil, err
	}

	s.logger.Info("User joined room", "userId", params.UserId, "roomId", params.RoomId, "seatIndex", seatIndex)

	// 广播房间更新
	snapshot := room.CopyRoomInfo()
	err = s.BroadcastToRoom(ctx, params.RoomId, "ROOM_UPDATED", snapshot)
	if err != nil {
		return nil, err
	}

	return snapshot, nil
}

// LeaveRoom 离开房间
func (s *RoomService) LeaveRoom(ctx context.Context, params LeaveRoomParams) (*model.Room, error) {
	// 获取房间
	room, ok := s.roomManager.Get(params.RoomId)
	if !ok {
		return nil, ErrRoomNotFound
	}

	// 离开房间
	if err := room.Leave(params.UserId); err != nil {
		s.logger.Warn("Failed to leave room", "error", err, "userId", params.UserId, "roomId", params.RoomId)
		return nil, err
	}

	s.logger.Info("User left room", "userId", params.UserId, "roomId", params.RoomId)

	// 广播房间更新
	snapshot := room.CopyRoomInfo()
	err := s.BroadcastToRoom(ctx, params.RoomId, "ROOM_UPDATED", snapshot)
	if err != nil {
		return nil, err
	}

	return snapshot, nil
}

// ReadyRoom 准备/取消准备
func (s *RoomService) ReadyRoom(ctx context.Context, params ReadyRoomParams) (*model.Room, error) {
	// 获取房间
	room, ok := s.roomManager.Get(params.RoomId)
	if !ok {
		return nil, ErrRoomNotFound
	}

	// 切换准备状态
	if err := room.Ready(params.UserId); err != nil {
		s.logger.Warn("Failed to ready", "error", err, "userId", params.UserId, "roomId", params.RoomId)
		return nil, err
	}

	s.logger.Info("User ready status changed", "userId", params.UserId, "roomId", params.RoomId)

	// 广播房间更新
	snapshot := room.CopyRoomInfo()
	err := s.BroadcastToRoom(ctx, params.RoomId, "ROOM_UPDATED", snapshot)
	if err != nil {
		return nil, err
	}

	return snapshot, nil
}

// ChangeSeat 换座位
func (s *RoomService) ChangeSeat(ctx context.Context, params ChangeSeatParams) (*model.Room, error) {
	// 获取房间
	room, ok := s.roomManager.Get(params.RoomId)
	if !ok {
		return nil, ErrRoomNotFound
	}

	// 换座位
	if err := room.ChangeSeat(params.UserId, params.TargetSeat); err != nil {
		s.logger.Warn("Failed to change seat", "error", err, "userId", params.UserId, "roomId", params.RoomId)
		return nil, err
	}

	s.logger.Info("User changed seat", "userId", params.UserId, "roomId", params.RoomId, "targetSeat", params.TargetSeat)

	// 广播房间更新
	snapshot := room.CopyRoomInfo()
	err := s.BroadcastToRoom(ctx, params.RoomId, "ROOM_UPDATED", snapshot)
	if err != nil {
		return nil, err
	}

	return snapshot, nil
}

// StartGame 开始游戏
func (s *RoomService) StartGame(ctx context.Context, params StartGameParams) (*model.Room, error) {
	// 获取房间
	r, ok := s.roomManager.Get(params.RoomId)
	if !ok {
		return nil, ErrRoomNotFound
	}

	// 获取策略（直接访问 roomInfo）
	strategy, err := s.getGameTypeStrategy(r.roomInfo.GameType)
	if err != nil {
		return nil, err
	}

	// 开始游戏（策略验证在 Room.StartGame 内部）
	if err := r.StartGame(params.UserId, strategy); err != nil {
		s.logger.Warn("Failed to start game", "error", err, "userId", params.UserId, "roomId", params.RoomId)
		return nil, err
	}

	s.logger.Info("Game started successfully", "userId", params.UserId, "roomId", params.RoomId)

	// 广播游戏开始
	snapshot := r.CopyRoomInfo()
	err = s.BroadcastToRoom(ctx, params.RoomId, "GAME_STARTED", snapshot)
	if err != nil {
		return nil, err
	}

	return snapshot, nil
}

// getGameTypeStrategy 根据游戏类型获取对应的策略
func (s *RoomService) getGameTypeStrategy(gameType string) (GameTypeStrategy, error) {
	switch gameType {
	case "HT_MAHJONG":
		return &MahjongGameStrategy{}, nil
	default:
		return nil, ErrUnsupportedGameType
	}
}
