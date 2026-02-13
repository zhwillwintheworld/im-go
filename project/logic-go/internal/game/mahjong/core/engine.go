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
		CurrentTaskID: "",
		TaskResponses: nil,
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
		"actionType", action.Type.String(),
		"taskID", action.TaskID)

	// 流局检查：摸牌请求但牌堆为空，触发流局结算
	if action.Type == ActionDraw && len(e.state.Deck) == 0 {
		e.handleExhaust()
		return nil
	}

	// 判断是否为任务响应
	if action.IsTaskResponse() {
		return e.handleTaskResponse(ctx, action)
	}

	// 主动动作处理
	return e.handleActiveAction(ctx, action)
}

// handleActiveAction 处理主动动作（非任务响应）
func (e *Engine) handleActiveAction(ctx context.Context, action Action) error {
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

		// 生成任务ID
		taskID := fmt.Sprintf("task:%d", time.Now().UnixNano())
		for i := range tasks {
			tasks[i].ID = taskID
			tasks[i].Timeout = time.Now().Add(e.taskTimeout)
		}

		e.state.PendingTasks = tasks
		e.state.CurrentTaskID = taskID
		e.state.TaskResponses = &TaskResponse{
			TaskID:    taskID,
			Responses: make(map[string]Action),
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

// handleTaskResponse 处理任务响应
func (e *Engine) handleTaskResponse(ctx context.Context, action Action) error {
	// 验证任务ID
	if action.TaskID != e.state.CurrentTaskID {
		return fmt.Errorf("任务ID不匹配: 期望 %s, 实际 %s", e.state.CurrentTaskID, action.TaskID)
	}

	// 验证任务是否存在
	task := e.state.GetTask(action.TaskID)
	if task == nil {
		return fmt.Errorf("任务不存在: %s", action.TaskID)
	}

	// 验证玩家是否有权响应此任务
	hasPermission := false
	for _, t := range e.state.PendingTasks {
		if t.PlayerID == action.PlayerID {
			hasPermission = true
			break
		}
	}
	if !hasPermission {
		return fmt.Errorf("玩家 %s 无权响应任务 %s", action.PlayerID, action.TaskID)
	}

	// 收集响应
	if e.state.TaskResponses == nil {
		e.state.TaskResponses = &TaskResponse{
			TaskID:    action.TaskID,
			Responses: make(map[string]Action),
		}
	}
	e.state.TaskResponses.AddResponse(action)

	e.logger.Info("收集任务响应",
		"taskID", action.TaskID,
		"playerID", action.PlayerID,
		"actionType", action.Type.String(),
		"responseCount", len(e.state.TaskResponses.Responses),
		"totalTasks", len(e.state.PendingTasks))

	// 检查是否所有玩家都已响应
	if len(e.state.TaskResponses.Responses) >= len(e.state.PendingTasks) {
		return e.processTaskResponses(ctx)
	}

	return nil
}

// processTaskResponses 处理所有任务响应
func (e *Engine) processTaskResponses(ctx context.Context) error {
	if e.state.TaskResponses == nil {
		return fmt.Errorf("没有任务响应")
	}

	// 获取最高优先级的动作
	highestAction := e.state.TaskResponses.GetHighestPriorityAction()
	if highestAction == nil {
		// 所有人都过，清除任务，切换到下一个玩家
		e.state.ClearPendingTasks()
		e.state.NextPlayer()
		return nil
	}

	e.logger.Info("执行最高优先级动作",
		"playerID", highestAction.PlayerID,
		"actionType", highestAction.Type.String())

	// 清除任务
	e.state.ClearPendingTasks()

	// 验证并执行最高优先级动作
	if err := e.actionHandler.ValidateAction(e.state, *highestAction); err != nil {
		return fmt.Errorf("动作验证失败: %w", err)
	}

	if err := e.actionHandler.ExecuteAction(e.state, *highestAction); err != nil {
		return fmt.Errorf("动作执行失败: %w", err)
	}

	// 检查是否胡牌
	if highestAction.Type == ActionWin || highestAction.Type == ActionQiangKong {
		e.handleWin(*highestAction)
		return nil
	}

	// 记录最后一个动作
	e.state.LastAction = highestAction

	// 切换到执行动作的玩家
	playerIndex := e.state.GetPlayerIndex(highestAction.PlayerID)
	if playerIndex >= 0 {
		e.state.CurrentPlayer = playerIndex
	}

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

// handleExhaust 处理流局（牌堆耗尽无人胡牌）
func (e *Engine) handleExhaust() {
	e.logger.Info("流局：牌堆耗尽")

	settlement := &Settlement{
		WinnerID:   "",
		LoserID:    "",
		WinType:    WinTypeExhaust,
		Patterns:   []WinPattern{},
		BaseScore:  0,
		TotalScore: 0,
		Transfers:  []Transfer{},
	}

	e.state.IsGameOver = true
	e.state.Settlement = settlement
}

// HasPendingTasks 检查是否有待处理任务
func (e *Engine) HasPendingTasks() bool {
	if e.state == nil {
		return false
	}
	return e.state.HasPendingTasks()
}

// GetTaskTimeout 获取任务超时时间
func (e *Engine) GetTaskTimeout() time.Duration {
	return e.taskTimeout
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

// ProcessTaskTimeout 处理任务超时
func (e *Engine) ProcessTaskTimeout() {
	if !e.HasPendingTasks() {
		return
	}

	now := time.Now()
	hasTimeout := false

	for _, task := range e.state.PendingTasks {
		if now.After(task.Timeout) {
			hasTimeout = true
			e.logger.Info("任务超时", "taskID", task.ID, "playerID", task.PlayerID)
			break
		}
	}

	if hasTimeout {
		e.logger.Info("任务超时，处理已收集的响应",
			"taskID", e.state.CurrentTaskID,
			"responseCount", len(e.state.TaskResponses.Responses),
			"totalTasks", len(e.state.PendingTasks))

		// 处理已收集的响应（可能部分玩家未响应）
		if e.state.TaskResponses != nil && len(e.state.TaskResponses.Responses) > 0 {
			// 获取最高优先级的动作
			highestAction := e.state.TaskResponses.GetHighestPriorityAction()
			if highestAction != nil {
				e.logger.Info("执行最高优先级动作（超时）",
					"playerID", highestAction.PlayerID,
					"actionType", highestAction.Type.String())

				// 清除任务
				e.state.ClearPendingTasks()

				// 执行动作
				if err := e.actionHandler.ValidateAction(e.state, *highestAction); err == nil {
					if err := e.actionHandler.ExecuteAction(e.state, *highestAction); err == nil {
						// 检查是否胡牌
						if highestAction.Type == ActionWin || highestAction.Type == ActionQiangKong {
							e.handleWin(*highestAction)
							return
						}

						// 记录最后一个动作
						e.state.LastAction = highestAction

						// 切换到执行动作的玩家
						playerIndex := e.state.GetPlayerIndex(highestAction.PlayerID)
						if playerIndex >= 0 {
							e.state.CurrentPlayer = playerIndex
						}
						return
					}
				}
			}
		}

		// 没有有效响应，清除所有任务，游戏继续
		e.state.ClearPendingTasks()
		e.state.NextPlayer()
	}
}
