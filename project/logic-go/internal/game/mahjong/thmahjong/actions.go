package thmahjong

import (
	"fmt"

	"sudooom.im.logic/internal/game/mahjong/core"
)

// ActionHandler 太湖麻将动作处理器
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
	case core.ActionChi:
		return h.validateChi(state, player, action)
	case core.ActionKong:
		return h.validateKong(state, player, action)
	case core.ActionWin:
		return h.validateWin(state, player, action)
	case core.ActionFlower:
		return h.validateFlower(state, player, action)
	case core.ActionPass:
		return nil
	default:
		return fmt.Errorf("不支持的动作类型: %s", action.Type.String())
	}
}

// validateDraw 验证摸牌
func (h *ActionHandler) validateDraw(state *core.GameState, player *core.Player) error {
	if state.GetCurrentPlayer() != player {
		return fmt.Errorf("不是当前玩家")
	}

	if len(state.Deck) == 0 {
		return fmt.Errorf("牌堆已空")
	}

	return nil
}

// validateDiscard 验证出牌
func (h *ActionHandler) validateDiscard(state *core.GameState, player *core.Player, action core.Action) error {
	if state.GetCurrentPlayer() != player {
		return fmt.Errorf("不是当前玩家")
	}

	if action.Tile == nil {
		return fmt.Errorf("没有指定出牌")
	}

	if !core.ContainsTile(player.Hand, *action.Tile) {
		return fmt.Errorf("手牌中没有这张牌")
	}

	return nil
}

// validatePong 验证碰牌
func (h *ActionHandler) validatePong(state *core.GameState, player *core.Player, action core.Action) error {
	if state.LastAction == nil || state.LastAction.Type != core.ActionDiscard {
		return fmt.Errorf("没有可碰的牌")
	}

	lastTile := state.LastAction.Tile
	if lastTile == nil {
		return fmt.Errorf("上一个动作没有牌")
	}

	count := core.CountTile(player.Hand, *lastTile)
	if count < 2 {
		return fmt.Errorf("手牌中没有足够的牌来碰")
	}

	return nil
}

// validateChi 验证吃牌
func (h *ActionHandler) validateChi(state *core.GameState, player *core.Player, action core.Action) error {
	if state.LastAction == nil || state.LastAction.Type != core.ActionDiscard {
		return fmt.Errorf("没有可吃的牌")
	}

	lastTile := state.LastAction.Tile
	if lastTile == nil {
		return fmt.Errorf("上一个动作没有牌")
	}

	// 必须是上家打的牌
	lastPlayerIndex := state.GetPlayerIndex(state.LastAction.PlayerID)
	currentPlayerIndex := state.GetPlayerIndex(player.ID)
	if (lastPlayerIndex+1)%len(state.Players) != currentPlayerIndex {
		return fmt.Errorf("只能吃上家的牌")
	}

	// 风、箭牌不能吃
	if lastTile.Suit >= core.TileSuitWind {
		return fmt.Errorf("风牌和箭牌不能吃")
	}

	// 检查是否有指定的牌组
	if len(action.Tiles) != 3 {
		return fmt.Errorf("吃牌必须指定3张牌")
	}

	// 检查是否包含上一张牌
	if !core.ContainsTile(action.Tiles, *lastTile) {
		return fmt.Errorf("吃牌必须包含上一张牌")
	}

	// 检查是否是顺子
	if !core.IsSequence(action.Tiles) {
		return fmt.Errorf("吃牌必须是顺子")
	}

	// 检查手牌中是否有另外2张牌
	otherTiles := []core.Tile{}
	for _, t := range action.Tiles {
		if !t.Equal(*lastTile) {
			otherTiles = append(otherTiles, t)
		}
	}

	if !core.ContainsTiles(player.Hand, otherTiles) {
		return fmt.Errorf("手牌中没有吃牌所需的牌")
	}

	return nil
}

// validateKong 验证杠牌
func (h *ActionHandler) validateKong(state *core.GameState, player *core.Player, action core.Action) error {
	if action.Tile == nil {
		return fmt.Errorf("没有指定杠牌")
	}

	// 明杠: 碰别人的牌
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

// validateFlower 验证花牌
func (h *ActionHandler) validateFlower(state *core.GameState, player *core.Player, action core.Action) error {
	if action.Tile == nil {
		return fmt.Errorf("没有指定花牌")
	}

	// 必须是花牌
	if action.Tile.Suit != core.TileSuitFlower {
		return fmt.Errorf("不是花牌")
	}

	// 手牌中必须有这张牌
	if !core.ContainsTile(player.Hand, *action.Tile) {
		return fmt.Errorf("手牌中没有这张花牌")
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
	case core.ActionChi:
		return h.executeChi(state, player, action)
	case core.ActionKong:
		return h.executeKong(state, player, action)
	case core.ActionFlower:
		return h.executeFlower(state, player, action)
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

	tile := state.Deck[0]
	state.Deck = state.Deck[1:]

	player.Hand = append(player.Hand, tile)
	core.SortTiles(player.Hand)

	// 如果摸到花牌,自动补花
	if tile.Suit == core.TileSuitFlower {
		h.autoFlower(state, player, tile)
	}

	return nil
}

// executeDiscard 执行出牌
func (h *ActionHandler) executeDiscard(state *core.GameState, player *core.Player, action core.Action) error {
	if action.Tile == nil {
		return fmt.Errorf("没有指定出牌")
	}

	player.Hand = core.RemoveTile(player.Hand, *action.Tile)
	player.Discards = append(player.Discards, *action.Tile)

	return nil
}

// executePong 执行碰牌
func (h *ActionHandler) executePong(state *core.GameState, player *core.Player, action core.Action) error {
	if state.LastAction == nil || state.LastAction.Tile == nil {
		return fmt.Errorf("没有可碰的牌")
	}

	tile := *state.LastAction.Tile

	player.Hand = core.RemoveTile(player.Hand, tile)
	player.Hand = core.RemoveTile(player.Hand, tile)

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

// executeChi 执行吃牌
func (h *ActionHandler) executeChi(state *core.GameState, player *core.Player, action core.Action) error {
	if state.LastAction == nil || state.LastAction.Tile == nil {
		return fmt.Errorf("没有可吃的牌")
	}

	lastTile := *state.LastAction.Tile

	// 移除手牌中的牌
	for _, t := range action.Tiles {
		if !t.Equal(lastTile) {
			player.Hand = core.RemoveTile(player.Hand, t)
		}
	}

	player.Melds = append(player.Melds, core.Meld{
		Type:  core.MeldTypeChi,
		Tiles: action.Tiles,
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
		for i := 0; i < 3; i++ {
			player.Hand = core.RemoveTile(player.Hand, tile)
		}

		// 移除上家的弃牌
		lastPlayer := state.GetPlayer(state.LastAction.PlayerID)
		if lastPlayer != nil && len(lastPlayer.Discards) > 0 {
			lastPlayer.Discards = lastPlayer.Discards[:len(lastPlayer.Discards)-1]
		}
	} else {
		// 暗杠
		for i := 0; i < 4; i++ {
			player.Hand = core.RemoveTile(player.Hand, tile)
		}
	}

	player.Melds = append(player.Melds, core.Meld{
		Type:  core.MeldTypeKong,
		Tiles: []core.Tile{tile, tile, tile, tile},
	})

	// 更新杠数
	if thState, ok := player.State.(*THPlayerState); ok {
		thState.KongCount++
	}

	// 杠牌后摸一张牌
	if len(state.Deck) > 0 {
		newTile := state.Deck[0]
		state.Deck = state.Deck[1:]
		player.Hand = append(player.Hand, newTile)
		core.SortTiles(player.Hand)

		// 如果摸到花牌,自动补花
		if newTile.Suit == core.TileSuitFlower {
			h.autoFlower(state, player, newTile)
		}
	}

	return nil
}

// executeFlower 执行花牌
func (h *ActionHandler) executeFlower(state *core.GameState, player *core.Player, action core.Action) error {
	if action.Tile == nil {
		return fmt.Errorf("没有指定花牌")
	}

	h.autoFlower(state, player, *action.Tile)
	return nil
}

// autoFlower 自动补花
func (h *ActionHandler) autoFlower(state *core.GameState, player *core.Player, flowerTile core.Tile) {
	// 从手牌移除花牌
	player.Hand = core.RemoveTile(player.Hand, flowerTile)

	// 更新花数
	if thState, ok := player.State.(*THPlayerState); ok {
		thState.FlowerCount++
	}

	// 从牌堆补一张牌
	if len(state.Deck) > 0 {
		newTile := state.Deck[0]
		state.Deck = state.Deck[1:]
		player.Hand = append(player.Hand, newTile)
		core.SortTiles(player.Hand)

		// 如果又是花牌,继续补花
		if newTile.Suit == core.TileSuitFlower {
			h.autoFlower(state, player, newTile)
		}
	}
}

// GetAvailableActions 获取玩家可用的动作
func (h *ActionHandler) GetAvailableActions(state *core.GameState, playerID string) []core.ActionType {
	player := state.GetPlayer(playerID)
	if player == nil {
		return []core.ActionType{}
	}

	actions := []core.ActionType{}

	if state.GetCurrentPlayer() == player {
		if len(state.Deck) > 0 {
			actions = append(actions, core.ActionDraw)
		}
		if len(player.Hand) > 0 {
			actions = append(actions, core.ActionDiscard)
		}
	}

	return actions
}
