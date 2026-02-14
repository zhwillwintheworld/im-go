package game

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"sudooom.im.logic/internal/service"
	"sudooom.im.shared/model"
)

// GameStarter 游戏启动器接口
// 由具体游戏服务（如 MahjongService）实现，解耦 game 包与具体游戏包
type GameStarter interface {
	// StartGameByType 根据内部游戏类型启动游戏
	StartGameByType(ctx context.Context, room *model.Room, internalGameType string) error
}

// RoomBroadcaster 房间广播接口（避免循环依赖）
type RoomBroadcaster interface {
	BroadcastToRoom(ctx context.Context, roomId string, event string, data interface{}) error
	GetRoom(ctx context.Context, roomId string) (*model.Room, error)
}

// GameService 游戏服务
// 注意：GameService 不使用 Redis，所有数据都在内存中（通过 RoomManager 管理）
// NATS 的一致性 Hash 保证同一房间的请求路由到同一 Logic 节点
type GameService struct {
	roomBroadcaster RoomBroadcaster        // 房间广播器（RoomService）
	routerService   *service.RouterService // 路由服务（用于消息分发）
	gameStarters    map[string]GameStarter // 外部游戏类型 → 启动器
	logger          *slog.Logger
}

// NewGameService 创建游戏服务
func NewGameService(
	roomBroadcaster RoomBroadcaster,
	routerService *service.RouterService,
) *GameService {
	return &GameService{
		roomBroadcaster: roomBroadcaster,
		routerService:   routerService,
		gameStarters:    make(map[string]GameStarter),
		logger:          slog.Default(),
	}
}

// RegisterGameStarter 注册游戏启动器
func (s *GameService) RegisterGameStarter(gameType string, starter GameStarter) {
	s.gameStarters[gameType] = starter
}

// BroadcastGameEvent 广播游戏事件给房间所有玩家（所有人收到相同消息）
// 通过 RoomBroadcaster 获取房间玩家列表（从内存中的 RoomManager）
func (s *GameService) BroadcastGameEvent(ctx context.Context, roomId string, event string, data interface{}) error {
	// 直接委托给 RoomBroadcaster
	return s.roomBroadcaster.BroadcastToRoom(ctx, roomId, event, data)
}

// SendPersonalizedGameEvents 给每个玩家发送个性化的游戏事件（并行发送）
// 注意：这里直接使用 routerService，因为需要给不同玩家发送不同的数据
func (s *GameService) SendPersonalizedGameEvents(ctx context.Context, roomId string, event string, userDataMap map[int64]interface{}) error {
	var wg sync.WaitGroup

	for userId, data := range userDataMap {
		wg.Add(1)
		go func(uid int64, d interface{}) {
			defer wg.Done()

			eventData, err := json.Marshal(d)
			if err != nil {
				s.logger.Error("Failed to marshal personalized event data",
					"error", err, "event", event, "userId", uid)
				return
			}

			// 直接使用 routerService 发送个性化消息
			if err := s.routerService.SendRoomPushToUsers(ctx, []int64{uid}, event, roomId, eventData); err != nil {
				s.logger.Warn("Failed to send personalized event",
					"error", err, "event", event, "userId", uid, "roomId", roomId)
			}
		}(userId, data)
	}

	wg.Wait()
	return nil
}

// StartGame 启动游戏（统一入口）
// 1. 查找内部类型映射
// 2. 通过注册的 GameStarter 初始化游戏引擎
// 3. 广播游戏开始事件
func (s *GameService) StartGame(ctx context.Context, room *model.Room) error {
	s.logger.Info("Starting game",
		"roomId", room.RoomID,
		"gameType", room.GameType,
		"playerCount", len(room.Players))

	// 1. 查找内部类型映射
	internalType, ok := GetInternalGameType(room.GameType)
	if !ok {
		s.logger.Warn("Unsupported game type", "gameType", room.GameType)
		return fmt.Errorf("unsupported game type: %s", room.GameType)
	}

	// 2. 查找并调用游戏启动器
	starter, ok := s.gameStarters[room.GameType]
	if !ok {
		s.logger.Warn("No starter registered for game type", "gameType", room.GameType)
		return fmt.Errorf("no starter registered for game type: %s", room.GameType)
	}

	if err := starter.StartGameByType(ctx, room, internalType); err != nil {
		return fmt.Errorf("failed to start game engine: %w", err)
	}

	// 3. 广播游戏开始事件
	return s.broadcastGameStart(ctx, room)
}

// broadcastGameStart 广播游戏开始事件
func (s *GameService) broadcastGameStart(ctx context.Context, room *model.Room) error {
	gameStartData := map[string]interface{}{
		"roomId":   room.RoomID,
		"gameType": room.GameType,
		"status":   "playing",
		"players":  room.Players,
	}

	if err := s.BroadcastGameEvent(ctx, room.RoomID, "GAME_STARTED", gameStartData); err != nil {
		return fmt.Errorf("failed to broadcast game start event: %w", err)
	}

	s.logger.Info("Game start event broadcasted", "roomId", room.RoomID)
	return nil
}
