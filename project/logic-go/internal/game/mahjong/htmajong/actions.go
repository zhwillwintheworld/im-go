package htmajong

import (
	"fmt"

	"sudooom.im.logic/internal/game/mahjong/core"
)

// ActionHandler 会同麻将动作处理器
type ActionHandler struct{}

// NewActionHandler 创建动作处理器
func NewActionHandler() *ActionHandler {
	return &ActionHandler{}
}

// ValidateAction 验证动作是否合法
func (h *ActionHandler) ValidateAction(state *core.GameState, action core.Action) error {
	player := state.GetPlayer(action.PlayerID)
	if player == nil {
		return fmt.Errorf("玩家不存在: %s", action.PlayerID)
	}

	switch action.Type {
	case core.ActionDraw:
		return h.validateDraw(state, player)
	case core.ActionDiscard:
		return h.validateDiscard(state, player, action)
	case core.ActionPong:
		return h.validatePong(state, player, action)
	case core.ActionKong:
		return h.validateKong(state, player, action)
	case core.ActionWin:
		return h.validateWin(state, player, action)
	case core.ActionTing:
		return h.validateTing(state, player)
	case core.ActionQiangKong:
		return h.validateQiangKong(state, player, action)
	case core.ActionPass:
		return nil
	default:
		return fmt.Errorf("不支持的动作类型: %s", action.Type.String())
	}
}

// validateDraw 验证摸牌
func (h *ActionHandler) validateDraw(state *core.GameState, player *core.Player) error {
	// 必须是当前玩家
	if state.GetCurrentPlayer() != player {
		return fmt.Errorf("不是当前玩家")
	}

	// 牌堆必须有牌
	if len(state.Deck) == 0 {
		return fmt.Errorf("牌堆已空")
	}

	return nil
}

// validateDiscard 验证出牌
func (h *ActionHandler) validateDiscard(state *core.GameState, player *core.Player, action core.Action) error {
	// 必须是当前玩家
	if state.GetCurrentPlayer() != player {
		return fmt.Errorf("不是当前玩家")
	}

	// 必须有出牌的牌
	if action.Tile == nil {
		return fmt.Errorf("没有指定出牌")
	}

	// 手牌中必须有这张牌
	if !core.ContainsTile(player.Hand, *action.Tile) {
		return fmt.Errorf("手牌中没有这张牌")
	}

	return nil
}

// validatePong 验证碰牌
func (h *ActionHandler) validatePong(state *core.GameState, player *core.Player, action core.Action) error {
	// 必须有上一个动作且是出牌
	if state.LastAction == nil || state.LastAction.Type != core.ActionDiscard {
		return fmt.Errorf("没有可碰的牌")
	}

	lastTile := state.LastAction.Tile
	if lastTile == nil {
		return fmt.Errorf("上一个动作没有牌")
	}

	// 手牌中必须有2张相同的牌
	count := core.CountTile(player.Hand, *lastTile)
	if count < 2 {
		return fmt.Errorf("手牌中没有足够的牌来碰")
	}

	return nil
}

// validateKong 验证杠牌
func (h *ActionHandler) validateKong(state *core.GameState, player *core.Player, action core.Action) error {
	if action.Tile == nil {
		return fmt.Errorf("没有指定杠牌")
	}

	// 明杠: 手牌有3张,碰别人的牌
	if state.LastAction != nil && state.LastAction.Type == core.ActionDiscard {
		lastTile := state.LastAction.Tile
		if lastTile == nil {
			return fmt.Errorf("上一个动作没有牌")
		}
		count := core.CountTile(player.Hand, *lastTile)
		if count < 3 {
			return fmt.Errorf("手牌中没有足够的牌来杠")
		}
		return nil
	}

	// 暗杠: 手牌有4张
	count := core.CountTile(player.Hand, *action.Tile)
	if count < 4 {
		return fmt.Errorf("手牌中没有足够的牌来杠")
	}

	return nil
}

// validateWin 验证胡牌
func (h *ActionHandler) validateWin(state *core.GameState, player *core.Player, action core.Action) error {
	// 胡牌验证由 WinningAlgorithm 完成
	return nil
}

// validateTing 验证报听
func (h *ActionHandler) validateTing(state *core.GameState, player *core.Player) error {
	// 必须是第一轮
	if state.Round > 1 {
		return fmt.Errorf("只能在第一轮报听")
	}

	// 玩家状态检查
	htState, ok := player.State.(*HTPlayerState)
	if !ok {
		return fmt.Errorf("玩家状态类型错误")
	}

	if htState.IsTing {
		return fmt.Errorf("已经报听")
	}

	return nil
}

// validateQiangKong 验证抢杠
func (h *ActionHandler) validateQiangKong(state *core.GameState, player *core.Player, action core.Action) error {
	// 必须有上一个动作且是杠牌
	if state.LastAction == nil || state.LastAction.Type != core.ActionKong {
		return fmt.Errorf("没有可抢的杠")
	}

	return nil
}

// ExecuteAction 执行动作
func (h *ActionHandler) ExecuteAction(state *core.GameState, action core.Action) error {
	player := state.GetPlayer(action.PlayerID)
	if player == nil {
		return fmt.Errorf("玩家不存在: %s", action.PlayerID)
	}

	switch action.Type {
	case core.ActionDraw:
		return h.executeDraw(state, player)
	case core.ActionDiscard:
		return h.executeDiscard(state, player, action)
	case core.ActionPong:
		return h.executePong(state, player, action)
	case core.ActionKong:
		return h.executeKong(state, player, action)
	case core.ActionTing:
		return h.executeTing(state, player)
	case core.ActionPass:
		return nil
	default:
		return fmt.Errorf("不支持的动作类型: %s", action.Type.String())
	}
}

// executeDraw 执行摸牌
func (h *ActionHandler) executeDraw(state *core.GameState, player *core.Player) error {
	if len(state.Deck) == 0 {
		return fmt.Errorf("牌堆已空")
	}

	// 从牌堆摸一张牌
	tile := state.Deck[0]
	state.Deck = state.Deck[1:]

	// 加入手牌
	player.Hand = append(player.Hand, tile)
	core.SortTiles(player.Hand)

	return nil
}

// executeDiscard 执行出牌
func (h *ActionHandler) executeDiscard(state *core.GameState, player *core.Player, action core.Action) error {
	if action.Tile == nil {
		return fmt.Errorf("没有指定出牌")
	}

	// 从手牌移除
	player.Hand = core.RemoveTile(player.Hand, *action.Tile)

	// 加入弃牌堆
	player.Discards = append(player.Discards, *action.Tile)

	return nil
}

// executePong 执行碰牌
func (h *ActionHandler) executePong(state *core.GameState, player *core.Player, action core.Action) error {
	if state.LastAction == nil || state.LastAction.Tile == nil {
		return fmt.Errorf("没有可碰的牌")
	}

	tile := *state.LastAction.Tile

	// 从手牌移除2张
	player.Hand = core.RemoveTile(player.Hand, tile)
	player.Hand = core.RemoveTile(player.Hand, tile)

	// 添加到明牌组合
	player.Melds = append(player.Melds, core.Meld{
		Type:  core.MeldTypePong,
		Tiles: []core.Tile{tile, tile, tile},
	})

	// 移除上家的弃牌
	lastPlayer := state.GetPlayer(state.LastAction.PlayerID)
	if lastPlayer != nil && len(lastPlayer.Discards) > 0 {
		lastPlayer.Discards = lastPlayer.Discards[:len(lastPlayer.Discards)-1]
	}

	return nil
}

// executeKong 执行杠牌
func (h *ActionHandler) executeKong(state *core.GameState, player *core.Player, action core.Action) error {
	if action.Tile == nil {
		return fmt.Errorf("没有指定杠牌")
	}

	tile := *action.Tile

	// 明杠
	if state.LastAction != nil && state.LastAction.Type == core.ActionDiscard {
		// 从手牌移除3张
		for i := 0; i < 3; i++ {
			player.Hand = core.RemoveTile(player.Hand, tile)
		}

		// 移除上家的弃牌
		lastPlayer := state.GetPlayer(state.LastAction.PlayerID)
		if lastPlayer != nil && len(lastPlayer.Discards) > 0 {
			lastPlayer.Discards = lastPlayer.Discards[:len(lastPlayer.Discards)-1]
		}
	} else {
		// 暗杠: 从手牌移除4张
		for i := 0; i < 4; i++ {
			player.Hand = core.RemoveTile(player.Hand, tile)
		}
	}

	// 添加到明牌组合
	player.Melds = append(player.Melds, core.Meld{
		Type:  core.MeldTypeKong,
		Tiles: []core.Tile{tile, tile, tile, tile},
	})

	// 杠牌后摸一张牌
	if len(state.Deck) > 0 {
		newTile := state.Deck[0]
		state.Deck = state.Deck[1:]
		player.Hand = append(player.Hand, newTile)
		core.SortTiles(player.Hand)
	}

	return nil
}

// executeTing 执行报听
func (h *ActionHandler) executeTing(state *core.GameState, player *core.Player) error {
	htState, ok := player.State.(*HTPlayerState)
	if !ok {
		return fmt.Errorf("玩家状态类型错误")
	}

	htState.IsTing = true
	htState.CanTingRound = state.Round

	return nil
}

// GetAvailableActions 获取玩家可用的动作
func (h *ActionHandler) GetAvailableActions(state *core.GameState, playerID string) []core.ActionType {
	player := state.GetPlayer(playerID)
	if player == nil {
		return []core.ActionType{}
	}

	actions := []core.ActionType{}

	// 当前玩家可以摸牌和出牌
	if state.GetCurrentPlayer() == player {
		if len(state.Deck) > 0 {
			actions = append(actions, core.ActionDraw)
		}
		if len(player.Hand) > 0 {
			actions = append(actions, core.ActionDiscard)
		}

		// 检查是否可以报听
		if state.Round == 1 {
			htState, ok := player.State.(*HTPlayerState)
			if ok && !htState.IsTing {
				actions = append(actions, core.ActionTing)
			}
		}
	}

	return actions
}
