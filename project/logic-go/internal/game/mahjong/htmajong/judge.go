package htmajong

import (
	"time"

	"sudooom.im.logic/internal/game/mahjong/core"
)

// TaskJudge 会同麻将任务判断器
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

	// 只有出牌和杠牌才会产生任务
	if action.Type != core.ActionDiscard && action.Type != core.ActionKong {
		return tasks
	}

	var targetTile *core.Tile
	if action.Tile != nil {
		targetTile = action.Tile
	}

	// 遍历其他玩家,检查是否可以碰/杠/胡
	for _, player := range state.Players {
		if player.ID == action.PlayerID {
			continue // 跳过出牌者自己
		}

		availableActions := []core.ActionType{}

		// 检查是否可以胡牌
		if targetTile != nil && j.winningAlgo.CanWin(player.Hand, targetTile, state, player.ID) {
			availableActions = append(availableActions, core.ActionWin)
		}

		// 检查是否可以杠牌 (针对出牌动作)
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

		// 检查是否可以抢杠
		if action.Type == core.ActionKong && targetTile != nil {
			if j.winningAlgo.CanWin(player.Hand, targetTile, state, player.ID) {
				availableActions = append(availableActions, core.ActionQiangKong)
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

// GetTaskPriority 获取任务优先级 (胡>抢杠>杠>碰)
func (j *TaskJudge) GetTaskPriority(task core.Task) int {
	priority := 0

	for _, actionType := range task.AvailableTypes {
		switch actionType {
		case core.ActionWin:
			priority = max(priority, 100)
		case core.ActionQiangKong:
			priority = max(priority, 90)
		case core.ActionKong:
			priority = max(priority, 80)
		case core.ActionPong:
			priority = max(priority, 70)
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
