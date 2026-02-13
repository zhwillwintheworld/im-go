package game

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"sudooom.im.logic/internal/task"
)

// MultiRoundGameManager 多轮游戏管理器（管理 MultiRoundGame）
type MultiRoundGameManager struct {
	games     sync.Map     // roomID -> *MultiRoundGame
	gameCount atomic.Int64 // 原子计数器

	// LRU 配置
	maxGames     int
	evictTimeout time.Duration
	evictTicker  *time.Ticker

	stopChan chan struct{} // 停止信号通道

	scheduler *task.Scheduler // 任务调度器

	logger *slog.Logger
}

// NewMultiRoundGameManager 创建多轮游戏管理器
func NewMultiRoundGameManager(maxGames int, evictTimeout time.Duration, scheduler *task.Scheduler) *MultiRoundGameManager {
	m := &MultiRoundGameManager{
		maxGames:     maxGames,
		evictTimeout: evictTimeout,
		evictTicker:  time.NewTicker(60 * time.Second),
		stopChan:     make(chan struct{}),
		scheduler:    scheduler,
		logger:       slog.Default().With("component", "MultiRoundGameManager"),
	}

	go m.evictLoop()

	return m
}

// CreateGame 创建游戏
func (m *MultiRoundGameManager) CreateGame(roomID string, gameType string, config GameConfig) (*MultiRoundGame, error) {
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

	// 创建游戏
	gameID := roomID + ":game:" + time.Now().Format("20060102150405")
	g := NewMultiRoundGame(gameID, roomID, gameType, config, m.scheduler)

	actual, loaded := m.games.LoadOrStore(roomID, g)
	if !loaded {
		m.gameCount.Add(1)
		m.logger.Info("Created game", "roomId", roomID, "gameId", gameID)
	}

	return actual.(*MultiRoundGame), nil
}

// Get 获取游戏
func (m *MultiRoundGameManager) Get(roomID string) (*MultiRoundGame, bool) {
	val, ok := m.games.Load(roomID)
	if !ok {
		return nil, false
	}
	return val.(*MultiRoundGame), true
}

// Remove 移除游戏
func (m *MultiRoundGameManager) Remove(roomID string) {
	if val, loaded := m.games.LoadAndDelete(roomID); loaded {
		g := val.(*MultiRoundGame)
		g.Close()
		m.gameCount.Add(-1)
		m.logger.Info("Removed game", "roomId", roomID)
	}
}

// Count 返回当前游戏数
func (m *MultiRoundGameManager) Count() int {
	return int(m.gameCount.Load())
}

// evictLoop 淘汰循环
func (m *MultiRoundGameManager) evictLoop() {
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
func (m *MultiRoundGameManager) evictInactive() {
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

			m.Remove(roomID)
			m.logger.Info("Evicted inactive game", "roomId", roomID)
		}
	}
}

// Shutdown 关闭管理器
func (m *MultiRoundGameManager) Shutdown(ctx context.Context) error {
	m.logger.Info("Shutting down MultiRoundGameManager")

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

	m.logger.Info("MultiRoundGameManager shutdown complete")
	return nil
}
