package mahjong

import (
	"context"
	"strconv"
	"sync"
	"time"

	"sudooom.im.logic/internal/game/mahjong/core"
	"sudooom.im.logic/internal/game/types"
)

// SafeMahjongEngine 线程安全的麻将引擎包装
// 实现 game.RoundEngine 接口
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

// Initialize 初始化（转换参数，实现 game.RoundEngine）
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

// HandleAction 处理动作（实现 game.RoundEngine）
// 使用 playerID 参数覆盖 action.PlayerID，确保玩家身份一致
func (e *SafeMahjongEngine) HandleAction(ctx context.Context, playerID string, action core.Action) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	action.PlayerID = playerID
	return e.engine.HandleAction(ctx, action)
}

// GetState 获取状态（实现 game.RoundEngine）
func (e *SafeMahjongEngine) GetState() *core.GameState {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.engine.GetState()
}

// IsRoundOver 检查单局是否结束（实现 game.RoundEngine）
func (e *SafeMahjongEngine) IsRoundOver() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.engine.IsGameOver()
}

// ProcessTaskTimeout 处理任务超时（实现 game.RoundEngine）
func (e *SafeMahjongEngine) ProcessTaskTimeout() {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 通过接口断言访问 core.Engine 的方法
	type taskProcessor interface {
		ProcessTaskTimeout()
	}

	proc, ok := e.engine.(taskProcessor)
	if !ok {
		return
	}

	proc.ProcessTaskTimeout()
}

// GetSettlement 获取结算结果（实现 game.RoundEngine）
// 返回 interface{} 避免循环依赖
func (e *SafeMahjongEngine) GetSettlement() interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	coreSettlement := e.engine.GetSettlement()
	if coreSettlement == nil {
		return nil
	}

	// 转换为 types.RoundSettlement
	settlement := &types.RoundSettlement{
		SettleTime: time.Now(),
	}

	// 转换结算类型
	switch coreSettlement.WinType {
	case core.WinTypeDraw:
		settlement.SettlementType = "win"
		settlement.WinType = "self_draw"
	case core.WinTypeDiscard:
		settlement.SettlementType = "win"
		settlement.WinType = "discard"
	case core.WinTypeExhaust:
		settlement.SettlementType = "draw"
		settlement.WinType = ""
	default:
		settlement.SettlementType = "win"
		settlement.WinType = ""
	}

	// 转换番型
	settlement.FanType = make([]string, len(coreSettlement.Patterns))
	totalFan := 0
	for i, pattern := range coreSettlement.Patterns {
		settlement.FanType[i] = pattern.Name
		totalFan += pattern.Score
	}
	settlement.FanScore = totalFan

	// 获取赢家手牌和胡牌
	state := e.engine.GetState()
	if coreSettlement.WinnerID != "" {
		for _, player := range state.Players {
			if player.ID == coreSettlement.WinnerID {
				settlement.WinnerHand = append([]core.Tile{}, player.Hand...)
				break
			}
		}
		if state.LastAction != nil && state.LastAction.Tile != nil {
			settlement.WinTile = state.LastAction.Tile
		}
	}

	// 转换分数变化
	// 使用 Transfers 来构建 Winners 和 Losers
	scoreChanges := make(map[string]int) // playerID -> 分数变化

	for _, transfer := range coreSettlement.Transfers {
		scoreChanges[transfer.FromID] -= transfer.Amount
		scoreChanges[transfer.ToID] += transfer.Amount
	}

	// 构建 Winners 和 Losers
	for playerID, change := range scoreChanges {
		playerIDInt, err := strconv.ParseInt(playerID, 10, 64)
		if err != nil {
			continue
		}

		playerScore := types.PlayerScore{
			PlayerID:    playerIDInt,
			ScoreChange: change,
		}

		if change > 0 {
			playerScore.Role = "winner"
			settlement.Winners = append(settlement.Winners, playerScore)
		} else if change < 0 {
			playerScore.Role = "loser"
			settlement.Losers = append(settlement.Losers, playerScore)
		}
	}

	return settlement
}

// GetGameType 获取游戏类型
func (e *SafeMahjongEngine) GetGameType() string {
	return e.gameType
}

// GetCoreSettlement 获取 core.Settlement（用于兼容旧代码）
func (e *SafeMahjongEngine) GetCoreSettlement() *core.Settlement {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.engine.GetSettlement()
}
