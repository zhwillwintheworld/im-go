package core

import "context"

// GameEngine 游戏引擎接口
type GameEngine interface {
	// Initialize 初始化游戏
	Initialize(ctx context.Context, playerIDs []string, config GameConfig) error

	// HandleAction 处理玩家动作
	HandleAction(ctx context.Context, action Action) error

	// GetState 获取游戏状态
	GetState() *GameState

	// IsGameOver 检查游戏是否结束
	IsGameOver() bool

	// GetSettlement 获取结算结果
	GetSettlement() *Settlement
}

// DeckGenerator 牌局生成器接口
type DeckGenerator interface {
	// GenerateDeck 生成牌堆
	GenerateDeck() []Tile

	// Shuffle 洗牌
	Shuffle(tiles []Tile)

	// Deal 发牌
	Deal(tiles []Tile, playerCount int, dealerIndex int) (hands map[int][]Tile, remaining []Tile)
}

// ActionHandler 动作处理器接口
type ActionHandler interface {
	// ValidateAction 验证动作是否合法
	ValidateAction(state *GameState, action Action) error

	// ExecuteAction 执行动作
	ExecuteAction(state *GameState, action Action) error

	// GetAvailableActions 获取玩家可用的动作
	GetAvailableActions(state *GameState, playerID string) []ActionType
}

// TaskJudge 任务判断器接口
type TaskJudge interface {
	// JudgeTasks 判断是否有任务产生 (其他玩家是否可以碰/杠/胡等)
	JudgeTasks(state *GameState, action Action) []Task

	// GetTaskPriority 获取任务优先级 (胡>杠>碰>吃)
	GetTaskPriority(task Task) int
}

// WinningAlgorithm 胡牌算法接口
type WinningAlgorithm interface {
	// CanWin 检查是否可以胡牌
	CanWin(hand []Tile, newTile *Tile, state *GameState, playerID string) bool

	// GetWinPatterns 获取胡牌牌型
	GetWinPatterns(hand []Tile, newTile *Tile, state *GameState, playerID string) []WinPattern

	// CalculateScore 计算番数/分数
	CalculateScore(patterns []WinPattern) int
}

// Settler 结算器接口
type Settler interface {
	// Calculate 计算结算结果
	Calculate(state *GameState, winnerID string, loserID string, winType WinType, patterns []WinPattern) *Settlement
}
