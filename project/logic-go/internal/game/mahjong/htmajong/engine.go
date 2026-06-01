package htmajong

import (
	"context"
	"strconv"
	"time"

	"sudooom.im.logic/internal/game/mahjong/core"
	"sudooom.im.logic/internal/game/types"
)

// Engine 会同麻将游戏引擎
// 直接实现 game.RoundEngine 接口，不需要适配器
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

// Initialize 初始化游戏（实现 game.RoundEngine 接口）
func (e *Engine) Initialize(ctx context.Context, playerIDs []string) error {
	config := core.GameConfig{
		PlayerCount: len(playerIDs),
		BaseScore:   10,
		Extra:       make(map[string]any),
	}

	// 调用 core.Engine 初始化
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

// HandleAction 处理动作（实现 game.RoundEngine 接口）
func (e *Engine) HandleAction(ctx context.Context, playerID string, action core.Action) error {
	action.PlayerID = playerID
	return e.Engine.HandleAction(ctx, action)
}

// IsRoundOver 检查单局是否结束（实现 game.RoundEngine 接口）
func (e *Engine) IsRoundOver() bool {
	return e.IsGameOver()
}

// GetSettlement 获取结算结果（实现 game.RoundEngine 接口）
func (e *Engine) GetSettlement() interface{} {
	coreSettlement := e.Engine.GetSettlement()
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
	state := e.GetState()
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

// ProcessTaskTimeout 处理任务超时（实现 game.RoundEngine 接口）
func (e *Engine) ProcessTaskTimeout() {
	e.Engine.ProcessTaskTimeout()
}
