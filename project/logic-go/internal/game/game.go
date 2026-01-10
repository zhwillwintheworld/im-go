package game

import (
	"context"
	"sync"
	"time"
)

// Game 游戏对象
// 管理游戏状态，使用 RWMutex 保证并发安全
type Game struct {
	mu sync.RWMutex

	roomID     string
	gameType   string // HT_MAHJONG, DOUDIZHU 等
	status     string // playing, finished
	lastActive time.Time
	dirty      bool

	// 游戏状态（不同游戏类型不同）
	state interface{} // *MahjongState, *DouDiZhuState 等
}

// NewGame 创建游戏
func NewGame(roomID string, gameType string) *Game {
	return &Game{
		roomID:     roomID,
		gameType:   gameType,
		status:     "playing",
		lastActive: time.Now(),
	}
}

// InitMahjongGame 初始化麻将游戏
func (g *Game) InitMahjongGame(ctx context.Context, players []int64) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// TODO: 初始化麻将游戏状态
	// - 创建牌堆
	// - 发牌
	// - 设置庄家

	g.dirty = true
	g.lastActive = time.Now()
	return nil
}

// HandlePlayerAction 处理玩家操作
func (g *Game) HandlePlayerAction(ctx context.Context, userId int64, action interface{}) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// TODO: 根据 gameType 和 action 处理

	g.dirty = true
	g.lastActive = time.Now()
	return nil
}

// GetState 获取游戏状态（只读）
func (g *Game) GetState() interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.state
}

// IsDirty 是否有未保存的修改
func (g *Game) IsDirty() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.dirty
}

// MarkClean 标记为已保存
func (g *Game) MarkClean() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.dirty = false
}

// LastActiveTime 获取最后活跃时间
func (g *Game) LastActiveTime() time.Time {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.lastActive
}
