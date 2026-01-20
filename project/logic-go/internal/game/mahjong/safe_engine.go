package mahjong

import (
	"context"
	"fmt"
	"sync"

	"sudooom.im.logic/internal/game/mahjong/core"
)

// SafeMahjongEngine 线程安全的麻将引擎包装
// 在 mahjong 包内实现，避免循环依赖
type SafeMahjongEngine struct {
	mu       sync.RWMutex
	engine   core.GameEngine // mahjong/core 的引擎
	gameType string
}

// NewSafeMahjongEngine 创建线程安全的麻将引擎
func NewSafeMahjongEngine(engine core.GameEngine, gameType string) *SafeMahjongEngine {
	return &SafeMahjongEngine{
		engine:   engine,
		gameType: gameType,
	}
}

// Initialize 初始化（转换参数）
func (e *SafeMahjongEngine) Initialize(ctx context.Context, playerIDs []string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	config := core.GameConfig{
		PlayerCount: len(playerIDs),
		BaseScore:   10,
		Extra:       make(map[string]any),
	}

	return e.engine.Initialize(ctx, playerIDs, config)
}

// HandleAction 处理动作（带锁）
func (e *SafeMahjongEngine) HandleAction(ctx context.Context, playerID string, action interface{}) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 将 action 转换为 core.Action
	coreAction, ok := action.(core.Action)
	if !ok {
		return fmt.Errorf("invalid action type, expected core.Action")
	}

	return e.engine.HandleAction(ctx, coreAction)
}

// GetState 获取状态（带锁）
func (e *SafeMahjongEngine) GetState() interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.engine.GetState()
}

// IsGameOver 检查游戏是否结束
func (e *SafeMahjongEngine) IsGameOver() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.engine.IsGameOver()
}

// GetGameType 获取游戏类型
func (e *SafeMahjongEngine) GetGameType() string {
	return e.gameType
}

// GetMahjongEngine 获取底层麻将引擎（仅供内部使用，需要额外锁保护）
func (e *SafeMahjongEngine) GetMahjongEngine() core.GameEngine {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.engine
}

// GetMahjongState 获取麻将游戏状态（类型安全的访问）
func (e *SafeMahjongEngine) GetMahjongState() *core.GameState {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.engine.GetState()
}

// GetSettlement 获取结算结果
func (e *SafeMahjongEngine) GetSettlement() *core.Settlement {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.engine.GetSettlement()
}
