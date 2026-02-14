# Game 模块设计文档

> 游戏引擎模块 (`project/logic-go/internal/game`) 设计说明与使用指南

---

## 目录

- [概述](#概述)
- [核心架构](#核心架构)
- [模块结构](#模块结构)
- [核心组件](#核心组件)
- [工作流程](#工作流程)
- [使用示例](#使用示例)
- [扩展指南](#扩展指南)

---

## 概述

Game 模块是一个**通用的多轮游戏管理框架**，专为房间类游戏（如麻将、斗地主等）设计。它提供了：

- **多轮游戏管理**：支持多局游戏，自动累计分数
- **单局游戏抽象**：Round 抽象层，隔离具体游戏逻辑
- **任务调度系统**：支持超时处理、自动托管
- **引擎插件化**：通过接口实现不同游戏类型
- **并发安全**：完善的锁机制保证线程安全

### 设计理念

1. **分层架构**：GameManager → MultiRoundGame → Round → RoundEngine
2. **接口驱动**：通过 `RoundEngine` 接口支持不同游戏类型
3. **职责分离**：游戏管理、轮次控制、引擎逻辑各司其职
4. **非阻塞设计**：所有操作异步处理，不阻塞主流程

---

## 核心架构

```
┌─────────────────────────────────────────────────────────────┐
│                        GameManager                          │
│  - 管理所有房间的游戏实例                                      │
│  - LRU 淘汰机制                                               │
│  - 广播游戏事件                                               │
└────────────────────┬────────────────────────────────────────┘
                     │ 1:N
                     ▼
┌─────────────────────────────────────────────────────────────┐
│                     MultiRoundGame                          │
│  - 管理单个房间的多轮游戏                                      │
│  - 累计玩家分数                                               │
│  - 判断游戏结束条件                                            │
└────────────────────┬────────────────────────────────────────┘
                     │ 1:1 (当前轮次)
                     ▼
┌─────────────────────────────────────────────────────────────┐
│                          Round                              │
│  - 单局游戏抽象                                               │
│  - 任务超时管理                                               │
│  - 操作串行化 (opMu)                                          │
└────────────────────┬────────────────────────────────────────┘
                     │ 1:1
                     ▼
┌─────────────────────────────────────────────────────────────┐
│                      RoundEngine                            │
│  接口：Initialize, HandleAction, GetState, IsRoundOver      │
│                                                             │
│  实现：HTMahjongEngine, THMahjongEngine, ...                │
└─────────────────────────────────────────────────────────────┘
```

---

## 模块结构

```
internal/game/
├── manager.go              # GameManager - 游戏管理器
├── multi_round_game.go     # MultiRoundGame - 多轮游戏
├── round.go                # Round - 单局游戏抽象
├── constants.go            # 游戏类型常量
├── errors.go               # 错误定义
├── status.go               # 状态定义
├── types/
│   └── settlement.go       # 结算数据结构
└── mahjong/
    ├── core/               # 麻将核心引擎
    │   ├── engine.go       # 通用麻将引擎
    │   ├── interfaces.go   # 接口定义
    │   ├── types.go        # 数据类型
    │   └── tile.go         # 牌相关
    ├── htmajong/           # 会同麻将实现
    │   ├── engine.go       # 引擎适配
    │   ├── actions.go      # 动作处理
    │   ├── judge.go        # 任务判断
    │   ├── winning.go      # 胡牌算法
    │   └── settlement.go   # 结算逻辑
    └── thmahjong/          # 太湖麻将实现
        └── ...
```

---

## 核心组件

### 1. GameManager（游戏管理器）

**职责**：
- 管理所有房间的游戏实例（`sync.Map`）
- 创建和销毁游戏
- 广播游戏事件给房间玩家
- LRU 淘汰不活跃游戏

**关键方法**：
```go
// 启动游戏（统一入口）
func (m *GameManager) StartGame(ctx context.Context, room *model.Room) error

// 处理玩家操作
func (m *GameManager) HandlePlayerAction(ctx context.Context, roomID string, userID int64, action core.Action) error

// 广播游戏事件
func (m *GameManager) BroadcastGameEvent(ctx context.Context, roomId string, event string, data interface{}) error

// 发送个性化事件（每个玩家看到不同内容）
func (m *GameManager) SendPersonalizedGameEvents(ctx context.Context, roomId string, event string, userDataMap map[int64]interface{}) error
```

**生命周期**：
1. 应用启动时创建 `GameManager`
2. 房间开始游戏时调用 `StartGame`
3. 玩家操作时调用 `HandlePlayerAction`
4. 游戏结束或超时后自动淘汰

---

### 2. MultiRoundGame（多轮游戏）

**职责**：
- 管理单个房间的多局游戏
- 累计玩家总分
- 判断游戏结束条件（达到最大局数或封顶分数）
- 持久化游戏记录

**关键字段**：
```go
type MultiRoundGame struct {
    ID           string              // 游戏ID
    RoomID       string              // 房间ID
    GameType     string              // 游戏类型
    Config       GameConfig          // 游戏配置
    currentRound *Round              // 当前轮次
    roundNumber  int                 // 当前轮次号
    playerScores map[int64]int       // 玩家总分
    status       MultiRoundGameStatus // 游戏状态
}
```

**关键方法**：
```go
// 开始新的一局
func (g *MultiRoundGame) StartNewRound(ctx context.Context, playerIDs []int64, engineFactory func() RoundEngine) (*Round, error)

// 完成当前局
func (g *MultiRoundGame) FinishCurrentRound(ctx context.Context) error

// 获取玩家总分
func (g *MultiRoundGame) GetPlayerScores() map[int64]int
```

**游戏结束条件**：
- 达到最大局数（`Config.MaxRounds`）
- 有玩家达到封顶分数（`Config.MaxScore`）

---

### 3. Round（单局游戏）

**职责**：
- 单局游戏的抽象层
- 操作串行化（`opMu` 保证同一时刻只有一个操作）
- 任务超时管理（通过 `task.Scheduler`）
- 状态管理（Initializing → Playing → Settling → Finished）

**关键字段**：
```go
type Round struct {
    opMu        sync.Mutex   // 操作级互斥锁（串行化）
    mu          sync.RWMutex // 元数据保护锁

    ID          string
    GameID      string
    RoundNumber int

    engine      RoundEngine  // 游戏引擎
    scheduler   *task.Scheduler
    taskVersion int64        // 任务版本号（用于失效旧任务）

    status      RoundStatus
    settlement  interface{}  // 结算结果
}
```

**关键方法**：
```go
// 初始化单局
func (r *Round) Initialize(ctx context.Context, playerIDs []string) error

// 处理玩家操作
func (r *Round) HandlePlayerAction(ctx context.Context, userId int64, action core.Action) error

// 获取游戏状态
func (r *Round) GetState() *core.GameState

// 任务超时回调
func (r *Round) onTaskTimeout(taskVersion int64)
```

**并发安全机制**：
- `opMu`：保证操作串行化，避免并发修改游戏状态
- `mu`：保护元数据字段（status、taskVersion 等）
- `taskVersion`：递增版本号，使旧任务失效

---

### 4. RoundEngine（游戏引擎接口）

**职责**：
- 定义单局游戏的标准接口
- 由具体游戏类型实现（如 HTMahjongEngine）

**接口定义**：
```go
type RoundEngine interface {
    // 初始化单局游戏
    Initialize(ctx context.Context, playerIDs []string) error

    // 处理玩家动作
    HandleAction(ctx context.Context, playerID string, action core.Action) error

    // 获取游戏状态
    GetState() *core.GameState

    // 检查单局是否结束
    IsRoundOver() bool

    // 获取结算结果
    GetSettlement() interface{}

    // 处理任务超时
    ProcessTaskTimeout()
}
```

**实现示例**：
- `htmajong.Engine`：会同麻将引擎
- `thmahjong.Engine`：太湖麻将引擎

---

## 工作流程

### 1. 游戏启动流程

```
用户点击"开始游戏"
    ↓
RoomHandler.handleStartGame
    ↓
GameManager.StartGame(room)
    ↓
1. 创建 MultiRoundGame
2. 调用 game.StartNewRound(playerIDs, engineFactory)
    ↓
    2.1 创建 Round
    2.2 创建 RoundEngine（通过 engineFactory）
    2.3 调用 round.Initialize(playerIDs)
        ↓
        2.3.1 engine.Initialize(playerIDs)
        2.3.2 发牌、初始化玩家状态
        2.3.3 状态切换：Initializing → Playing
    ↓
3. 广播 GAME_STARTED 事件
```

### 2. 玩家操作流程

```
客户端发送操作（如出牌）
    ↓
RoomHandler.HandleGameRequest
    ↓
GameManager.HandlePlayerAction(roomID, userID, action)
    ↓
1. 获取 MultiRoundGame
2. 获取 currentRound
3. 调用 round.HandlePlayerAction(userID, action)
    ↓
    3.1 opMu.Lock()（串行化）
    3.2 递增 taskVersion（使旧任务失效）
    3.3 engine.HandleAction(playerID, action)
        ↓
        3.3.1 验证动作合法性
        3.3.2 执行动作（修改游戏状态）
        3.3.3 判断是否产生新任务（碰、杠、胡）
        3.3.4 广播状态变化
    3.4 检查是否结束
        - 如果结束：状态切换为 Settling
    3.5 如果有新任务：调度超时任务
    3.6 opMu.Unlock()
    ↓
4. 如果 Round 结束：调用 game.FinishCurrentRound()
    ↓
    4.1 获取结算结果
    4.2 更新玩家总分
    4.3 标记 Round 为 Finished
    4.4 检查游戏是否结束
```

### 3. 任务超时流程

```
玩家未在规定时间内响应任务
    ↓
task.Scheduler 触发超时回调
    ↓
round.onTaskTimeout(taskVersion)
    ↓
1. opMu.Lock()（串行化）
2. 检查 taskVersion 是否匹配
    - 不匹配：任务已失效，直接返回
3. 调用 engine.ProcessTaskTimeout()
    ↓
    3.1 自动选择默认动作（如"过"）
    3.2 继续游戏流程
4. 检查是否结束
5. opMu.Unlock()
```

---

## 使用示例

### 示例 1：启动游戏

```go
// 在 RoomHandler 中
func (h *RoomHandler) handleStartGame(ctx context.Context, req *proto.RoomRequest, ...) error {
    // 1. 验证房间状态
    roomInfo, err := h.roomService.StartGame(ctx, room.StartGameParams{
        UserId: req.UserId,
        RoomId: req.RoomId,
    })
    if err != nil {
        return err
    }

    // 2. 启动游戏
    if err := h.gameManager.StartGame(ctx, roomInfo); err != nil {
        return err
    }

    return nil
}
```

### 示例 2：处理玩家操作

```go
// 在 RoomHandler 中
func (h *RoomHandler) handleMahjongGame(ctx context.Context, req *proto.GameRequest, ...) error {
    // 1. 解析动作
    action := core.Action{
        Type:     core.ActionDiscard,
        PlayerID: fmt.Sprintf("%d", req.UserId),
        Tile:     &core.Tile{Suit: core.TileSuitWan, Value: 1},
    }

    // 2. 处理动作
    if err := h.gameManager.HandlePlayerAction(ctx, req.RoomId, req.UserId, action); err != nil {
        return err
    }

    return nil
}
```

### 示例 3：广播游戏事件

```go
// 在引擎内部
func (e *Engine) broadcastStateChange(ctx context.Context) {
    state := e.GetState()

    // 方式1：广播相同内容给所有玩家
    gameManager.BroadcastGameEvent(ctx, roomID, "STATE_CHANGED", state)

    // 方式2：发送个性化内容（每个玩家看到不同手牌）
    userDataMap := make(map[int64]interface{})
    for _, player := range state.Players {
        userID, _ := strconv.ParseInt(player.ID, 10, 64)
        userDataMap[userID] = map[string]interface{}{
            "hand":     player.Hand,      // 只发送自己的手牌
            "discards": state.Discards,   // 公共信息
        }
    }
    gameManager.SendPersonalizedGameEvents(ctx, roomID, "HAND_UPDATED", userDataMap)
}
```

---

## 扩展指南

### 如何添加新的游戏类型

#### 1. 定义游戏类型常量

```go
// constants.go
const (
    GameTypePoker = "POKER" // 扑克游戏
)

var gameTypeInternalMap = map[string]string{
    GameTypePoker: "poker",
}
```

#### 2. 实现 RoundEngine 接口

```go
// poker/engine.go
package poker

import (
    "context"
    "sudooom.im.logic/internal/game/mahjong/core"
)

type PokerEngine struct {
    state *PokerGameState
}

func NewEngine() *PokerEngine {
    return &PokerEngine{}
}

func (e *PokerEngine) Initialize(ctx context.Context, playerIDs []string) error {
    // 初始化扑克游戏
    e.state = &PokerGameState{
        Players: make([]*PokerPlayer, len(playerIDs)),
    }
    // 发牌逻辑
    return nil
}

func (e *PokerEngine) HandleAction(ctx context.Context, playerID string, action core.Action) error {
    // 处理扑克动作（出牌、跟注、弃牌等）
    return nil
}

func (e *PokerEngine) GetState() *core.GameState {
    // 返回游戏状态
    return e.state.ToGameState()
}

func (e *PokerEngine) IsRoundOver() bool {
    // 判断是否结束
    return e.state.IsFinished
}

func (e *PokerEngine) GetSettlement() interface{} {
    // 返回结算结果
    return e.state.Settlement
}

func (e *PokerEngine) ProcessTaskTimeout() {
    // 处理超时（自动弃牌）
}
```

#### 3. 在 GameManager 中注册

```go
// manager.go
func (m *GameManager) createEngine(gameType string) RoundEngine {
    switch gameType {
    case "HT_MAHJONG":
        return htmajong.NewEngine()
    case "TH_MAHJONG":
        return thmahjong.NewEngine()
    case "POKER":
        return poker.NewEngine() // 新增
    default:
        return htmajong.NewEngine()
    }
}
```

### 如何自定义结算逻辑

```go
// 在引擎中实现 GetSettlement
func (e *Engine) GetSettlement() interface{} {
    coreSettlement := e.Engine.GetSettlement()

    // 转换为自定义结算结构
    settlement := &types.RoundSettlement{
        RoundID:        e.roundID,
        SettlementType: "win",
        Winners:        []types.PlayerScore{},
        Losers:         []types.PlayerScore{},
    }

    // 计算分数变化
    for _, transfer := range coreSettlement.Transfers {
        // 转换逻辑
    }

    return settlement
}
```

### 如何添加任务调度

```go
// 在 Round.HandlePlayerAction 中
if len(state.PendingTasks) > 0 {
    r.mu.Lock()
    r.scheduleTaskTimeout(30) // 30秒超时
    r.mu.Unlock()
}

// 超时回调会自动触发
func (r *Round) onTaskTimeout(taskVersion int64) {
    // 检查版本号
    if taskVersion != r.taskVersion {
        return // 任务已失效
    }

    // 处理超时
    r.engine.ProcessTaskTimeout()
}
```

---

## 关键设计决策

### 1. 为什么使用 opMu 串行化操作？

**问题**：多个玩家可能同时发送操作，导致游戏状态并发修改。

**解决方案**：
- `opMu` 保证同一时刻只有一个操作在执行
- 避免复杂的状态同步逻辑
- 简化引擎实现（引擎内部不需要锁）

### 2. 为什么使用 taskVersion？

**问题**：玩家操作后，之前调度的超时任务应该失效。

**解决方案**：
- 每次操作递增 `taskVersion`
- 超时回调检查版本号，不匹配则忽略
- 避免取消任务的复杂逻辑

### 3. 为什么 Round 和 MultiRoundGame 分离？

**职责分离**：
- `Round`：单局游戏逻辑，生命周期短
- `MultiRoundGame`：多局管理，累计分数，生命周期长

**好处**：
- Round 结束后可以释放内存
- 历史轮次可以持久化到数据库
- 便于实现游戏回放

### 4. 为什么使用 interface{} 返回结算结果？

**避免循环依赖**：
- `game` 包不应该依赖 `mahjong` 包
- 使用 `interface{}` + 类型断言解耦

**替代方案**：
- 可以定义通用的 `Settlement` 接口
- 各游戏类型实现该接口

---

## 性能优化建议

### 1. LRU 淘汰策略

```go
// 配置
maxGames:     1000,              // 最多缓存 1000 个游戏
evictTimeout: 30 * time.Minute,  // 30 分钟不活跃则淘汰
```

### 2. 并行广播

```go
// SendPersonalizedGameEvents 内部使用 goroutine 并行发送
var wg sync.WaitGroup
for userId, data := range userDataMap {
    wg.Add(1)
    go func(uid int64, d interface{}) {
        defer wg.Done()
        // 发送消息
    }(userId, data)
}
wg.Wait()
```

### 3. 状态快照

```go
// 避免频繁序列化，缓存状态快照
type Engine struct {
    state         *GameState
    stateSnapshot []byte // 缓存的 JSON
    stateDirty    bool   // 是否需要重新序列化
}
```

---

## 常见问题

### Q1: 如何处理断线重连？

**方案**：
1. 玩家重连后，从 `GameManager` 获取游戏实例
2. 调用 `round.GetState()` 获取当前状态
3. 发送完整状态给客户端

### Q2: 如何实现游戏回放？

**方案**：
1. 在 `MultiRoundGame.FinishCurrentRound` 中持久化 Round 记录
2. 保存每个动作的时间戳和状态快照
3. 回放时按时间顺序重放动作

### Q3: 如何支持观战？

**方案**：
1. 在 `GameManager.BroadcastGameEvent` 中添加观战者列表
2. 观战者只接收公共信息（不包含手牌）
3. 使用 `SendPersonalizedGameEvents` 区分玩家和观战者

### Q4: 如何处理作弊检测？

**方案**：
1. 在 `ActionHandler.ValidateAction` 中验证动作合法性
2. 记录所有动作日志（时间戳、玩家、动作）
3. 异步分析日志，检测异常模式

---

## 总结

Game 模块是一个**通用的多轮游戏框架**，核心优势：

1. **分层清晰**：GameManager → MultiRoundGame → Round → RoundEngine
2. **并发安全**：完善的锁机制，避免状态竞争
3. **可扩展**：通过接口支持不同游戏类型
4. **高性能**：LRU 淘汰、并行广播、任务调度

**适用场景**：
- 房间类多轮游戏（麻将、斗地主、德州扑克等）
- 需要累计分数的游戏
- 需要任务超时处理的游戏

**不适用场景**：
- 实时对战游戏（FPS、MOBA）
- 单局游戏（不需要多轮管理）
- 无状态游戏

---

**最后更新**: 2026-02-14
