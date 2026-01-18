package htmajong

import (
	"context"

	"sudooom.im.logic/internal/game/mahjong/core"
)

// Engine 会同麻将游戏引擎
type Engine struct {
	*core.Engine
}

// NewEngine 创建会同麻将游戏引擎
func NewEngine() *Engine {
	deckGen := NewDeckGenerator()
	actionHandler := NewActionHandler()
	winningAlgo := NewWinningAlgorithm()
	taskJudge := NewTaskJudge(winningAlgo)
	settler := NewSettler()

	coreEngine := core.NewEngine(
		deckGen,
		actionHandler,
		taskJudge,
		winningAlgo,
		settler,
	)

	return &Engine{
		Engine: coreEngine,
	}
}

// Initialize 初始化游戏 (重写以添加会同麻将特定的初始化逻辑)
func (e *Engine) Initialize(ctx context.Context, playerIDs []string, config core.GameConfig) error {
	// 调用父类初始化
	if err := e.Engine.Initialize(ctx, playerIDs, config); err != nil {
		return err
	}

	// 为每个玩家初始化会同麻将特定状态
	state := e.GetState()
	for _, player := range state.Players {
		player.State = &HTPlayerState{
			IsTing:       false,
			CanTingRound: 0,
		}
	}

	return nil
}
