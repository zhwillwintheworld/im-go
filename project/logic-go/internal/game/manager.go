package game

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// GameManager 游戏管理器
type GameManager struct {
	games sync.Map // gameId -> *Game

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
func (m *GameManager) GetOrCreate(roomID string, gameType string) *Game {
	gameId := roomID // 一个房间一个游戏

	if val, ok := m.games.Load(gameId); ok {
		return val.(*Game)
	}

	game := NewGame(roomID, gameType)
	actual, _ := m.games.LoadOrStore(gameId, game)
	return actual.(*Game)
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
	m.games.Delete(roomID)
	m.logger.Info("Removed game", "roomId", roomID)
}

// Count 返回当前游戏数
func (m *GameManager) Count() int {
	count := 0
	m.games.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
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
	toEvict := []string{}

	m.games.Range(func(key, value interface{}) bool {
		gameId := key.(string)
		game := value.(*Game)

		if now.Sub(game.LastActiveTime()) > m.evictTimeout {
			toEvict = append(toEvict, gameId)
		}

		return true
	})

	for _, gameId := range toEvict {
		if val, ok := m.games.Load(gameId); ok {
			game := val.(*Game)

			if game.IsDirty() {
				// TODO: 保存到数据库
				m.logger.Info("Saving game before eviction", "gameId", gameId)
			}

			m.Remove(gameId)
			m.logger.Info("Evicted inactive game", "gameId", gameId)
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

	// 保存所有脏游戏
	m.games.Range(func(key, value interface{}) bool {
		game := value.(*Game)
		if game.IsDirty() {
			// TODO: 保存到数据库
			m.logger.Info("Saving game on shutdown", "gameId", game.roomID)
		}
		return true
	})

	m.logger.Info("GameManager shutdown complete")
	return nil
}
