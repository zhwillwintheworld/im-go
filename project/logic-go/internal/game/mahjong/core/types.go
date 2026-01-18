package core

import "time"

// TileSuit 牌的花色
type TileSuit int8

const (
	TileSuitWan    TileSuit = iota // 万
	TileSuitTiao                   // 条
	TileSuitTong                   // 筒
	TileSuitWind                   // 风 (东南西北)
	TileSuitDragon                 // 箭牌 (中发白)
	TileSuitFlower                 // 花牌
)

// String 返回花色的字符串表示
func (s TileSuit) String() string {
	switch s {
	case TileSuitWan:
		return "万"
	case TileSuitTiao:
		return "条"
	case TileSuitTong:
		return "筒"
	case TileSuitWind:
		return "风"
	case TileSuitDragon:
		return "箭"
	case TileSuitFlower:
		return "花"
	default:
		return "未知"
	}
}

// Tile 麻将牌
type Tile struct {
	Suit  TileSuit `json:"suit"`  // 花色
	Value int8     `json:"value"` // 值 (1-9, 风牌:1东2南3西4北, 箭牌:1中2发3白)
}

// String 返回牌的字符串表示
func (t Tile) String() string {
	return t.Suit.String() + string(rune('0'+t.Value))
}

// Equal 判断两张牌是否相同
func (t Tile) Equal(other Tile) bool {
	return t.Suit == other.Suit && t.Value == other.Value
}

// MeldType 明牌组合类型
type MeldType int8

const (
	MeldTypePong MeldType = iota // 碰 (3张相同)
	MeldTypeKong                 // 杠 (4张相同)
	MeldTypeChi                  // 吃 (3张顺子)
)

// Meld 明牌组合
type Meld struct {
	Type  MeldType `json:"type"`  // 组合类型
	Tiles []Tile   `json:"tiles"` // 牌
}

// ActionType 动作类型
type ActionType int8

const (
	ActionDraw      ActionType = iota // 摸牌
	ActionDiscard                     // 出牌
	ActionPong                        // 碰
	ActionKong                        // 杠
	ActionWin                         // 胡
	ActionChi                         // 吃
	ActionTing                        // 报听
	ActionFlower                      // 花
	ActionQiangKong                   // 抢杠
	ActionPass                        // 过
)

// String 返回动作类型的字符串表示
func (a ActionType) String() string {
	switch a {
	case ActionDraw:
		return "摸牌"
	case ActionDiscard:
		return "出牌"
	case ActionPong:
		return "碰"
	case ActionKong:
		return "杠"
	case ActionWin:
		return "胡"
	case ActionChi:
		return "吃"
	case ActionTing:
		return "报听"
	case ActionFlower:
		return "花"
	case ActionQiangKong:
		return "抢杠"
	case ActionPass:
		return "过"
	default:
		return "未知"
	}
}

// Action 玩家动作
type Action struct {
	Type     ActionType `json:"type"`     // 动作类型
	PlayerID string     `json:"playerId"` // 玩家ID
	Tile     *Tile      `json:"tile"`     // 相关的牌 (出牌/摸牌时使用)
	Tiles    []Tile     `json:"tiles"`    // 相关的多张牌 (吃/碰/杠时使用)
}

// PlayerState 玩家状态接口 (允许各麻将类型扩展)
type PlayerState interface {
	Clone() PlayerState
}

// Player 玩家
type Player struct {
	ID       string      `json:"id"`       // 玩家ID
	Hand     []Tile      `json:"hand"`     // 手牌
	Discards []Tile      `json:"discards"` // 弃牌
	Melds    []Meld      `json:"melds"`    // 明牌组合
	State    PlayerState `json:"state"`    // 自定义状态
	Score    int         `json:"score"`    // 分数
}

// Task 任务 (其他玩家可以执行的动作)
type Task struct {
	PlayerID       string       `json:"playerId"`       // 任务所属玩家
	AvailableTypes []ActionType `json:"availableTypes"` // 可执行的动作类型
	RelatedTile    *Tile        `json:"relatedTile"`    // 相关的牌
	Priority       int          `json:"priority"`       // 优先级 (胡>杠>碰>吃)
	Timeout        time.Time    `json:"timeout"`        // 超时时间
}

// WinType 胡牌类型
type WinType int8

const (
	WinTypeDraw      WinType = iota // 自摸
	WinTypeDiscard                  // 点炮
	WinTypeTing                     // 报听胡
	WinTypeQiangKong                // 抢杠胡
)

// String 返回胡牌类型的字符串表示
func (w WinType) String() string {
	switch w {
	case WinTypeDraw:
		return "自摸"
	case WinTypeDiscard:
		return "点炮"
	case WinTypeTing:
		return "报听胡"
	case WinTypeQiangKong:
		return "抢杠胡"
	default:
		return "未知"
	}
}

// WinPattern 胡牌牌型
type WinPattern struct {
	Name  string `json:"name"`  // 牌型名称 (如"清一色", "七小对")
	Score int    `json:"score"` // 牌型分数/番数
}

// Settlement 结算结果
type Settlement struct {
	WinnerID   string       `json:"winnerId"`   // 赢家ID
	LoserID    string       `json:"loserId"`    // 输家ID (点炮者)
	WinType    WinType      `json:"winType"`    // 胡牌类型
	Patterns   []WinPattern `json:"patterns"`   // 胡牌牌型
	BaseScore  int          `json:"baseScore"`  // 底分
	TotalScore int          `json:"totalScore"` // 总分
	Transfers  []Transfer   `json:"transfers"`  // 分数转移记录
}

// Transfer 分数转移记录
type Transfer struct {
	FromID string `json:"fromId"` // 转出玩家ID
	ToID   string `json:"toId"`   // 转入玩家ID
	Amount int    `json:"amount"` // 分数
	Reason string `json:"reason"` // 原因
}

// GameConfig 游戏配置
type GameConfig struct {
	PlayerCount int            `json:"playerCount"` // 玩家数量
	BaseScore   int            `json:"baseScore"`   // 底分
	Extra       map[string]any `json:"extra"`       // 扩展配置
}

// GameState 游戏状态
type GameState struct {
	Players       []*Player   `json:"players"`       // 玩家列表
	Deck          []Tile      `json:"deck"`          // 牌堆
	CurrentPlayer int         `json:"currentPlayer"` // 当前玩家索引
	LastAction    *Action     `json:"lastAction"`    // 上一个动作
	Round         int         `json:"round"`         // 当前轮次
	DealerIndex   int         `json:"dealerIndex"`   // 庄家索引
	Config        GameConfig  `json:"config"`        // 游戏配置
	IsGameOver    bool        `json:"isGameOver"`    // 游戏是否结束
	Settlement    *Settlement `json:"settlement"`    // 结算结果
	PendingTasks  []Task      `json:"pendingTasks"`  // 待处理任务
}

// GetPlayer 根据ID获取玩家
func (s *GameState) GetPlayer(playerID string) *Player {
	for _, p := range s.Players {
		if p.ID == playerID {
			return p
		}
	}
	return nil
}

// GetPlayerIndex 根据ID获取玩家索引
func (s *GameState) GetPlayerIndex(playerID string) int {
	for i, p := range s.Players {
		if p.ID == playerID {
			return i
		}
	}
	return -1
}

// GetCurrentPlayer 获取当前玩家
func (s *GameState) GetCurrentPlayer() *Player {
	if s.CurrentPlayer >= 0 && s.CurrentPlayer < len(s.Players) {
		return s.Players[s.CurrentPlayer]
	}
	return nil
}

// NextPlayer 切换到下一个玩家
func (s *GameState) NextPlayer() {
	s.CurrentPlayer = (s.CurrentPlayer + 1) % len(s.Players)
}
