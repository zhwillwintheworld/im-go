package thmahjong

import (
	"time"

	"sudooom.im.logic/internal/game/mahjong/core"
)

// TaskJudge 太湖麻将任务判断器
type TaskJudge struct {
	winningAlgo *WinningAlgorithm
}

// NewTaskJudge 创建任务判断器
func NewTaskJudge(winningAlgo *WinningAlgorithm) *TaskJudge {
	return &TaskJudge{
		winningAlgo: winningAlgo,
	}
}

// JudgeTasks 判断是否有任务产生
func (j *TaskJudge) JudgeTasks(state *core.GameState, action core.Action) []core.Task {
	tasks := []core.Task{}

	if action.Type != core.ActionDiscard && action.Type != core.ActionKong {
		return tasks
	}

	var targetTile *core.Tile
	if action.Tile != nil {
		targetTile = action.Tile
	}

	actionPlayerIndex := state.GetPlayerIndex(action.PlayerID)

	// 遍历其他玩家
	for i, player := range state.Players {
		if player.ID == action.PlayerID {
			continue
		}

		availableActions := []core.ActionType{}

		// 检查是否可以胡牌
		if targetTile != nil && j.winningAlgo.CanWin(player.Hand, targetTile, state, player.ID) {
			availableActions = append(availableActions, core.ActionWin)
		}

		// 检查是否可以杠牌
		if action.Type == core.ActionDiscard && targetTile != nil {
			count := core.CountTile(player.Hand, *targetTile)
			if count >= 3 {
				availableActions = append(availableActions, core.ActionKong)
			}
		}

		// 检查是否可以碰牌
		if action.Type == core.ActionDiscard && targetTile != nil {
			count := core.CountTile(player.Hand, *targetTile)
			if count >= 2 {
				availableActions = append(availableActions, core.ActionPong)
			}
		}

		// 检查是否可以吃牌 (只能吃上家的牌)
		if action.Type == core.ActionDiscard && targetTile != nil {
			if (actionPlayerIndex+1)%len(state.Players) == i {
				if j.canChi(player.Hand, *targetTile) {
					availableActions = append(availableActions, core.ActionChi)
				}
			}
		}

		if len(availableActions) > 0 {
			task := core.Task{
				PlayerID:       player.ID,
				AvailableTypes: availableActions,
				RelatedTile:    targetTile,
				Priority:       j.GetTaskPriority(core.Task{AvailableTypes: availableActions}),
				Timeout:        time.Now().Add(30 * time.Second),
			}
			tasks = append(tasks, task)
		}
	}

	return tasks
}

// canChi 检查是否可以吃牌
func (j *TaskJudge) canChi(hand []core.Tile, targetTile core.Tile) bool {
	// 风、箭牌不能吃
	if targetTile.Suit >= core.TileSuitWind {
		return false
	}

	// 检查是否可以组成顺子
	// 情况1: targetTile-2, targetTile-1, targetTile
	if targetTile.Value >= 3 {
		tile1 := core.Tile{Suit: targetTile.Suit, Value: targetTile.Value - 2}
		tile2 := core.Tile{Suit: targetTile.Suit, Value: targetTile.Value - 1}
		if core.ContainsTile(hand, tile1) && core.ContainsTile(hand, tile2) {
			return true
		}
	}

	// 情况2: targetTile-1, targetTile, targetTile+1
	if targetTile.Value >= 2 && targetTile.Value <= 8 {
		tile1 := core.Tile{Suit: targetTile.Suit, Value: targetTile.Value - 1}
		tile2 := core.Tile{Suit: targetTile.Suit, Value: targetTile.Value + 1}
		if core.ContainsTile(hand, tile1) && core.ContainsTile(hand, tile2) {
			return true
		}
	}

	// 情况3: targetTile, targetTile+1, targetTile+2
	if targetTile.Value <= 7 {
		tile1 := core.Tile{Suit: targetTile.Suit, Value: targetTile.Value + 1}
		tile2 := core.Tile{Suit: targetTile.Suit, Value: targetTile.Value + 2}
		if core.ContainsTile(hand, tile1) && core.ContainsTile(hand, tile2) {
			return true
		}
	}

	return false
}

// GetTaskPriority 获取任务优先级 (胡>杠>碰>吃)
func (j *TaskJudge) GetTaskPriority(task core.Task) int {
	priority := 0

	for _, actionType := range task.AvailableTypes {
		switch actionType {
		case core.ActionWin:
			priority = max(priority, 100)
		case core.ActionKong:
			priority = max(priority, 80)
		case core.ActionPong:
			priority = max(priority, 70)
		case core.ActionChi:
			priority = max(priority, 60)
		}
	}

	return priority
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
