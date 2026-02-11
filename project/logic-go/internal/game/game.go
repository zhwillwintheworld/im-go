package game

import (
	"context"
	"fmt"
	"sync"
	"time"

	"sudooom.im.logic/internal/game/mahjong/core"
)

// GameEngine 游戏引擎接口
// 由具体游戏引擎（如 SafeMahjongEngine）实现
// game 包通过此接口与引擎交互，避免依赖具体实现
type GameEngine interface {
	// Initialize 初始化游戏
	Initialize(ctx context.Context, playerIDs []string) error

	// HandleAction 处理玩家动作
	HandleAction(ctx context.Context, playerID string, action core.Action) error

	// GetState 获取游戏状态
	GetState() *core.GameState

	// IsGameOver 检查游戏是否结束
	IsGameOver() bool

	// CheckAndProcessTaskTimeout 检查并处理任务超时
	// 返回 true 表示有任务超时并被处理
	CheckAndProcessTaskTimeout() bool
}

// ActionResult 动作执行结果
// 调用方可根据此结果决定后续操作（广播、通知等）
type ActionResult struct {
	IsGameOver  bool // 游戏是否结束
	HasNewTasks bool // 是否产生了新的待处理任务
}

// Game 游戏对象
// opMu 保证 InitGame / HandlePlayerAction / onTaskTimeout 操作串行化
// mu 保护 Game 自身元数据字段的短时读写
// engine 内部的并发安全由引擎自身负责（如 SafeMahjongEngine.mu）
type Game struct {
	opMu sync.Mutex   // 操作级互斥锁，保证游戏操作串行化
	mu   sync.RWMutex // 保护元数据字段的短时读写

	roomID     string     // 房间ID，全局唯一标识符
	gameType   string     // 游戏类型：HT_MAHJONG（会同麻将）等
	status     GameStatus // 游戏状态（状态机）
	lastActive time.Time  // 最后活跃时间，用于LRU淘汰策略
	dirty      bool       // 脏标记，true表示有未保存的修改

	engine    GameEngine  // 游戏引擎（类型安全）
	taskTimer *time.Timer // 任务超时定时器
}

// NewGame 创建游戏
func NewGame(roomID string, gameType string) *Game {
	return &Game{
		roomID:     roomID,
		gameType:   gameType,
		status:     GameStatusWaiting,
		lastActive: time.Now(),
	}
}

// InitGame 初始化游戏
func (g *Game) InitGame(ctx context.Context, playerIDs []string) error {
	g.opMu.Lock()
	defer g.opMu.Unlock()

	g.mu.RLock()
	eng := g.engine
	status := g.status
	g.mu.RUnlock()

	if eng == nil {
		return ErrEngineNotInitialized
	}

	// 状态校验：只有 Waiting 状态才能初始化
	if err := status.ValidateTransition(GameStatusPlaying); err != nil {
		return err
	}

	// engine 内部负责并发安全
	if err := eng.Initialize(ctx, playerIDs); err != nil {
		return err
	}

	// 更新元数据
	g.mu.Lock()
	g.status = GameStatusPlaying
	g.dirty = true
	g.lastActive = time.Now()
	g.mu.Unlock()

	return nil
}

// HandlePlayerAction 处理玩家操作
func (g *Game) HandlePlayerAction(ctx context.Context, userId int64, action core.Action) (*ActionResult, error) {
	g.opMu.Lock()
	defer g.opMu.Unlock()

	g.mu.RLock()
	eng := g.engine
	status := g.status
	g.mu.RUnlock()

	if eng == nil {
		return nil, ErrEngineNotInitialized
	}

	// 状态校验：只有 Playing 状态才能处理动作
	if status != GameStatusPlaying {
		return nil, fmt.Errorf("%w: game is %s, expected playing", ErrInvalidGameState, status)
	}

	// engine 内部负责并发安全
	playerID := fmt.Sprintf("%d", userId)
	if err := eng.HandleAction(ctx, playerID, action); err != nil {
		return nil, err
	}

	// 构建执行结果（在 opMu 保护下读取，此时不会有并发修改）
	state := eng.GetState()
	result := &ActionResult{
		IsGameOver:  state.IsGameOver,
		HasNewTasks: len(state.PendingTasks) > 0,
	}

	// 根据结果处理状态转换和超时定时器（原子操作，在同一把锁内）
	g.mu.Lock()
	if result.IsGameOver {
		if err := g.status.ValidateTransition(GameStatusFinished); err == nil {
			g.status = GameStatusFinished
		}
		// 取消定时器（在 mu 锁内）
		if g.taskTimer != nil {
			g.taskTimer.Stop()
			g.taskTimer = nil
		}
	} else if result.HasNewTasks {
		// 启动任务超时定时器（在 mu 锁内）
		if g.taskTimer != nil {
			g.taskTimer.Stop()
		}
		g.taskTimer = time.AfterFunc(30*time.Second, g.onTaskTimeout)
	}
	g.dirty = true
	g.lastActive = time.Now()
	g.mu.Unlock()

	return result, nil
}

// GetState 获取游戏状态（只读）
func (g *Game) GetState() *core.GameState {
	g.mu.RLock()
	eng := g.engine
	g.mu.RUnlock()

	if eng == nil {
		return nil
	}

	return eng.GetState()
}

// SetEngine 设置游戏引擎
func (g *Game) SetEngine(engine GameEngine) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.engine = engine
	g.lastActive = time.Now()
	g.dirty = true
}

// GetEngine 获取游戏引擎（只读）
func (g *Game) GetEngine() GameEngine {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.engine
}

// GetStatus 获取当前游戏状态
func (g *Game) GetStatus() GameStatus {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.status
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

// RoomID 获取房间ID
func (g *Game) RoomID() string {
	return g.roomID
}

// onTaskTimeout 任务超时回调（由 time.AfterFunc 触发）
func (g *Game) onTaskTimeout() {
	g.opMu.Lock()
	defer g.opMu.Unlock()

	g.mu.RLock()
	eng := g.engine
	status := g.status
	g.mu.RUnlock()

	// 游戏已结束则忽略
	if status != GameStatusPlaying || eng == nil {
		return
	}

	if !eng.CheckAndProcessTaskTimeout() {
		return
	}

	// 超时处理后检查游戏是否结束
	g.mu.Lock()
	if eng.IsGameOver() {
		if err := g.status.ValidateTransition(GameStatusFinished); err == nil {
			g.status = GameStatusFinished
		}
	}
	g.dirty = true
	g.lastActive = time.Now()
	g.mu.Unlock()
}

// Close 关闭游戏，释放资源
func (g *Game) Close() {
	g.mu.Lock()
	if g.taskTimer != nil {
		g.taskTimer.Stop()
		g.taskTimer = nil
	}
	g.mu.Unlock()
}
