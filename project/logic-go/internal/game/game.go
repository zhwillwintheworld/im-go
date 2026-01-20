package game

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// gameInitializer 游戏初始化接口
// 用于类型断言，避免直接依赖具体游戏类型
type gameInitializer interface {
	Initialize(ctx context.Context, playerIDs []string) error
}

// gameActionHandler 游戏动作处理接口
type gameActionHandler interface {
	HandleAction(ctx context.Context, playerID string, action interface{}) error
}

// gameStateGetter 游戏状态获取接口
type gameStateGetter interface {
	GetState() interface{}
}

// Game 游戏对象
// 管理游戏状态，使用 RWMutex 保证并发安全
type Game struct {
	mu sync.RWMutex // 并发保护锁，保护所有字段

	roomID     string    // 房间ID，全局唯一标识符
	gameType   string    // 游戏类型：HT_MAHJONG（会同麻将）、DOUDIZHU（斗地主）等
	status     string    // 游戏状态：playing（进行中）、finished（已结束）
	lastActive time.Time // 最后活跃时间，用于LRU淘汰策略
	dirty      bool      // 脏标记，true表示有未保存的修改

	// 游戏引擎实例（实际类型为 *mahjong.SafeMahjongEngine 或其他游戏的 safe engine）
	// 使用 interface{} 避免循环依赖，通过小型接口+类型断言访问具体方法
	engine interface{}
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

	if g.engine == nil {
		return ErrEngineNotInitialized
	}

	// 转换玩家 ID
	playerIDs := make([]string, len(players))
	for i, id := range players {
		playerIDs[i] = fmt.Sprintf("%d", id)
	}

	// 使用接口进行类型断言
	if eng, ok := g.engine.(gameInitializer); ok {
		if err := eng.Initialize(ctx, playerIDs); err != nil {
			return err
		}
	} else {
		return ErrEngineNotSupported
	}

	g.dirty = true
	g.lastActive = time.Now()
	return nil
}

// HandlePlayerAction 处理玩家操作
func (g *Game) HandlePlayerAction(ctx context.Context, userId int64, action interface{}) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.engine == nil {
		return ErrEngineNotInitialized
	}

	playerID := fmt.Sprintf("%d", userId)

	// 使用接口进行类型断言
	if eng, ok := g.engine.(gameActionHandler); ok {
		if err := eng.HandleAction(ctx, playerID, action); err != nil {
			return err
		}
	} else {
		return ErrEngineNotSupported
	}

	g.dirty = true
	g.lastActive = time.Now()
	return nil
}

// GetState 获取游戏状态（只读）
func (g *Game) GetState() interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.engine == nil {
		return nil
	}

	// 使用接口进行类型断言
	if eng, ok := g.engine.(gameStateGetter); ok {
		return eng.GetState()
	}

	return nil
}

// SetEngine 设置游戏引擎
func (g *Game) SetEngine(engine interface{}) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.engine = engine
	g.lastActive = time.Now()
	g.dirty = true
}

// GetEngine 获取游戏引擎（只读）
func (g *Game) GetEngine() interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.engine
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
