package game

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// GameManager 游戏管理器
type GameManager struct {
	games     sync.Map     // roomID -> *Game
	gameCount atomic.Int64 // 原子计数器

	// LRU 配置
	maxGames     int
	evictTimeout time.Duration
	evictTicker  *time.Ticker

	stopChan chan struct{} // 停止信号通道

	logger *slog.Logger
}

// NewGameManager 创建游戏管理器
func NewGameManager(maxGames int, evictTimeout time.Duration) *GameManager {
	m := &GameManager{
		maxGames:     maxGames,
		evictTimeout: evictTimeout,
		evictTicker:  time.NewTicker(60 * time.Second),
		stopChan:     make(chan struct{}),
		logger:       slog.Default().With("component", "GameManager"),
	}

	go m.evictLoop()

	return m
}

// GetOrCreate 获取或创建游戏
func (m *GameManager) GetOrCreate(roomID string, gameType string) (*Game, error) {
	// 快路径：已存在则直接返回
	if val, ok := m.games.Load(roomID); ok {
		return val.(*Game), nil
	}

	// 慢路径：需要创建，先检查容量
	if m.maxGames > 0 && m.gameCount.Load() >= int64(m.maxGames) {
		m.logger.Warn("Max games limit reached",
			"maxGames", m.maxGames,
			"current", m.gameCount.Load())
		return nil, ErrMaxGamesReached
	}

	g := NewGame(roomID, gameType)
	actual, loaded := m.games.LoadOrStore(roomID, g)
	if !loaded {
		// 新创建的，递增计数器
		m.gameCount.Add(1)
	}
	return actual.(*Game), nil
}

// Get 获取游戏
func (m *GameManager) Get(roomID string) (*Game, bool) {
	val, ok := m.games.Load(roomID)
	if !ok {
		return nil, false
	}
	return val.(*Game), true
}

// Remove 移除游戏
func (m *GameManager) Remove(roomID string) {
	if val, loaded := m.games.LoadAndDelete(roomID); loaded {
		g := val.(*Game)
		g.Close()
		m.gameCount.Add(-1)
		m.logger.Info("Removed game", "roomId", roomID)
	}
}

// Count 返回当前游戏数
func (m *GameManager) Count() int {
	return int(m.gameCount.Load())
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
		g := value.(*Game)

		if now.Sub(g.LastActiveTime()) > m.evictTimeout {
			toEvict = append(toEvict, roomID)
		}

		return true
	})

	for _, roomID := range toEvict {
		if val, ok := m.games.Load(roomID); ok {
			g := val.(*Game)

			// 重新检查活跃时间，避免误淘汰刚变活跃的游戏
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
func (m *GameManager) Shutdown(ctx context.Context) error {
	m.logger.Info("Shutting down GameManager")

	// 发送停止信号给 evictLoop
	close(m.stopChan)

	// 停止定时器
	m.evictTicker.Stop()

	// 保存所有脏游戏并关闭
	m.games.Range(func(key, value interface{}) bool {
		g := value.(*Game)
		if g.IsDirty() {
			// TODO: 保存到数据库
			m.logger.Info("Saving game on shutdown", "roomId", g.roomID)
		}
		g.Close()
		return true
	})

	m.logger.Info("GameManager shutdown complete")
	return nil
}
