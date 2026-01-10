package game

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
	"sudooom.im.logic/internal/service"
	"sudooom.im.shared/model"
	sharedRedis "sudooom.im.shared/redis"
)

// GameService 游戏服务
type GameService struct {
	gameManager   *GameManager
	redisClient   *redis.Client
	routerService *service.RouterService
	logger        *slog.Logger
}

// NewGameService 创建游戏服务
func NewGameService(
	gameManager *GameManager,
	redisClient *redis.Client,
	routerService *service.RouterService,
) *GameService {
	return &GameService{
		gameManager:   gameManager,
		redisClient:   redisClient,
		routerService: routerService,
		logger:        slog.Default(),
	}
}

// BroadcastGameEvent 广播游戏事件给房间所有玩家（所有人收到相同消息）
// 适用场景：游戏开始、游戏结束、公共信息等
func (s *GameService) BroadcastGameEvent(ctx context.Context, roomId string, event string, data interface{}) error {
	eventData, err := json.Marshal(data)
	if err != nil {
		s.logger.Error("Failed to marshal game event data", "error", err, "event", event)
		return err
	}

	// 获取房间所有用户ID
	roomUsersKey := sharedRedis.BuildRoomUsersKey(roomId)
	userIdStrs, err := s.redisClient.SMembers(ctx, roomUsersKey).Result()
	if err != nil {
		s.logger.Warn("Failed to get room users", "error", err, "roomId", roomId)
		return err
	}

	// 转换为 int64 数组
	userIds := make([]int64, 0, len(userIdStrs))
	for _, userIdStr := range userIdStrs {
		var userId int64
		if _, err := fmt.Sscanf(userIdStr, "%d", &userId); err != nil {
			s.logger.Warn("Invalid user id in room users", "userIdStr", userIdStr)
			continue
		}
		userIds = append(userIds, userId)
	}

	// 广播事件
	if err := s.routerService.SendRoomPushToUsers(ctx, userIds, event, roomId, eventData); err != nil {
		s.logger.Warn("Failed to broadcast game event", "error", err, "event", event, "roomId", roomId)
		return err
	}

	return nil
}

// SendPersonalizedGameEvents 给每个玩家发送个性化的游戏事件
// 适用场景：发牌（每个玩家手牌不同）、摸牌（只有摸牌者知道牌面）等
// userDataMap: key 为 userId，value 为该玩家应该收到的数据
func (s *GameService) SendPersonalizedGameEvents(ctx context.Context, roomId string, event string, userDataMap map[int64]interface{}) error {
	// 遍历每个玩家，发送个性化消息
	for userId, data := range userDataMap {
		eventData, err := json.Marshal(data)
		if err != nil {
			s.logger.Error("Failed to marshal personalized event data",
				"error", err,
				"event", event,
				"userId", userId)
			continue // 不阻塞其他玩家的消息
		}

		// 给单个玩家发送消息
		if err := s.routerService.SendRoomPushToUsers(ctx, []int64{userId}, event, roomId, eventData); err != nil {
			s.logger.Warn("Failed to send personalized event",
				"error", err,
				"event", event,
				"userId", userId,
				"roomId", roomId)
			// 不阻塞其他玩家
		}
	}

	return nil
}

// StartGame 启动游戏（分发到具体游戏类型）
func (s *GameService) StartGame(ctx context.Context, room *model.Room) error {
	s.logger.Info("Starting game",
		"roomId", room.RoomID,
		"gameType", room.GameType,
		"playerCount", len(room.Players))

	// 根据游戏类型分发到具体的游戏服务
	switch room.GameType {
	case "HT_MAHJONG":
		return s.startMahjongGame(ctx, room)
	default:
		s.logger.Warn("Unsupported game type", "gameType", room.GameType)
		return fmt.Errorf("unsupported game type: %s", room.GameType)
	}
}

// startMahjongGame 启动麻将游戏（委托给 mahjong service）
func (s *GameService) startMahjongGame(ctx context.Context, room *model.Room) error {
	// TODO: 这里将委托给 MahjongService 处理
	// 目前先实现简单的广播逻辑

	// 广播游戏开始事件
	gameStartData := map[string]interface{}{
		"room_id":   room.RoomID,
		"game_type": room.GameType,
		"status":    "playing",
		"players":   room.Players,
	}

	if err := s.BroadcastGameEvent(ctx, room.RoomID, "GAME_STARTED", gameStartData); err != nil {
		return err
	}

	s.logger.Info("Game started successfully", "roomId", room.RoomID)
	return nil
}
