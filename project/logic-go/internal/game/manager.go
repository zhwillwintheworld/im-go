package game

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"sudooom.im.logic/internal/game/mahjong/core"
	"sudooom.im.logic/internal/game/mahjong/htmajong"
	"sudooom.im.logic/internal/game/mahjong/thmahjong"
	"sudooom.im.logic/internal/service"
	"sudooom.im.logic/internal/task"
	"sudooom.im.shared/model"
	"sudooom.im.shared/snowflake"
)

// RoomBroadcaster 房间广播接口（避免循环依赖）
type RoomBroadcaster interface {
	BroadcastToRoom(ctx context.Context, roomId string, event string, data interface{}) error
	GetRoom(ctx context.Context, roomId string) (*model.Room, error)
}

// GameManager 游戏管理器（统一管理游戏实例和引擎创建）
// 合并了原来的 GameService、MahjongService、MultiRoundGameService、MultiRoundGameManager
type GameManager struct {
	games     sync.Map     // roomID -> *MultiRoundGame
	gameCount atomic.Int64 // 原子计数器

	// LRU 配置
	maxGames     int
	evictTimeout time.Duration
	evictTicker  *time.Ticker

	stopChan chan struct{} // 停止信号通道

	// 依赖
	roomBroadcaster RoomBroadcaster        // 房间广播器（RoomService）
	routerService   *service.RouterService // 路由服务（用于消息分发）
	scheduler       *task.Scheduler        // 任务调度器
	sfNode          *snowflake.Node        // 雪花ID生成器

	logger *slog.Logger
}

// NewGameManager 创建游戏管理器
func NewGameManager(
	maxGames int,
	evictTimeout time.Duration,
	roomBroadcaster RoomBroadcaster,
	routerService *service.RouterService,
	scheduler *task.Scheduler,
	sfNode *snowflake.Node,
) *GameManager {
	m := &GameManager{
		maxGames:        maxGames,
		evictTimeout:    evictTimeout,
		evictTicker:     time.NewTicker(60 * time.Second),
		stopChan:        make(chan struct{}),
		roomBroadcaster: roomBroadcaster,
		routerService:   routerService,
		scheduler:       scheduler,
		sfNode:          sfNode,
		logger:          slog.Default().With("component", "GameManager"),
	}

	go m.evictLoop()

	return m
}

// StartGame 启动游戏（统一入口）
func (m *GameManager) StartGame(ctx context.Context, room *model.Room) error {
	m.logger.Info("Starting game",
		"roomId", room.RoomID,
		"gameType", room.GameType,
		"playerCount", len(room.Players))

	// 1. 查找内部类型映射
	internalType, ok := GetInternalGameType(room.GameType)
	if !ok {
		m.logger.Warn("Unsupported game type", "gameType", room.GameType)
		return fmt.Errorf("unsupported game type: %s", room.GameType)
	}

	// 2. 创建游戏实例
	config := GameConfig{
		MaxRounds: 8, // 默认 8 局
		BaseScore: 1, // 默认底分 1
	}

	game, err := m.createGame(room.RoomID, internalType, config)
	if err != nil {
		return fmt.Errorf("failed to create game: %w", err)
	}

	// 3. 开始第一局
	playerIDs := extractPlayerIDs(room.Players)
	engineFactory := func() RoundEngine {
		return m.createEngine(internalType)
	}

	round, err := game.StartNewRound(ctx, playerIDs, engineFactory)
	if err != nil {
		m.removeGame(room.RoomID)
		return fmt.Errorf("failed to start first round: %w", err)
	}

	m.logger.Info("First round started",
		"roomId", room.RoomID,
		"roundId", round.ID,
		"playerCount", len(playerIDs))

	// 4. 广播游戏开始事件
	return m.broadcastGameStart(ctx, room)
}

// HandlePlayerAction 处理玩家操作
func (m *GameManager) HandlePlayerAction(ctx context.Context, roomID string, userID int64, action core.Action) error {
	// 获取游戏
	game, ok := m.getGame(roomID)
	if !ok {
		return ErrGameNotFound
	}

	// 获取当前局
	round := game.GetCurrentRound()
	if round == nil {
		return fmt.Errorf("no current round")
	}

	// 处理动作（Round 内部有锁保护）
	if err := round.HandlePlayerAction(ctx, userID, action); err != nil {
		m.logger.Error("Failed to handle player action",
			"error", err,
			"roomId", roomID,
			"userId", userID,
			"action", action.Type)
		return err
	}

	// 检查 Round 是否结束
	if round.GetStatus() == RoundStatusSettling {
		// Round 结束，进行结算
		if err := m.finishRound(ctx, game); err != nil {
			return err
		}
	}

	return nil
}

// BroadcastGameEvent 广播游戏事件给房间所有玩家
func (m *GameManager) BroadcastGameEvent(ctx context.Context, roomId string, event string, data interface{}) error {
	return m.roomBroadcaster.BroadcastToRoom(ctx, roomId, event, data)
}

// SendPersonalizedGameEvents 给每个玩家发送个性化的游戏事件（并行发送）
func (m *GameManager) SendPersonalizedGameEvents(ctx context.Context, roomId string, event string, userDataMap map[int64]interface{}) error {
	var wg sync.WaitGroup

	for userId, data := range userDataMap {
		wg.Add(1)
		go func(uid int64, d interface{}) {
			defer wg.Done()

			eventData, err := json.Marshal(d)
			if err != nil {
				m.logger.Error("Failed to marshal personalized event data",
					"error", err, "event", event, "userId", uid)
				return
			}

			if err := m.routerService.SendRoomPushToUsers(ctx, []int64{uid}, event, roomId, eventData); err != nil {
				m.logger.Warn("Failed to send personalized event",
					"error", err, "event", event, "userId", uid, "roomId", roomId)
			}
		}(userId, data)
	}

	wg.Wait()
	return nil
}

// GetGame 获取游戏
func (m *GameManager) GetGame(roomID string) (*MultiRoundGame, error) {
	game, ok := m.getGame(roomID)
	if !ok {
		return nil, ErrGameNotFound
	}
	return game, nil
}

// RemoveGame 移除游戏
func (m *GameManager) RemoveGame(roomID string) {
	m.removeGame(roomID)
	m.logger.Info("Game removed", "roomId", roomID)
}

// Count 返回当前游戏数
func (m *GameManager) Count() int {
	return int(m.gameCount.Load())
}

// Shutdown 关闭管理器
func (m *GameManager) Shutdown(ctx context.Context) error {
	m.logger.Info("Shutting down GameManager")

	// 发送停止信号
	close(m.stopChan)

	// 停止定时器
	m.evictTicker.Stop()

	// 保存所有脏游戏并关闭
	m.games.Range(func(key, value interface{}) bool {
		g := value.(*MultiRoundGame)
		if g.IsDirty() {
			// TODO: 保存到数据库
			m.logger.Info("Saving game on shutdown", "roomId", g.RoomID)
		}
		g.Close()
		return true
	})

	m.logger.Info("GameManager shutdown complete")
	return nil
}

// ========== 私有方法 ==========

// createGame 创建游戏
func (m *GameManager) createGame(roomID string, gameType string, config GameConfig) (*MultiRoundGame, error) {
	// 检查是否已存在
	if _, ok := m.games.Load(roomID); ok {
		return nil, ErrGameAlreadyExists
	}

	// 检查容量
	if m.maxGames > 0 && m.gameCount.Load() >= int64(m.maxGames) {
		m.logger.Warn("Max games limit reached",
			"maxGames", m.maxGames,
			"current", m.gameCount.Load())
		return nil, ErrMaxGamesReached
	}

	// 创建游戏（使用雪花算法生成唯一ID）
	gameID := m.sfNode.Generate().String()
	g := NewMultiRoundGame(gameID, roomID, gameType, config, m.scheduler)

	actual, loaded := m.games.LoadOrStore(roomID, g)
	if !loaded {
		m.gameCount.Add(1)
		m.logger.Info("Created game", "roomId", roomID, "gameId", gameID)
	}

	return actual.(*MultiRoundGame), nil
}

// getGame 获取游戏
func (m *GameManager) getGame(roomID string) (*MultiRoundGame, bool) {
	val, ok := m.games.Load(roomID)
	if !ok {
		return nil, false
	}
	return val.(*MultiRoundGame), true
}

// removeGame 移除游戏
func (m *GameManager) removeGame(roomID string) {
	if val, loaded := m.games.LoadAndDelete(roomID); loaded {
		g := val.(*MultiRoundGame)
		g.Close()
		m.gameCount.Add(-1)
		m.logger.Info("Removed game", "roomId", roomID)
	}
}

// createEngine 根据游戏类型创建引擎
func (m *GameManager) createEngine(gameType string) RoundEngine {
	switch gameType {
	case "HT_MAHJONG":
		return htmajong.NewEngine()
	case "TH_MAHJONG":
		return thmahjong.NewEngine()
	default:
		m.logger.Warn("Unknown game type, using default", "gameType", gameType)
		return htmajong.NewEngine()
	}
}

// finishRound 完成当前局
func (m *GameManager) finishRound(ctx context.Context, game *MultiRoundGame) error {
	if err := game.FinishCurrentRound(ctx); err != nil {
		m.logger.Error("Failed to finish round", "error", err, "gameId", game.ID)
		return err
	}

	m.logger.Info("Round finished", "gameId", game.ID)

	// 检查游戏是否结束
	if game.GetStatus() == MultiRoundGameStatusFinished {
		m.logger.Info("Game finished", "gameId", game.ID, "finalScores", game.GetPlayerScores())
		// TODO: 触发游戏结束事件
	}

	return nil
}

// broadcastGameStart 广播游戏开始事件
func (m *GameManager) broadcastGameStart(ctx context.Context, room *model.Room) error {
	gameStartData := map[string]interface{}{
		"roomId":   room.RoomID,
		"gameType": room.GameType,
		"status":   "playing",
		"players":  room.Players,
	}

	if err := m.BroadcastGameEvent(ctx, room.RoomID, "GAME_STARTED", gameStartData); err != nil {
		return fmt.Errorf("failed to broadcast game start event: %w", err)
	}

	m.logger.Info("Game start event broadcasted", "roomId", room.RoomID)
	return nil
}

// evictLoop 淘汰循环
func (m *GameManager) evictLoop() {
	for {
		select {
		case <-m.evictTicker.C:
			m.evictInactive()
		case <-m.stopChan:
			m.logger.Info("Evict loop stopped")
			return
		}
	}
}

// evictInactive 淘汰不活跃的游戏
func (m *GameManager) evictInactive() {
	now := time.Now()
	var toEvict []string

	m.games.Range(func(key, value interface{}) bool {
		roomID := key.(string)
		g := value.(*MultiRoundGame)

		if now.Sub(g.LastActiveTime()) > m.evictTimeout {
			toEvict = append(toEvict, roomID)
		}

		return true
	})

	for _, roomID := range toEvict {
		if val, ok := m.games.Load(roomID); ok {
			g := val.(*MultiRoundGame)

			// 重新检查活跃时间
			if now.Sub(g.LastActiveTime()) <= m.evictTimeout {
				continue
			}

			if g.IsDirty() {
				// TODO: 保存到数据库
				m.logger.Info("Saving game before eviction", "roomId", roomID)
			}

			m.removeGame(roomID)
			m.logger.Info("Evicted inactive game", "roomId", roomID)
		}
	}
}

// extractPlayerIDs 提取玩家 ID 列表
func extractPlayerIDs(players []model.RoomPlayer) []int64 {
	ids := make([]int64, len(players))
	for i, p := range players {
		ids[i] = p.UserID
	}
	return ids
}
