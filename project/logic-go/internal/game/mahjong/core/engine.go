package core

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Engine 通用游戏引擎实现
type Engine struct {
	state         *GameState
	deckGenerator DeckGenerator
	actionHandler ActionHandler
	taskJudge     TaskJudge
	winningAlgo   WinningAlgorithm
	settler       Settler
	logger        *slog.Logger
	taskTimeout   time.Duration // 任务超时时间
}

// NewEngine 创建游戏引擎
func NewEngine(
	deckGen DeckGenerator,
	actionHandler ActionHandler,
	taskJudge TaskJudge,
	winningAlgo WinningAlgorithm,
	settler Settler,
) *Engine {
	return &Engine{
		deckGenerator: deckGen,
		actionHandler: actionHandler,
		taskJudge:     taskJudge,
		winningAlgo:   winningAlgo,
		settler:       settler,
		logger:        slog.Default(),
		taskTimeout:   30 * time.Second, // 默认30秒超时
	}
}

// Initialize 初始化游戏
func (e *Engine) Initialize(ctx context.Context, playerIDs []string, config GameConfig) error {
	e.logger.Info("初始化麻将游戏", "playerCount", len(playerIDs))

	// 验证玩家数量
	if len(playerIDs) != config.PlayerCount {
		return fmt.Errorf("玩家数量不匹配: 期望 %d, 实际 %d", config.PlayerCount, len(playerIDs))
	}

	// 生成牌堆
	deck := e.deckGenerator.GenerateDeck()
	e.logger.Info("生成牌堆", "tileCount", len(deck))

	// 洗牌
	e.deckGenerator.Shuffle(deck)

	// 创建玩家
	players := make([]*Player, len(playerIDs))
	for i, id := range playerIDs {
		players[i] = &Player{
			ID:       id,
			Hand:     []Tile{},
			Discards: []Tile{},
			Melds:    []Meld{},
			Score:    0,
		}
	}

	// 发牌
	dealerIndex := 0 // 默认第一个玩家是庄家
	hands, remaining := e.deckGenerator.Deal(deck, config.PlayerCount, dealerIndex)

	// 分配手牌
	for i, hand := range hands {
		players[i].Hand = hand
		SortTiles(players[i].Hand)
	}

	// 初始化游戏状态
	e.state = &GameState{
		Players:       players,
		Deck:          remaining,
		CurrentPlayer: dealerIndex,
		LastAction:    nil,
		Round:         1,
		DealerIndex:   dealerIndex,
		Config:        config,
		IsGameOver:    false,
		Settlement:    nil,
		PendingTasks:  []Task{},
	}

	e.logger.Info("游戏初始化成功",
		"deckRemaining", len(remaining),
		"dealer", playerIDs[dealerIndex])

	return nil
}

// HandleAction 处理玩家动作
func (e *Engine) HandleAction(ctx context.Context, action Action) error {
	e.logger.Info("处理玩家动作",
		"playerID", action.PlayerID,
		"actionType", action.Type.String())

	// 验证动作
	if err := e.actionHandler.ValidateAction(e.state, action); err != nil {
		return fmt.Errorf("动作验证失败: %w", err)
	}

	// 执行动作
	if err := e.actionHandler.ExecuteAction(e.state, action); err != nil {
		return fmt.Errorf("动作执行失败: %w", err)
	}

	// 检查是否胡牌
	if action.Type == ActionWin {
		e.handleWin(action)
		return nil
	}

	// 判断是否产生任务
	tasks := e.taskJudge.JudgeTasks(e.state, action)
	if len(tasks) > 0 {
		e.logger.Info("产生任务", "taskCount", len(tasks))
		e.state.PendingTasks = tasks
		// 设置任务超时时间
		timeout := time.Now().Add(e.taskTimeout)
		for i := range e.state.PendingTasks {
			e.state.PendingTasks[i].Timeout = timeout
		}
	} else {
		// 没有任务,切换到下一个玩家
		if action.Type == ActionDiscard || action.Type == ActionPass {
			e.state.NextPlayer()
		}
	}

	// 记录最后一个动作
	e.state.LastAction = &action

	return nil
}

// handleWin 处理胡牌
func (e *Engine) handleWin(action Action) {
	e.logger.Info("玩家胡牌", "playerID", action.PlayerID)

	player := e.state.GetPlayer(action.PlayerID)
	if player == nil {
		e.logger.Error("找不到玩家", "playerID", action.PlayerID)
		return
	}

	// 确定胡牌类型和输家
	var winType WinType
	var loserID string

	if e.state.LastAction != nil && e.state.LastAction.Type == ActionDiscard {
		// 点炮
		winType = WinTypeDiscard
		loserID = e.state.LastAction.PlayerID
	} else if e.state.LastAction != nil && e.state.LastAction.Type == ActionKong {
		// 抢杠
		winType = WinTypeQiangKong
		loserID = e.state.LastAction.PlayerID
	} else {
		// 自摸
		winType = WinTypeDraw
		loserID = ""
	}

	// 获取胡牌牌型
	patterns := e.winningAlgo.GetWinPatterns(player.Hand, action.Tile, e.state, action.PlayerID)

	// 计算结算
	settlement := e.settler.Calculate(e.state, action.PlayerID, loserID, winType, patterns)

	e.state.IsGameOver = true
	e.state.Settlement = settlement

	e.logger.Info("游戏结束",
		"winner", action.PlayerID,
		"winType", winType.String(),
		"totalScore", settlement.TotalScore)
}

// GetState 获取游戏状态
func (e *Engine) GetState() *GameState {
	return e.state
}

// IsGameOver 检查游戏是否结束
func (e *Engine) IsGameOver() bool {
	if e.state == nil {
		return false
	}
	return e.state.IsGameOver
}

// GetSettlement 获取结算结果
func (e *Engine) GetSettlement() *Settlement {
	if e.state == nil {
		return nil
	}
	return e.state.Settlement
}

// ClearPendingTasks 清除待处理任务
func (e *Engine) ClearPendingTasks() {
	e.state.PendingTasks = []Task{}
}

// ProcessTaskTimeout 处理任务超时
func (e *Engine) ProcessTaskTimeout() {
	now := time.Now()
	hasTimeout := false

	for _, task := range e.state.PendingTasks {
		if now.After(task.Timeout) {
			hasTimeout = true
			e.logger.Info("任务超时", "playerID", task.PlayerID)
			break
		}
	}

	if hasTimeout {
		// 清除所有任务,游戏继续
		e.ClearPendingTasks()
		e.state.NextPlayer()
	}
}
