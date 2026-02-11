package mahjong

import (
	"context"
	"sync"

	"sudooom.im.logic/internal/game/mahjong/core"
)

// SafeMahjongEngine 线程安全的麻将引擎包装
// 实现 game.GameEngine 接口
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

// Initialize 初始化（转换参数，实现 game.GameEngine）
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

// HandleAction 处理动作（实现 game.GameEngine）
// 使用 playerID 参数覆盖 action.PlayerID，确保玩家身份一致
func (e *SafeMahjongEngine) HandleAction(ctx context.Context, playerID string, action core.Action) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	action.PlayerID = playerID
	return e.engine.HandleAction(ctx, action)
}

// GetState 获取状态（实现 game.GameEngine）
func (e *SafeMahjongEngine) GetState() *core.GameState {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.engine.GetState()
}

// IsGameOver 检查游戏是否结束（实现 game.GameEngine）
func (e *SafeMahjongEngine) IsGameOver() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.engine.IsGameOver()
}

// CheckAndProcessTaskTimeout 检查并处理任务超时（实现 game.GameEngine）
// 返回 true 表示有任务超时并被处理
func (e *SafeMahjongEngine) CheckAndProcessTaskTimeout() bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 通过接口断言访问 core.Engine 的方法
	type taskProcessor interface {
		HasPendingTasks() bool
		ProcessTaskTimeout()
	}

	proc, ok := e.engine.(taskProcessor)
	if !ok {
		return false
	}

	if !proc.HasPendingTasks() {
		return false
	}

	proc.ProcessTaskTimeout()
	return true
}

// GetGameType 获取游戏类型
func (e *SafeMahjongEngine) GetGameType() string {
	return e.gameType
}

// GetSettlement 获取结算结果
func (e *SafeMahjongEngine) GetSettlement() *core.Settlement {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.engine.GetSettlement()
}
