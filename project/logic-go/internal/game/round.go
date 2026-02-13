package game

import (
	"context"
	"fmt"
	"sync"
	"time"

	"sudooom.im.logic/internal/game/mahjong/core"
	"sudooom.im.logic/internal/task"
)

// RoundSettlement 是 mahjong.RoundSettlement 的别名
// 避免循环依赖，实际类型定义在 mahjong 包中
type RoundSettlement interface{}

// RoundEngine 单局游戏引擎接口
// 由具体游戏引擎（如 SafeMahjongEngine）实现
type RoundEngine interface {
	// Initialize 初始化单局游戏
	Initialize(ctx context.Context, playerIDs []string) error

	// HandleAction 处理玩家动作
	HandleAction(ctx context.Context, playerID string, action core.Action) error

	// GetState 获取游戏状态
	GetState() *core.GameState

	// IsRoundOver 检查单局是否结束
	IsRoundOver() bool

	// GetSettlement 获取结算结果（返回 interface{} 避免循环依赖）
	GetSettlement() interface{}

	// ProcessTaskTimeout 处理任务超时
	ProcessTaskTimeout()
}

// RoundStatus 单局状态
type RoundStatus int

const (
	RoundStatusInitializing RoundStatus = iota
	RoundStatusPlaying
	RoundStatusSettling
	RoundStatusFinished
)

func (s RoundStatus) String() string {
	switch s {
	case RoundStatusInitializing:
		return "initializing"
	case RoundStatusPlaying:
		return "playing"
	case RoundStatusSettling:
		return "settling"
	case RoundStatusFinished:
		return "finished"
	default:
		return "unknown"
	}
}

// Round 单局游戏
// opMu 保证操作串行化
// mu 保护 Round 自身元数据字段
type Round struct {
	opMu sync.Mutex   // 操作级互斥锁
	mu   sync.RWMutex // 保护元数据字段

	ID          string
	GameID      string
	RoundNumber int

	// 引擎
	engine RoundEngine

	// 任务管理
	scheduler   *task.Scheduler
	taskVersion int64

	// 轮次状态
	status    RoundStatus
	startTime time.Time
	endTime   time.Time

	// 结算（使用 interface{} 避免循环依赖）
	settlement interface{}
}

// NewRound 创建单局游戏
func NewRound(roundID string, gameID string, roundNumber int, scheduler *task.Scheduler) *Round {
	return &Round{
		ID:          roundID,
		GameID:      gameID,
		RoundNumber: roundNumber,
		scheduler:   scheduler,
		status:      RoundStatusInitializing,
		startTime:   time.Now(),
	}
}

// Initialize 初始化单局游戏
func (r *Round) Initialize(ctx context.Context, playerIDs []string) error {
	r.opMu.Lock()
	defer r.opMu.Unlock()

	r.mu.RLock()
	eng := r.engine
	status := r.status
	r.mu.RUnlock()

	if eng == nil {
		return ErrEngineNotInitialized
	}

	if status != RoundStatusInitializing {
		return fmt.Errorf("invalid round status: %s, expected initializing", status)
	}

	if err := eng.Initialize(ctx, playerIDs); err != nil {
		return err
	}

	r.mu.Lock()
	r.status = RoundStatusPlaying
	r.mu.Unlock()

	return nil
}

// HandlePlayerAction 处理玩家操作
func (r *Round) HandlePlayerAction(ctx context.Context, userId int64, action core.Action) error {
	r.opMu.Lock()
	defer r.opMu.Unlock()

	r.mu.RLock()
	eng := r.engine
	status := r.status
	r.mu.RUnlock()

	if eng == nil {
		return ErrEngineNotInitialized
	}

	if status != RoundStatusPlaying {
		return fmt.Errorf("invalid round status: %s, expected playing", status)
	}

	// 玩家操作时，递增任务版本号（标记旧任务失效）
	r.mu.Lock()
	r.taskVersion++
	r.mu.Unlock()

	playerID := fmt.Sprintf("%d", userId)
	if err := eng.HandleAction(ctx, playerID, action); err != nil {
		return err
	}

	// 检查是否结束
	if eng.IsRoundOver() {
		r.mu.Lock()
		r.status = RoundStatusSettling
		r.settlement = eng.GetSettlement()
		r.endTime = time.Now()
		r.mu.Unlock()
		return nil
	}

	// 检查是否有新任务
	state := eng.GetState()
	if len(state.PendingTasks) > 0 {
		r.mu.Lock()
		r.scheduleTaskTimeout(30) // 30秒超时
		r.mu.Unlock()
	}

	return nil
}

// GetState 获取游戏状态
func (r *Round) GetState() *core.GameState {
	r.mu.RLock()
	eng := r.engine
	r.mu.RUnlock()

	if eng == nil {
		return nil
	}

	return eng.GetState()
}

// GetSettlement 获取结算结果（返回 interface{} 避免循环依赖）
func (r *Round) GetSettlement() interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.settlement
}

// GetStatus 获取状态
func (r *Round) GetStatus() RoundStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.status
}

// SetEngine 设置引擎
func (r *Round) SetEngine(engine RoundEngine) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.engine = engine
}

// GetEngine 获取引擎
func (r *Round) GetEngine() RoundEngine {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.engine
}

// onTaskTimeout 任务超时回调
func (r *Round) onTaskTimeout(taskVersion int64) {
	r.opMu.Lock()
	defer r.opMu.Unlock()

	r.mu.RLock()
	eng := r.engine
	status := r.status
	currentVersion := r.taskVersion
	r.mu.RUnlock()

	// 版本号不匹配，任务已失效
	if taskVersion != currentVersion {
		return
	}

	// 游戏已结束则忽略
	if status != RoundStatusPlaying || eng == nil {
		return
	}

	// 处理超时
	eng.ProcessTaskTimeout()

	// 检查是否结束
	if eng.IsRoundOver() {
		r.mu.Lock()
		r.status = RoundStatusSettling
		r.settlement = eng.GetSettlement()
		r.endTime = time.Now()
		r.mu.Unlock()
	}
}

// scheduleTaskTimeout 调度任务超时（必须在 mu 锁内调用）
func (r *Round) scheduleTaskTimeout(delaySec int) {
	if r.scheduler == nil {
		return
	}

	r.taskVersion++
	currentVersion := r.taskVersion

	taskID := fmt.Sprintf("round:%s:timeout:%d", r.ID, currentVersion)

	t := task.NewTask(taskID, r.ID, delaySec, func(ctx context.Context, target string, metadata map[string]any) error {
		r.onTaskTimeout(currentVersion)
		return nil
	})

	_ = r.scheduler.AddTask(t)
}

// Close 关闭单局，释放资源
func (r *Round) Close() {
	r.mu.Lock()
	r.taskVersion++
	r.mu.Unlock()
}

// Finish 完成单局（标记为已结束）
func (r *Round) Finish() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.status != RoundStatusSettling {
		return fmt.Errorf("invalid round status: %s, expected settling", r.status)
	}

	r.status = RoundStatusFinished
	return nil
}
