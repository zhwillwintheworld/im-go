package types

import (
	"time"

	"sudooom.im.logic/internal/game/mahjong/core"
)

// PlayerScore 玩家分数变化
type PlayerScore struct {
	PlayerID    int64  `json:"playerId"`
	PlayerName  string `json:"playerName"`
	ScoreChange int    `json:"scoreChange"` // 分数变化（正数为赢，负数为输）
	Role        string `json:"role"`        // "winner", "loser", "neutral"
}

// RoundSettlement 单局结算信息
type RoundSettlement struct {
	RoundID     string `json:"roundId"`
	RoundNumber int    `json:"roundNumber"`

	// 输赢详情
	Winners []PlayerScore `json:"winners"` // 赢家列表（可能多人）
	Losers  []PlayerScore `json:"losers"`  // 输家列表（可能多人）

	// 结算类型
	SettlementType string `json:"settlementType"` // "win"(胡牌), "draw"(流局), "timeout"(超时)
	WinType        string `json:"winType"`        // "self_draw"(自摸), "discard"(点炮), ""(非胡牌)

	// 额外信息
	WinnerHand []core.Tile `json:"winnerHand"` // 胡牌手牌
	WinTile    *core.Tile  `json:"winTile"`    // 胡的牌
	FanType    []string    `json:"fanType"`    // 番型列表
	FanScore   int         `json:"fanScore"`   // 番数

	SettleTime time.Time `json:"settleTime"`
}
