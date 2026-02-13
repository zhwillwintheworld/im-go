package game

import (
	"context"
	"fmt"
	"sync"
	"time"

	"sudooom.im.logic/internal/game/types"
	"sudooom.im.logic/internal/task"
)

// MultiRoundGameStatus 多轮游戏状态
type MultiRoundGameStatus int

const (
	MultiRoundGameStatusPlaying MultiRoundGameStatus = iota
	MultiRoundGameStatusFinished
)

func (s MultiRoundGameStatus) String() string {
	switch s {
	case MultiRoundGameStatusPlaying:
		return "playing"
	case MultiRoundGameStatusFinished:
		return "finished"
	default:
		return "unknown"
	}
}

// GameConfig 游戏配置
type GameConfig struct {
	MaxRounds int            `json:"maxRounds"` // 最大局数（0 表示无限制）
	MaxScore  int            `json:"maxScore"`  // 封顶分数（0 表示无限制）
	BaseScore int            `json:"baseScore"` // 底分
	TimeLimit time.Duration  `json:"timeLimit"` // 单局时间限制
	Extra     map[string]any `json:"extra"`     // 扩展配置
}

// MultiRoundGame 游戏（多轮管理器）
type MultiRoundGame struct {
	mu sync.RWMutex

	ID       string
	RoomID   string
	GameType string
	Config   GameConfig

	// 轮次管理
	rounds       []*Round // 所有轮次
	currentRound *Round   // 当前轮次
	roundNumber  int      // 当前轮次号

	// 分数统计
	playerScores map[int64]int // 玩家ID → 总分（累计）

	// 游戏状态
	status    MultiRoundGameStatus
	startTime time.Time
	endTime   time.Time

	// 依赖
	scheduler *task.Scheduler

	// 最后活跃时间
	lastActive time.Time
	dirty      bool
}

// NewMultiRoundGame 创建游戏
func NewMultiRoundGame(gameID string, roomID string, gameType string, config GameConfig, scheduler *task.Scheduler) *MultiRoundGame {
	return &MultiRoundGame{
		ID:           gameID,
		RoomID:       roomID,
		GameType:     gameType,
		Config:       config,
		rounds:       make([]*Round, 0),
		playerScores: make(map[int64]int),
		status:       MultiRoundGameStatusPlaying,
		startTime:    time.Now(),
		scheduler:    scheduler,
		lastActive:   time.Now(),
	}
}

// StartNewRound 开始新的一局
func (g *MultiRoundGame) StartNewRound(ctx context.Context, playerIDs []int64, engineFactory func() RoundEngine) (*Round, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.status != MultiRoundGameStatusPlaying {
		return nil, fmt.Errorf("game is not playing: %s", g.status)
	}

	// 检查是否已达到最大局数
	if g.Config.MaxRounds > 0 && g.roundNumber >= g.Config.MaxRounds {
		return nil, fmt.Errorf("max rounds reached: %d", g.Config.MaxRounds)
	}

	// 创建新的 Round
	g.roundNumber++
	roundID := fmt.Sprintf("%s:round:%d", g.ID, g.roundNumber)
	round := NewRound(roundID, g.ID, g.roundNumber, g.scheduler)

	// 设置引擎
	engine := engineFactory()
	round.SetEngine(engine)

	// 初始化
	playerIDStrs := make([]string, len(playerIDs))
	for i, id := range playerIDs {
		playerIDStrs[i] = fmt.Sprintf("%d", id)
	}

	if err := round.Initialize(ctx, playerIDStrs); err != nil {
		return nil, fmt.Errorf("failed to initialize round: %w", err)
	}

	// 保存
	g.rounds = append(g.rounds, round)
	g.currentRound = round
	g.lastActive = time.Now()
	g.dirty = true

	return round, nil
}

// FinishCurrentRound 完成当前局
func (g *MultiRoundGame) FinishCurrentRound(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.currentRound == nil {
		return fmt.Errorf("no current round")
	}

	// 获取结算结果
	settlementInterface := g.currentRound.GetSettlement()
	if settlementInterface == nil {
		return fmt.Errorf("no settlement available")
	}

	// 类型断言为 types.RoundSettlement
	settlement, ok := settlementInterface.(*types.RoundSettlement)
	if !ok {
		return fmt.Errorf("invalid settlement type")
	}

	// 更新总分
	for _, winner := range settlement.Winners {
		g.playerScores[winner.PlayerID] += winner.ScoreChange
	}
	for _, loser := range settlement.Losers {
		g.playerScores[loser.PlayerID] += loser.ScoreChange
	}

	// 标记 Round 为已完成
	if err := g.currentRound.Finish(); err != nil {
		return err
	}

	// TODO: 持久化 Round 记录到数据库
	// roundService.SaveRound(ctx, g.currentRound, settlement)

	// 检查游戏是否结束
	if g.shouldGameEnd() {
		g.status = MultiRoundGameStatusFinished
		g.endTime = time.Now()
		// TODO: 持久化 Game 总分记录到数据库
		// gameService.SaveGame(ctx, g)
	}

	g.lastActive = time.Now()
	g.dirty = true

	return nil
}

// shouldGameEnd 检查游戏是否应该结束
func (g *MultiRoundGame) shouldGameEnd() bool {
	// 达到最大局数
	if g.Config.MaxRounds > 0 && g.roundNumber >= g.Config.MaxRounds {
		return true
	}

	// 有玩家达到封顶分数
	if g.Config.MaxScore > 0 {
		for _, score := range g.playerScores {
			if score >= g.Config.MaxScore {
				return true
			}
		}
	}

	return false
}

// GetCurrentRound 获取当前局
func (g *MultiRoundGame) GetCurrentRound() *Round {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.currentRound
}

// GetPlayerScores 获取玩家总分
func (g *MultiRoundGame) GetPlayerScores() map[int64]int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// 返回副本
	scores := make(map[int64]int, len(g.playerScores))
	for k, v := range g.playerScores {
		scores[k] = v
	}
	return scores
}

// GetStatus 获取游戏状态
func (g *MultiRoundGame) GetStatus() MultiRoundGameStatus {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.status
}

// GetRoundNumber 获取当前轮次号
func (g *MultiRoundGame) GetRoundNumber() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.roundNumber
}

// IsDirty 是否有未保存的修改
func (g *MultiRoundGame) IsDirty() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.dirty
}

// MarkClean 标记为已保存
func (g *MultiRoundGame) MarkClean() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.dirty = false
}

// LastActiveTime 获取最后活跃时间
func (g *MultiRoundGame) LastActiveTime() time.Time {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.lastActive
}

// Close 关闭游戏
func (g *MultiRoundGame) Close() {
	g.mu.Lock()
	defer g.mu.Unlock()

	// 关闭当前 Round
	if g.currentRound != nil {
		g.currentRound.Close()
	}
}
