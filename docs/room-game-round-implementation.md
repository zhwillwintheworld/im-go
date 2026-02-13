# Room-Game-Round 架构实现文档

## 1. 架构概览

本文档描述了游戏系统的三层架构实现：Room（房间）→ MultiRoundGame（多轮游戏）→ Round（单局游戏）。

```
┌─────────────────────────────────────────────────────────────┐
│                          Room (房间)                          │
│  - 社交容器，管理玩家和房间状态                                  │
│  - 持有当前活跃的 MultiRoundGame (1:1 关系)                    │
│  - 状态：waiting ⇄ playing                                    │
└─────────────────────────────────────────────────────────────┘
                              │
                              │ 1:1 (可切换)
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    MultiRoundGame (多轮游戏)                  │
│  - 管理多个 Round，统计总分                                     │
│  - 判断游戏结束条件（最大局数/封顶分数）                          │
│  - 状态：playing → finished                                   │
└─────────────────────────────────────────────────────────────┘
                              │
                              │ 1:N
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                        Round (单局游戏)                        │
│  - 管理单局状态和任务                                           │
│  - 委托 RoundEngine 执行具体逻辑                               │
│  - 状态：created → playing → settling → finished              │
└─────────────────────────────────────────────────────────────┘
                              │
                              │ 1:1
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    RoundEngine (游戏引擎)                      │
│  - 具体游戏规则实现（会同麻将/太湖麻将等）                        │
│  - 动作验证、执行、胡牌判断、番型计算                             │
└─────────────────────────────────────────────────────────────┘
```

## 2. 核心数据结构

### 2.1 Room (房间)

**文件位置**: `internal/room/room.go`

```go
type Room struct {
    RoomID          string
    RoomName        string
    RoomType        string  // public/private
    Status          RoomStatus  // waiting/playing/finished
    Players         []*Player
    MaxPlayers      int
    GameType        string
    GameSettings    map[string]string
    CurrentGameID   string  // 当前活跃的 MultiRoundGame ID

    mu sync.RWMutex
}
```

**职责**：
- 管理玩家加入/离开/准备
- 座位分配和房间配置
- 持有当前活跃的 Game（通过 `CurrentGameID`）
- 支持 Game 切换（结束后可开始新 Game）

**状态转换**：
```
waiting → playing (开始游戏)
playing → waiting (游戏结束)
```

**关键方法**：
- `SetCurrentGame(gameID string)` - 设置当前游戏
- `FinishGame()` - 游戏结束，回到 waiting 状态

### 2.2 MultiRoundGame (多轮游戏)

**文件位置**: `internal/game/multi_round_game.go`

```go
type MultiRoundGame struct {
    ID       string
    RoomID   string
    GameType string
    Config   GameConfig

    // 轮次管理
    rounds       []*Round  // 所有轮次
    currentRound *Round    // 当前轮次
    roundNumber  int       // 当前轮次号

    // 分数统计
    playerScores map[int64]int  // 玩家ID → 总分（累计）

    // 游戏状态
    status    MultiRoundGameStatus  // playing/finished
    startTime time.Time
    endTime   time.Time

    // 依赖
    scheduler *task.Scheduler

    mu sync.RWMutex
}

type GameConfig struct {
    MaxRounds int           // 最大局数（0 表示无限制）
    MaxScore  int           // 封顶分数（0 表示无限制）
    BaseScore int           // 底分
    TimeLimit time.Duration // 单局时间限制
    Extra     map[string]any
}
```

**职责**：
- 管理多个 Round（1:N 关系）
- 轮次切换和总分统计
- 游戏结束判断（达到局数/分数上限）
- Game 结束时结算所有 Round 总分

**状态转换**：
```
playing → finished (达到结束条件)
```

**关键方法**：
- `StartNewRound(ctx, playerIDs, engineFactory)` - 开始新一局
- `FinishCurrentRound(ctx)` - 完成当前局，更新总分
- `shouldGameEnd()` - 检查是否应该结束游戏
- `GetPlayerScores()` - 获取玩家总分

**结束条件**：
1. 达到最大局数：`roundNumber >= MaxRounds`
2. 有玩家达到封顶分数：`playerScore >= MaxScore`

### 2.3 Round (单局游戏)

**文件位置**: `internal/game/round.go`

```go
type Round struct {
    ID          string
    GameID      string
    RoundNumber int

    // 游戏引擎
    engine RoundEngine

    // 任务管理
    scheduler   *task.Scheduler
    taskVersion int64  // 任务版本号（用于标记清除）

    // 状态
    status     RoundStatus  // created/playing/settling/finished
    settlement interface{}  // 结算结果

    opMu sync.Mutex    // 操作序列化锁
    mu   sync.RWMutex  // 状态读写锁
}

type RoundStatus int
const (
    RoundStatusCreated RoundStatus = iota
    RoundStatusPlaying
    RoundStatusSettling
    RoundStatusFinished
)
```

**职责**：
- 单局游戏状态管理
- 任务管理（碰/杠/胡响应）
- 单局结算
- 委托 Engine 执行具体逻辑

**状态转换**：
```
created → playing (初始化完成)
playing → settling (有人胡牌/流局)
settling → finished (结算完成)
```

**关键方法**：
- `Initialize(ctx, playerIDs)` - 初始化单局
- `HandlePlayerAction(ctx, userID, action)` - 处理玩家动作
- `ScheduleTaskTimeout(taskID, timeout, callback)` - 调度任务超时
- `Finish()` - 完成单局

**任务管理**：
使用版本号标记清除算法：
- 每次玩家操作时递增 `taskVersion`
- 任务回调时检查版本号是否匹配
- 不匹配则说明任务已过期，直接忽略

### 2.4 RoundEngine (游戏引擎接口)

**文件位置**: `internal/game/round.go`

```go
type RoundEngine interface {
    // 初始化游戏
    Initialize(ctx context.Context, playerIDs []string) error

    // 处理玩家动作
    HandleAction(ctx context.Context, userID int64, action interface{}) error

    // 处理任务超时
    ProcessTaskTimeout(ctx context.Context) error

    // 获取游戏状态
    GetState() interface{}

    // 获取结算结果
    GetSettlement() interface{}

    // 关闭引擎
    Close()
}
```

**实现**：
- `SafeMahjongEngine` - 线程安全的麻将引擎包装器
  - 文件位置：`internal/game/mahjong/safe_engine.go`
  - 包装了具体的麻将引擎（会同麻将/太湖麻将）

## 3. 服务层设计

### 3.1 GameService (游戏启动服务)

**文件位置**: `internal/game/service.go`

```go
type GameService struct {
    redisClient   *redis.Client
    routerService *service.RouterService
    gameStarters  map[string]GameStarter  // 游戏类型 → 启动器
}
```

**职责**：
- 注册和调用 GameStarter（如 MahjongService）
- 广播游戏事件给房间所有玩家
- 发送个性化游戏事件

**关键方法**：
- `RegisterGameStarter(gameType, starter)` - 注册游戏启动器
- `StartGame(ctx, room)` - 启动游戏（统一入口）
- `BroadcastGameEvent(ctx, roomId, event, data)` - 广播事件
- `SendPersonalizedGameEvents(ctx, roomId, event, userDataMap)` - 发送个性化事件

### 3.2 MultiRoundGameService (多轮游戏服务)

**文件位置**: `internal/game/multi_round_service.go`

```go
type MultiRoundGameService struct {
    gameManager *MultiRoundGameManager
}
```

**职责**：
- 管理 MultiRoundGame 和 Round 的生命周期
- 协调轮次切换
- 处理玩家动作

**关键方法**：
- `CreateGame(ctx, roomID, gameType, config)` - 创建游戏
- `StartNewRound(ctx, game, playerIDs)` - 开始新一局
- `FinishRound(ctx, game)` - 完成当前局
- `HandlePlayerAction(ctx, roomID, userID, action)` - 处理玩家动作

### 3.3 MultiRoundGameManager (多轮游戏管理器)

**文件位置**: `internal/game/multi_round_manager.go`

```go
type MultiRoundGameManager struct {
    games     sync.Map  // roomID -> *MultiRoundGame
    gameCount atomic.Int64

    maxGames     int
    evictTimeout time.Duration
    scheduler    *task.Scheduler
}
```

**职责**：
- 管理所有 MultiRoundGame 实例
- LRU 淘汰不活跃的游戏
- 优雅关闭时保存脏数据

**关键方法**：
- `CreateGame(roomID, gameType, config)` - 创建游戏
- `Get(roomID)` - 获取游戏
- `Remove(roomID)` - 移除游戏
- `evictInactive()` - 淘汰不活跃的游戏
- `Shutdown(ctx)` - 优雅关闭

### 3.4 RoomService (房间服务)

**文件位置**: `internal/room/service.go`

```go
type RoomService struct {
    roomManager   *RoomManager
    redisClient   *redis.Client
    snowflake     *snowflake.Node
    routerService *service.RouterService
}
```

**职责**：
- 房间生命周期管理
- 玩家操作处理（加入/离开/准备/换座位）
- 开始游戏的验证和状态更新

**关键方法**：
- `CreateRoom(ctx, params)` - 创建房间
- `JoinRoom(ctx, params)` - 加入房间
- `LeaveRoom(ctx, params)` - 离开房间
- `ReadyRoom(ctx, params)` - 准备/取消准备
- `StartGame(ctx, params)` - 开始游戏

## 4. 数据流程

### 4.1 开始游戏流程

```
1. 用户点击"开始游戏"
   ↓
2. RoomHandler.StartGameHandler
   ↓
3. RoomService.StartGame()
   - 验证房间状态
   - 验证玩家准备状态
   - 更新房间状态为 playing
   ↓
4. GameService.StartGame()
   - 查找游戏类型对应的 GameStarter
   - 调用 GameStarter.StartGameByType()
   ↓
5. MahjongService.StartGameByType()
   - 创建 MultiRoundGame (通过 MultiRoundGameManager)
   - 开始第一局 Round
   - 创建并设置 RoundEngine
   - 初始化游戏状态
   ↓
6. GameService.BroadcastGameEvent()
   - 广播游戏开始事件给所有玩家
```

### 4.2 单局游戏流程

```
1. Round.Initialize()
   - 创建 RoundEngine
   - 调用 engine.Initialize()
   - 发牌、确定庄家等
   ↓
2. 玩家出牌/碰/杠/胡
   ↓
3. Round.HandlePlayerAction()
   - 递增 taskVersion（使旧任务失效）
   - 调用 engine.HandleAction()
   - 检查是否有任务需要响应
   ↓
4. 如果有任务（如有人打出可以碰/杠/胡的牌）
   - Round.ScheduleTaskTimeout()
   - 等待其他玩家响应或超时
   ↓
5. 任务超时或所有玩家响应
   - Round.ProcessTaskTimeout()
   - 选择最高优先级动作执行
   ↓
6. 游戏结束（有人胡牌/流局）
   - engine.GetSettlement() 获取结算
   - Round 状态变为 settling
   ↓
7. MultiRoundGame.FinishCurrentRound()
   - 更新玩家总分
   - 持久化 Round 记录
   - 检查是否达到游戏结束条件
   ↓
8. 如果游戏未结束
   - MultiRoundGame.StartNewRound()
   - 开始下一局
   ↓
9. 如果游戏结束
   - MultiRoundGame 状态变为 finished
   - 持久化 Game 总分记录
   - Room 状态回到 waiting
```

### 4.3 任务响应流程（以碰/杠/胡为例）

```
1. 玩家 A 打出一张牌
   ↓
2. Engine 检测到玩家 B、C 可以碰/杠/胡
   ↓
3. Engine 创建 Task，返回需要响应的玩家列表
   ↓
4. Round.ScheduleTaskTimeout()
   - 捕获当前 taskVersion
   - 调度超时任务到 task.Scheduler
   ↓
5. 玩家 B 点击"碰"
   ↓
6. Round.HandlePlayerAction()
   - 递增 taskVersion（使超时任务失效）
   - 提交 Action 到 Engine
   ↓
7. Engine 收集所有响应
   - 如果所有玩家都响应了，立即处理
   - 否则等待超时
   ↓
8. 超时回调触发
   - 检查 taskVersion 是否匹配
   - 如果不匹配，说明任务已被新动作取消，直接返回
   - 如果匹配，调用 engine.ProcessTaskTimeout()
   ↓
9. Engine 选择最高优先级动作执行
   - 胡(100) > 杠(50) > 碰(30) > 吃(20) > 过(0)
```

## 5. 并发控制

### 5.1 锁的使用

**Room**:
- `sync.RWMutex` - 保护房间状态和玩家列表

**MultiRoundGame**:
- `sync.RWMutex` - 保护游戏状态、轮次列表、分数统计

**Round**:
- `opMu sync.Mutex` - 操作序列化锁，确保动作按顺序处理
- `mu sync.RWMutex` - 状态读写锁，保护状态和结算结果

**SafeMahjongEngine**:
- `sync.RWMutex` - 保护引擎状态

### 5.2 任务版本号机制

为了避免显式取消任务（从调度器中移除），使用版本号标记清除算法：

```go
// Round 结构
type Round struct {
    taskVersion int64  // 当前任务版本号
    // ...
}

// 调度任务时捕获版本号
func (r *Round) ScheduleTaskTimeout(taskID string, timeout time.Duration, callback func()) {
    currentVersion := r.taskVersion  // 捕获当前版本

    r.scheduler.Schedule(timeout, func() {
        r.mu.RLock()
        if r.taskVersion != currentVersion {
            // 版本号不匹配，任务已过期
            r.mu.RUnlock()
            return
        }
        r.mu.RUnlock()

        // 执行回调
        callback()
    })
}

// 玩家操作时递增版本号
func (r *Round) HandlePlayerAction(ctx context.Context, userID int64, action interface{}) error {
    r.opMu.Lock()
    defer r.opMu.Unlock()

    r.mu.Lock()
    r.taskVersion++  // 递增版本号，使旧任务失效
    r.mu.Unlock()

    // 处理动作...
}
```

**优点**：
- 不需要显式取消任务
- 避免了从调度器中移除任务的复杂性
- 简单高效

## 6. 结算数据结构

### 6.1 RoundSettlement (单局结算)

**文件位置**: `internal/game/types/settlement.go`

```go
type PlayerScore struct {
    PlayerID    int64
    ScoreChange int     // 分数变化（正数为赢，负数为输）
    Role        string  // 角色（winner/loser/dealer等）
}

type RoundSettlement struct {
    RoundID     string
    RoundNumber int
    Winners     []PlayerScore  // 赢家列表（支持多赢家）
    Losers      []PlayerScore  // 输家列表（支持多输家）
    Details     map[string]interface{}  // 详细信息（番型、牌型等）
}
```

**特点**：
- 支持多赢家、多输家（如一炮多响）
- 记录详细的输赢信息
- 存储到数据库供查询

### 6.2 GameSettlement (游戏总结算)

游戏结束时，`MultiRoundGame` 持有的 `playerScores` 就是总分：

```go
type MultiRoundGame struct {
    playerScores map[int64]int  // 玩家ID → 总分（累计）
}
```

持久化时保存：
- GameID
- RoomID
- 玩家总分列表
- 开始时间、结束时间
- 总局数

## 7. 持久化设计

### 7.1 Round 持久化

**时机**：每局结束时

**数据**：
```go
type RoundRecord struct {
    RoundID     string
    GameID      string
    RoundNumber int
    Winners     []PlayerScore
    Losers      []PlayerScore
    Details     map[string]interface{}
    StartTime   time.Time
    EndTime     time.Time
}
```

### 7.2 Game 持久化

**时机**：游戏结束时

**数据**：
```go
type GameRecord struct {
    GameID       string
    RoomID       string
    GameType     string
    PlayerScores map[int64]int  // 玩家总分
    TotalRounds  int
    StartTime    time.Time
    EndTime      time.Time
}
```

### 7.3 查询历史

Room 不持有历史 Game，通过数据库查询：

```sql
-- 查询房间的游戏历史
SELECT * FROM game_records WHERE room_id = ? ORDER BY start_time DESC;

-- 查询某场游戏的所有局
SELECT * FROM round_records WHERE game_id = ? ORDER BY round_number;
```

## 8. 错误处理

### 8.1 常见错误

**文件位置**: `internal/game/errors.go`, `internal/room/errors.go`

```go
// Game 相关错误
var (
    ErrGameNotFound      = errors.New("game not found")
    ErrGameAlreadyExists = errors.New("game already exists")
    ErrGameFinished      = errors.New("game is finished")
    ErrMaxGamesReached   = errors.New("max games limit reached")
    ErrRoundNotFound     = errors.New("round not found")
    ErrMaxRoundsReached  = errors.New("max rounds reached")
)

// Room 相关错误
var (
    ErrRoomNotFound     = errors.New("room not found")
    ErrRoomFull         = errors.New("room is full")
    ErrRoomBusy         = errors.New("room is busy")
    ErrGameStarted      = errors.New("game already started")
    ErrGameNotStarted   = errors.New("game not started")
    ErrInvalidPassword  = errors.New("invalid password")
    ErrAlreadyInRoom    = errors.New("already in room")
)
```

### 8.2 错误处理原则

1. **服务层捕获错误**：在 Service 层捕获并记录日志
2. **返回友好错误**：向客户端返回可读的错误信息
3. **不阻塞流程**：Handler 层捕获错误后不阻塞消息处理
4. **事务回滚**：数据库操作失败时回滚状态

## 9. 扩展性设计

### 9.1 支持新游戏类型

1. 实现 `RoundEngine` 接口
2. 创建对应的 GameStarter
3. 注册到 GameService

```go
// 实现新游戏引擎
type MyGameEngine struct {
    // ...
}

func (e *MyGameEngine) Initialize(ctx context.Context, playerIDs []string) error {
    // 实现初始化逻辑
}

// 实现其他接口方法...

// 创建 GameStarter
type MyGameService struct {
    // ...
}

func (s *MyGameService) StartGameByType(ctx context.Context, room *model.Room, internalGameType string) error {
    // 创建 MultiRoundGame
    // 开始第一局 Round
    // 设置 MyGameEngine
}

// 注册
gameService.RegisterGameStarter("MY_GAME", myGameService)
```

### 9.2 支持不同结算规则

通过 `GameConfig.Extra` 传递自定义配置：

```go
config := GameConfig{
    MaxRounds: 8,
    MaxScore:  1000,
    BaseScore: 10,
    Extra: map[string]any{
        "doubleScore": true,
        "allowRobKong": false,
    },
}
```

### 9.3 支持观战模式

在 Room 中添加观战者列表：

```go
type Room struct {
    Players   []*Player
    Observers []*Observer  // 观战者
}
```

广播事件时同时发送给观战者。

## 10. 性能优化

### 10.1 内存管理

- **LRU 淘汰**：`MultiRoundGameManager` 定期淘汰不活跃的游戏
- **对象池**：可以为频繁创建的对象（如 Action、Task）使用 `sync.Pool`

### 10.2 并发优化

- **读写锁**：大量读操作使用 `sync.RWMutex`
- **无锁数据结构**：`sync.Map` 用于高并发场景
- **原子操作**：计数器使用 `atomic.Int64`

### 10.3 任务调度优化

- **时间轮算法**：`task.Scheduler` 使用时间轮实现高效的定时任务调度
- **版本号机制**：避免显式取消任务的开销

## 11. 测试建议

### 11.1 单元测试

- 测试 Round 的状态转换
- 测试 MultiRoundGame 的结束条件判断
- 测试任务版本号机制
- 测试并发安全性

### 11.2 集成测试

- 测试完整的游戏流程（开始 → 多局 → 结束）
- 测试玩家中途离开的情况
- 测试任务超时处理
- 测试并发玩家操作

### 11.3 压力测试

- 测试大量房间同时游戏
- 测试 LRU 淘汰机制
- 测试内存占用和 GC 压力

## 12. 未来改进方向

1. **持久化实现**：完成 Round 和 Game 的数据库持久化
2. **断线重连**：支持玩家断线后重新连接并恢复游戏状态
3. **回放功能**：记录所有动作，支持游戏回放
4. **AI 托管**：玩家离线时由 AI 代为操作
5. **排行榜**：基于 Game 总分统计排行榜
6. **成就系统**：基于游戏数据统计成就
7. **观战系统**：完善观战功能，支持延迟观战
8. **录像分享**：支持分享游戏录像

## 13. 常见问题

### Q1: Room 和 MultiRoundGame 是什么关系？

A: Room 和 MultiRoundGame 是 1:1 关系，但可以切换。一个 Room 同时只有一个活跃的 MultiRoundGame，当 Game 结束后，Room 可以开始新的 Game。

### Q2: 为什么不在 Room 中保存历史 Game？

A: 为了避免内存占用过大，历史 Game 在结束时持久化到数据库，通过查询获取。

### Q3: 任务版本号机制如何工作？

A: 每次玩家操作时递增 `taskVersion`，任务回调时检查版本号是否匹配。不匹配说明任务已过期，直接忽略。这样避免了显式取消任务的复杂性。

### Q4: 如何支持一炮多响？

A: `RoundSettlement` 的 `Winners` 和 `Losers` 都是数组，支持多个赢家和输家。Engine 在结算时填充多个玩家的分数变化。

### Q5: 如何扩展支持新游戏？

A: 实现 `RoundEngine` 接口，创建对应的 GameStarter，然后注册到 GameService 即可。

### Q6: 并发安全如何保证？

A: 使用多层锁保护：
- Room 使用 `sync.RWMutex` 保护状态
- MultiRoundGame 使用 `sync.RWMutex` 保护状态
- Round 使用 `opMu` 序列化操作，使用 `mu` 保护状态
- Engine 使用 `sync.RWMutex` 保护状态

### Q7: 如何处理玩家中途离开？

A: 当前设计中，玩家离开会导致游戏异常结束。未来可以支持 AI 托管或者允许其他玩家加入。

---

**文档版本**: v1.0
**最后更新**: 2026-02-13
**维护者**: 开发团队
