# HTMajong 对象设计文档

> 本文档分析 htmajong 麻将游戏对象的设计，说明其优劣势，并提供可拓展性和可维护性的优化方案。

---

## 目录

- [1. 架构概览](#1-架构概览)
- [2. 核心对象设计分析](#2-核心对象设计分析)
- [3. 当前设计的优劣势](#3-当前设计的优劣势)
- [4. 可拓展性设计方案](#4-可拓展性设计方案)
- [5. 可维护性优化建议](#5-可维护性优化建议)
- [6. 设计模式应用](#6-设计模式应用)
- [7. 重构路线图](#7-重构路线图)

---

## 1. 架构概览

### 1.1 目录结构

```
internal/game/htmajong/
└── htmajong/
    ├── algorithm.go      # 胡牌算法（核心业务逻辑）
    ├── mahjong.go        # 麻将牌对象
    ├── seat.go           # 座位对象
    ├── table.go          # 牌桌对象
    ├── lease.go          # 租约系统（多人响应机制）
    ├── hu.go             # 胡牌类型定义
    ├── color.go          # 颜色枚举
    ├── position.go       # 位置枚举
    ├── supplier_type.go  # 供牌方式枚举
    └── task.go           # 任务类型枚举
```

### 1.2 对象关系图

```
┌─────────────┐
│   Table     │ (牌桌 - 游戏容器)
│             │
│  ┌────────┐ │
│  │ East   │ │
│  │ South  │ │
│  │ West   │ │
│  │ North  │ │
│  └────────┘ │
│             │
│  Lease      │ (租约 - 多人响应)
└─────────────┘
       │
       ├──> Seat (座位 - 玩家状态)
       │      ├── ExtraList (手牌)
       │      ├── PublicList (公开牌)
       │      └── OutList (出牌)
       │
       └──> Algorithm (算法 - 规则判断)
              ├── CheckHu (检查胡牌)
              ├── CheckPeng (检查碰)
              └── CheckGang (检查杠)
```

---

## 2. 核心对象设计分析

### 2.1 Mahjong（麻将牌）

**设计**：
```go
type Mahjong struct {
    Color  Color  // 颜色
    Number int    // 数字
    Order  int    // 顺序
}
```

**职责**：表示单张麻将牌

**优点**：
- 简单直观，数据结构清晰
- Number 使用特殊编码（1-9万，11-19条，21-29饼），便于计算

**缺点**：
- 缺少不变性保证（应该是值对象）
- Order 字段的语义不够明确（是生成顺序还是排序依据？）
- 缺少牌的合法性验证

**改进建议**：
```go
// 1. 添加牌的合法性验证
type Mahjong struct {
    color  Color  // 私有字段，通过方法访问
    number int
    order  int
}

// 2. 工厂方法保证合法性
func NewMahjong(color Color, number int, order int) (Mahjong, error) {
    if !isValidMahjong(color, number) {
        return Mahjong{}, ErrInvalidMahjong
    }
    return Mahjong{color, number, order}, nil
}

// 3. 添加比较和相等性方法
func (m Mahjong) Equals(other Mahjong) bool {
    return m.color == other.color && m.number == other.number
}

// 4. 实现 Comparable 接口
func (m Mahjong) Compare(other Mahjong) int {
    return m.number - other.number
}
```

---

### 2.2 Seat（座位）

**设计**：
```go
type Seat struct {
    User             *model.User
    Position         Position
    ExtraList        []Mahjong  // 手牌
    PublicList       []Mahjong  // 公开牌
    OutList          []Mahjong  // 出牌
    Points           int
    Step             atomic.Int32
    IsPublic         bool       // 是否报听
    IsReady          bool       // 是否听牌
    PublicWinMahjong []Mahjong  // 报听后可胡的牌
}
```

**职责**：管理玩家的游戏状态

**优点**：
- 完整封装了玩家的所有游戏数据
- 使用 atomic 保证并发安全
- 清晰区分手牌、公开牌、出牌

**缺点**：
- **数据与行为分离**：Seat 只是数据结构，缺少行为方法
- **职责过重**：同时管理玩家信息、手牌状态、报听状态
- **缺少封装**：所有字段都是公开的，外部可以随意修改
- **缺少状态验证**：没有保证状态的一致性（如手牌数量限制）

**改进建议**：

```go
// 1. 封装内部状态
type Seat struct {
    user             *model.User
    position         Position
    hand             *Hand          // 手牌管理器
    publicTiles      *PublicTiles   // 公开牌管理器
    discardPile      *DiscardPile   // 出牌堆
    points           int
    step             atomic.Int32
    listeningState   *ListeningState // 报听状态
}

// 2. 添加行为方法
func (s *Seat) DrawTile(tile Mahjong) error {
    if s.hand.Size() >= MaxHandSize {
        return ErrHandFull
    }
    s.hand.Add(tile)
    return nil
}

func (s *Seat) DiscardTile(tile Mahjong) error {
    if !s.hand.Contains(tile) {
        return ErrTileNotInHand
    }
    s.hand.Remove(tile)
    s.discardPile.Add(tile)
    return nil
}

// 3. 状态查询方法
func (s *Seat) CanDeclareReady() bool {
    return s.hand.IsReady() && s.step.Load() == 1
}

// 4. 不可变视图
func (s *Seat) GetHandView() []Mahjong {
    return s.hand.Clone() // 返回副本，防止外部修改
}
```

---

### 2.3 Table（牌桌）

**设计**：
```go
type Table struct {
    RoomID    string
    TableID   string
    East      *Seat
    South     *Seat
    West      *Seat
    North     *Seat
    Extra     []Mahjong  // 牌堆
    // ... 其他配置和状态
}
```

**职责**：游戏容器，管理4个座位和牌堆

**优点**：
- 清晰的游戏容器概念
- 包含了游戏所需的所有配置

**缺点**：
- **字段过多**（20+ 个字段），违反单一职责原则
- **缺少游戏状态机**：状态管理混乱
- **直接暴露 Seat 指针**：外部可以随意修改
- **缺少游戏流程控制**：没有回合管理

**改进建议**：

```go
// 1. 拆分配置和状态
type TableConfig struct {
    CanFireWinner        bool
    BigBigWinConfig      bool
    CompleteWinnerConfig bool
    FireWinnerConfig     int
    CanPublic            bool
}

type GameState struct {
    CurrentRound  int
    CurrentSeat   *Seat
    Phase         GamePhase  // 摸牌、出牌、响应
}

type Table struct {
    id       string
    config   TableConfig
    seats    map[Position]*Seat
    deck     *Deck          // 牌堆管理器
    state    *GameState
    lease    *LeaseManager  // 租约管理器
    history  *GameHistory   // 游戏历史
}

// 2. 添加游戏流程控制
func (t *Table) NextTurn() error {
    if err := t.validateState(); err != nil {
        return err
    }
    t.state.CurrentSeat = t.getNextSeat()
    t.state.Phase = PhaseDrawTile
    return nil
}

// 3. 封装座位访问
func (t *Table) GetSeat(pos Position) (*Seat, error) {
    seat, ok := t.seats[pos]
    if !ok {
        return nil, ErrInvalidPosition
    }
    return seat, nil
}

// 4. 游戏状态机
type GamePhase int
const (
    PhaseDrawTile GamePhase = iota
    PhaseDiscardTile
    PhaseWaitingResponse
    PhaseGameEnd
)
```

---

### 2.4 LeaseInfo（租约系统）

**设计**：处理多人同时响应的机制（如多人同时喊胡）

**优点**：
- 巧妙的设计，解决了并发响应问题
- 优先级管理（First 胡优先，Second 碰杠次之）

**缺点**：
- **命名不直观**：Lease（租约）的概念不够清晰
- **复杂度高**：逻辑复杂，难以理解
- **缺少超时机制**：没有响应超时处理

**改进建议**：

```go
// 1. 重命名为更清晰的名字
type PlayerResponseManager struct {
    trigger      *ActionTrigger
    highPriority []*Response  // 胡牌响应
    lowPriority  []*Response  // 碰、杠响应
    timeout      time.Duration
    result       *ResponseResult
}

// 2. 添加超时机制
func (m *PlayerResponseManager) WaitForResponses(ctx context.Context) (*ResponseResult, error) {
    ctx, cancel := context.WithTimeout(ctx, m.timeout)
    defer cancel()

    select {
    case <-ctx.Done():
        return nil, ErrResponseTimeout
    case result := <-m.resultChan:
        return result, nil
    }
}

// 3. 明确的优先级规则
type ResponsePriority int
const (
    PriorityWin     ResponsePriority = 100  // 胡牌最高
    PriorityKong    ResponsePriority = 50   // 杠次之
    PriorityPong    ResponsePriority = 30   // 碰再次
    PriorityChow    ResponsePriority = 10   // 吃最低
)
```

---

### 2.5 Algorithm（算法模块）

**设计**：胡牌规则判断的核心算法

**优点**：
- 算法实现正确且完整
- 已优化过，使用了辅助函数减少重复代码
- 支持多种胡牌类型

**缺点**：
- **所有算法都在一个文件中**（487 行），违反单一职责
- **缺少策略模式**：不同规则混在一起
- **难以拓展新规则**：添加新的胡牌类型需要修改多处代码
- **缺少规则组合机制**：无法灵活配置启用的规则

**改进建议**：

```go
// 1. 策略模式 - 抽象胡牌规则
type WinRule interface {
    Name() string
    Check(hand *Hand, tile Mahjong, context *GameContext) bool
    GetPoints() int
}

// 2. 实现具体规则
type ClearHandRule struct{}
func (r *ClearHandRule) Check(hand *Hand, tile Mahjong, ctx *GameContext) bool {
    return checkClearHand(hand, tile)
}

type SevenPairsRule struct{}
func (r *SevenPairsRule) Check(hand *Hand, tile Mahjong, ctx *GameContext) bool {
    return checkSevenPairs(hand, tile)
}

// 3. 规则引擎
type RuleEngine struct {
    rules []WinRule
}

func (e *RuleEngine) CheckWin(hand *Hand, tile Mahjong, ctx *GameContext) (bool, []WinRule) {
    matchedRules := make([]WinRule, 0)
    for _, rule := range e.rules {
        if rule.Check(hand, tile, ctx) {
            matchedRules = append(matchedRules, rule)
        }
    }
    return len(matchedRules) > 0, matchedRules
}

// 4. 规则配置
type RuleConfig struct {
    EnabledRules []string
    CustomRules  []WinRule
}

func NewRuleEngine(config RuleConfig) *RuleEngine {
    engine := &RuleEngine{rules: make([]WinRule, 0)}
    // 根据配置加载规则
    for _, ruleName := range config.EnabledRules {
        if rule := GetRuleByName(ruleName); rule != nil {
            engine.AddRule(rule)
        }
    }
    return engine
}
```

---

## 3. 当前设计的优劣势

### 3.1 优势

| 优势 | 说明 |
|------|------|
| **简单直观** | 结构清晰，容易理解基本逻辑 |
| **完整性** | 涵盖了麻将游戏的所有核心元素 |
| **算法正确** | 胡牌算法经过优化，逻辑正确 |
| **类型安全** | 使用枚举类型，避免魔法数字 |
| **并发支持** | 使用 atomic 类型保证并发安全 |

### 3.2 劣势

| 劣势 | 影响 | 严重程度 |
|------|------|---------|
| **职责不清** | 对象承担过多责任，难以维护 | ⚠️⚠️⚠️ 高 |
| **封装不足** | 公开字段过多，容易被误用 | ⚠️⚠️⚠️ 高 |
| **缺少抽象** | 硬编码规则，难以拓展 | ⚠️⚠️ 中 |
| **状态管理混乱** | 没有明确的状态机 | ⚠️⚠️⚠️ 高 |
| **缺少验证** | 状态变更没有一致性检查 | ⚠️⚠️ 中 |
| **测试困难** | 紧耦合，单元测试不易编写 | ⚠️⚠️ 中 |

### 3.3 技术债务分析

```
技术债务象限图：

高回报 ┆ 谨慎的债务          │ 轻率的债务
      ┆ - 先快速实现再重构  │ - 缺少设计
      ┆ ✅ 当前部分代码     │ ⚠️ Table 对象
─────┼────────────────────┼──────────────
低回报 ┆ 深思熟虑的债务      │ 无意的债务
      ┆ - 知道如何优化      │ - 不知道更好的做法
      ┆ ✅ Algorithm       │ ⚠️ Lease 系统

      深思熟虑 ──────────── 轻率
```

---

## 4. 可拓展性设计方案

### 4.1 支持不同麻将规则

**问题**：不同地区麻将规则不同（四川麻将、广东麻将等）

**方案一：策略模式 + 配置**

```go
// 1. 定义规则集接口
type MahjongRuleSet interface {
    Name() string
    GetWinRules() []WinRule
    GetScoringRules() []ScoringRule
    GetSpecialRules() []SpecialRule
}

// 2. 实现具体规则集
type SichuanMahjong struct {
    config SichuanConfig
}

func (s *SichuanMahjong) GetWinRules() []WinRule {
    return []WinRule{
        &ClearHandRule{},
        &SevenPairsRule{},
        &DragonSevenPairsRule{},
        // 四川麻将特有规则...
    }
}

type GuangdongMahjong struct {
    config GuangdongConfig
}

func (g *GuangdongMahjong) GetWinRules() []WinRule {
    return []WinRule{
        &ClearHandRule{},
        &AllPongsRule{},
        &ThirteenOrphansRule{},  // 十三幺
        // 广东麻将特有规则...
    }
}

// 3. 规则工厂
type RuleSetFactory struct {
    registry map[string]MahjongRuleSet
}

func (f *RuleSetFactory) Create(name string) (MahjongRuleSet, error) {
    ruleSet, ok := f.registry[name]
    if !ok {
        return nil, ErrUnknownRuleSet
    }
    return ruleSet, nil
}

// 4. 使用示例
func NewTable(ruleSetName string) (*Table, error) {
    ruleSet, err := ruleSetFactory.Create(ruleSetName)
    if err != nil {
        return nil, err
    }

    return &Table{
        ruleEngine: NewRuleEngine(ruleSet),
        // ...
    }, nil
}
```

**方案二：插件架构**

```go
// 1. 定义插件接口
type GamePlugin interface {
    Name() string
    Version() string
    Initialize(ctx *GameContext) error
    OnGameStart(table *Table) error
    OnTileDrawn(seat *Seat, tile Mahjong) error
    OnTileDiscarded(seat *Seat, tile Mahjong) error
}

// 2. 插件管理器
type PluginManager struct {
    plugins []GamePlugin
}

func (m *PluginManager) LoadPlugin(plugin GamePlugin) error {
    if err := plugin.Initialize(m.context); err != nil {
        return err
    }
    m.plugins = append(m.plugins, plugin)
    return nil
}

func (m *PluginManager) TriggerEvent(event GameEvent) error {
    for _, plugin := range m.plugins {
        if err := plugin.HandleEvent(event); err != nil {
            return err
        }
    }
    return nil
}

// 3. 实现自定义插件
type CustomScoringPlugin struct{}

func (p *CustomScoringPlugin) OnGameEnd(result *GameResult) error {
    // 自定义计分逻辑
    return nil
}
```

---

### 4.2 支持不同玩家数量

**问题**：当前固定4人，需要支持2人、3人麻将

**方案**：

```go
// 1. 抽象座位管理
type SeatManager struct {
    seats    []Seat
    maxSeats int
    round    int
}

func NewSeatManager(playerCount int) (*SeatManager, error) {
    if playerCount < 2 || playerCount > 4 {
        return nil, ErrInvalidPlayerCount
    }

    seats := make([]Seat, playerCount)
    return &SeatManager{
        seats:    seats,
        maxSeats: playerCount,
    }, nil
}

func (m *SeatManager) GetCurrentSeat() *Seat {
    index := m.round % m.maxSeats
    return &m.seats[index]
}

func (m *SeatManager) GetNextSeat() *Seat {
    m.round++
    return m.GetCurrentSeat()
}

// 2. Table 使用 SeatManager
type Table struct {
    id          string
    seatManager *SeatManager  // 替代 East/South/West/North
    deck        *Deck
    // ...
}

// 3. 位置枚举动态化
type Position int

func GetPositions(playerCount int) []Position {
    switch playerCount {
    case 2:
        return []Position{EAST, WEST}
    case 3:
        return []Position{EAST, SOUTH, WEST}
    case 4:
        return []Position{EAST, SOUTH, WEST, NORTH}
    default:
        return nil
    }
}
```

---

### 4.3 支持牌堆变化

**问题**：不同规则使用不同的牌（如：去掉字牌、只用万子等）

**方案**：

```go
// 1. 牌堆生成器接口
type DeckGenerator interface {
    Generate() []Mahjong
    TileCount() int
}

// 2. 标准牌堆
type StandardDeck struct {
    loop int  // 每种牌的数量（通常4）
}

func (d *StandardDeck) Generate() []Mahjong {
    return Generate(d.loop)  // 生成 27*4=108 张牌
}

// 3. 自定义牌堆
type CustomDeck struct {
    colors []Color
    loop   int
}

func (d *CustomDeck) Generate() []Mahjong {
    tiles := make([]Mahjong, 0)
    for _, color := range d.colors {
        tiles = append(tiles, generateColorTiles(color, d.loop)...)
    }
    return tiles
}

// 4. 牌堆管理器
type Deck struct {
    generator DeckGenerator
    tiles     []Mahjong
    position  int
}

func NewDeck(generator DeckGenerator) *Deck {
    return &Deck{
        generator: generator,
        tiles:     generator.Generate(),
        position:  0,
    }
}

func (d *Deck) Shuffle() {
    rand.Shuffle(len(d.tiles), func(i, j int) {
        d.tiles[i], d.tiles[j] = d.tiles[j], d.tiles[i]
    })
}

func (d *Deck) Draw() (Mahjong, error) {
    if d.position >= len(d.tiles) {
        return Mahjong{}, ErrDeckEmpty
    }
    tile := d.tiles[d.position]
    d.position++
    return tile, nil
}
```

---

### 4.4 支持不同的AI策略

**问题**：需要支持多种AI难度和策略

**方案**：

```go
// 1. AI 决策接口
type AIStrategy interface {
    Name() string
    Difficulty() int
    DecideDiscard(seat *Seat, table *Table) (Mahjong, error)
    DecideResponse(action Action, seat *Seat) ResponseDecision
}

// 2. 简单 AI
type SimpleAI struct{}

func (ai *SimpleAI) DecideDiscard(seat *Seat, table *Table) (Mahjong, error) {
    // 随机出牌
    hand := seat.GetHandView()
    return hand[rand.Intn(len(hand))], nil
}

// 3. 高级 AI
type AdvancedAI struct {
    evaluator *HandEvaluator
}

func (ai *AdvancedAI) DecideDiscard(seat *Seat, table *Table) (Mahjong, error) {
    // 分析手牌，选择最优出牌
    hand := seat.GetHandView()
    bestTile := ai.evaluator.FindBestDiscard(hand)
    return bestTile, nil
}

// 4. AI 工厂
type AIFactory struct{}

func (f *AIFactory) Create(difficulty string) AIStrategy {
    switch difficulty {
    case "easy":
        return &SimpleAI{}
    case "normal":
        return &NormalAI{}
    case "hard":
        return &AdvancedAI{evaluator: NewHandEvaluator()}
    default:
        return &SimpleAI{}
    }
}
```

---

## 5. 可维护性优化建议

### 5.1 分层架构

**当前问题**：所有逻辑混在一起

**改进方案**：

```
┌─────────────────────────────────────┐
│         API Layer (Handler)         │  HTTP/WebSocket 接口层
├─────────────────────────────────────┤
│       Service Layer (Logic)         │  业务逻辑层
│  - GameService                      │
│  - RoomService                      │
│  - PlayerService                    │
├─────────────────────────────────────┤
│      Domain Layer (Models)          │  领域模型层
│  - Table, Seat, Mahjong             │
│  - RuleEngine, Algorithm            │
├─────────────────────────────────────┤
│   Infrastructure Layer (Infra)      │  基础设施层
│  - Repository (数据持久化)          │
│  - MessageQueue (消息队列)          │
│  - Cache (缓存)                     │
└─────────────────────────────────────┘
```

**示例代码**：

```go
// 1. Service 层
type GameService struct {
    tableRepo TableRepository
    ruleEngine *RuleEngine
    eventBus  *EventBus
}

func (s *GameService) StartGame(roomID string) error {
    table, err := s.tableRepo.FindByRoomID(roomID)
    if err != nil {
        return err
    }

    // 业务逻辑
    if err := table.ValidateCanStart(); err != nil {
        return err
    }

    table.Start()

    // 发送事件
    s.eventBus.Publish(GameStartedEvent{TableID: table.ID()})

    // 持久化
    return s.tableRepo.Save(table)
}

// 2. Domain 层
type Table struct {
    // 领域模型，包含业务逻辑
}

func (t *Table) ValidateCanStart() error {
    if t.seatManager.GetSeatCount() < 4 {
        return ErrNotEnoughPlayers
    }
    if t.state.Phase != PhaseWaiting {
        return ErrGameAlreadyStarted
    }
    return nil
}

// 3. Infrastructure 层
type TableRepository interface {
    FindByRoomID(roomID string) (*Table, error)
    Save(table *Table) error
    Delete(id string) error
}
```

---

### 5.2 依赖注入

**问题**：对象创建硬编码，难以测试和替换

**方案**：

```go
// 1. 定义依赖接口
type Dependencies struct {
    TableRepo    TableRepository
    UserRepo     UserRepository
    EventBus     EventBus
    RuleEngine   *RuleEngine
    Logger       Logger
}

// 2. 使用依赖注入容器（如 wire）
func InitializeGameService(deps Dependencies) *GameService {
    return &GameService{
        tableRepo:  deps.TableRepo,
        ruleEngine: deps.RuleEngine,
        eventBus:   deps.EventBus,
        logger:     deps.Logger,
    }
}

// 3. 测试时注入 Mock
func TestGameService(t *testing.T) {
    mockRepo := &MockTableRepository{}
    mockEventBus := &MockEventBus{}

    service := InitializeGameService(Dependencies{
        TableRepo: mockRepo,
        EventBus:  mockEventBus,
        Logger:    NewTestLogger(),
    })

    // 测试...
}
```

---

### 5.3 错误处理

**问题**：错误处理不统一，缺少上下文

**方案**：

```go
// 1. 定义错误类型
type GameError struct {
    Code    string
    Message string
    Cause   error
    Context map[string]interface{}
}

func (e *GameError) Error() string {
    return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
}

// 2. 预定义错误
var (
    ErrInvalidMove     = &GameError{Code: "INVALID_MOVE", Message: "Invalid move"}
    ErrGameNotFound    = &GameError{Code: "GAME_NOT_FOUND", Message: "Game not found"}
    ErrPlayerNotInGame = &GameError{Code: "PLAYER_NOT_IN_GAME", Message: "Player not in game"}
)

// 3. 包装错误，添加上下文
func (s *GameService) DiscardTile(tableID string, userID int64, tile Mahjong) error {
    table, err := s.tableRepo.FindByID(tableID)
    if err != nil {
        return &GameError{
            Code:    "TABLE_LOAD_FAILED",
            Message: "Failed to load table",
            Cause:   err,
            Context: map[string]interface{}{
                "tableID": tableID,
                "userID":  userID,
            },
        }
    }

    // ...
}

// 4. 错误恢复
func (s *GameService) handleError(err error) {
    var gameErr *GameError
    if errors.As(err, &gameErr) {
        s.logger.Error("Game error",
            "code", gameErr.Code,
            "context", gameErr.Context,
            "cause", gameErr.Cause,
        )
    }
}
```

---

### 5.4 日志和监控

```go
// 1. 结构化日志
type GameLogger struct {
    logger *slog.Logger
}

func (l *GameLogger) LogGameAction(action string, context map[string]interface{}) {
    l.logger.Info("game_action",
        "action", action,
        "timestamp", time.Now(),
        "context", context,
    )
}

// 2. 指标收集
type GameMetrics struct {
    gamesStarted    prometheus.Counter
    gamesFinished   prometheus.Counter
    playerActions   *prometheus.CounterVec
    gameDuration    prometheus.Histogram
}

func (m *GameMetrics) RecordGameStarted() {
    m.gamesStarted.Inc()
}

func (m *GameMetrics) RecordPlayerAction(action string) {
    m.playerActions.WithLabelValues(action).Inc()
}

// 3. 追踪
func (s *GameService) DiscardTile(ctx context.Context, req *DiscardRequest) error {
    span := trace.SpanFromContext(ctx)
    span.SetAttributes(
        attribute.String("table_id", req.TableID),
        attribute.Int64("user_id", req.UserID),
    )

    // 业务逻辑...
}
```

---

### 5.5 测试策略

```go
// 1. 单元测试 - 测试单个对象
func TestMahjong_Generate(t *testing.T) {
    tiles := Generate(4)
    assert.Equal(t, 108, len(tiles))

    // 验证每种牌有4张
    counts := make(map[int]int)
    for _, tile := range tiles {
        counts[tile.Number]++
    }
    for _, count := range counts {
        assert.Equal(t, 4, count)
    }
}

// 2. 集成测试 - 测试对象协作
func TestGameService_StartGame(t *testing.T) {
    // 准备
    deps := setupTestDependencies(t)
    service := InitializeGameService(deps)

    // 创建房间和玩家
    tableID := createTestTable(t, deps.TableRepo)
    addTestPlayers(t, tableID, 4)

    // 执行
    err := service.StartGame(tableID)

    // 验证
    assert.NoError(t, err)
    table, _ := deps.TableRepo.FindByID(tableID)
    assert.Equal(t, PhaseDrawTile, table.State().Phase)
}

// 3. 表格驱动测试 - 测试算法
func TestCheckWin(t *testing.T) {
    tests := []struct {
        name     string
        hand     []Mahjong
        tile     Mahjong
        expected bool
    }{
        {
            name: "clear hand win",
            hand: createHand([]int{1, 1, 1, 2, 3, 4, 5, 6, 7, 8, 8, 8, 9}),
            tile: createTile(9),
            expected: true,
        },
        // 更多测试用例...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := CheckHu(CATCH, nil, createSeat(tt.hand), tt.tile)
            assert.Equal(t, tt.expected, result)
        })
    }
}

// 4. 基准测试 - 性能测试
func BenchmarkCanFormWinningHand(b *testing.B) {
    hand := map[int]int{1: 3, 2: 3, 3: 3, 4: 3, 5: 2}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        CanFormWinningHand(hand, false)
    }
}
```

---

## 6. 设计模式应用

### 6.1 策略模式（Strategy Pattern）

**应用场景**：不同的胡牌规则、计分规则

```go
// 策略接口
type ScoringStrategy interface {
    CalculateScore(winType []HuType, context *GameContext) int
}

// 具体策略
type BasicScoring struct{}
func (s *BasicScoring) CalculateScore(winType []HuType, ctx *GameContext) int {
    score := 0
    for _, ht := range winType {
        score += getBaseScore(ht)
    }
    return score
}

type MultiplierScoring struct {
    multiplier int
}
func (s *MultiplierScoring) CalculateScore(winType []HuType, ctx *GameContext) int {
    baseScore := new(BasicScoring).CalculateScore(winType, ctx)
    return baseScore * s.multiplier
}
```

### 6.2 建造者模式（Builder Pattern）

**应用场景**：复杂的 Table 对象创建

```go
type TableBuilder struct {
    table *Table
}

func NewTableBuilder() *TableBuilder {
    return &TableBuilder{
        table: &Table{},
    }
}

func (b *TableBuilder) WithRoomID(id string) *TableBuilder {
    b.table.RoomID = id
    return b
}

func (b *TableBuilder) WithPlayers(users []*model.User) *TableBuilder {
    // 初始化座位...
    return b
}

func (b *TableBuilder) WithRuleSet(ruleSet string) *TableBuilder {
    // 设置规则...
    return b
}

func (b *TableBuilder) Build() (*Table, error) {
    if err := b.validate(); err != nil {
        return nil, err
    }
    return b.table, nil
}

// 使用
table, err := NewTableBuilder().
    WithRoomID("room123").
    WithPlayers(players).
    WithRuleSet("sichuan").
    Build()
```

### 6.3 观察者模式（Observer Pattern）

**应用场景**：游戏事件通知

```go
// 事件接口
type GameEvent interface {
    Type() string
    Timestamp() time.Time
}

// 具体事件
type TileDiscardedEvent struct {
    timestamp time.Time
    seat      *Seat
    tile      Mahjong
}

func (e *TileDiscardedEvent) Type() string { return "tile_discarded" }
func (e *TileDiscardedEvent) Timestamp() time.Time { return e.timestamp }

// 观察者接口
type GameObserver interface {
    OnEvent(event GameEvent)
}

// 事件总线
type EventBus struct {
    observers map[string][]GameObserver
}

func (b *EventBus) Subscribe(eventType string, observer GameObserver) {
    b.observers[eventType] = append(b.observers[eventType], observer)
}

func (b *EventBus) Publish(event GameEvent) {
    for _, observer := range b.observers[event.Type()] {
        go observer.OnEvent(event)  // 异步通知
    }
}
```

### 6.4 状态模式（State Pattern）

**应用场景**：游戏阶段管理

```go
// 状态接口
type GameState interface {
    Name() string
    OnEnter(table *Table) error
    OnExit(table *Table) error
    CanTransitionTo(next GameState) bool
}

// 具体状态
type WaitingState struct{}
func (s *WaitingState) Name() string { return "waiting" }
func (s *WaitingState) OnEnter(table *Table) error {
    // 等待玩家准备...
    return nil
}
func (s *WaitingState) CanTransitionTo(next GameState) bool {
    _, ok := next.(*PlayingState)
    return ok  // 只能转到 Playing
}

type PlayingState struct{}
func (s *PlayingState) Name() string { return "playing" }
func (s *PlayingState) OnEnter(table *Table) error {
    // 发牌，开始游戏...
    return nil
}

// 状态机
type GameStateMachine struct {
    current GameState
    table   *Table
}

func (sm *GameStateMachine) TransitionTo(next GameState) error {
    if !sm.current.CanTransitionTo(next) {
        return ErrInvalidStateTransition
    }

    if err := sm.current.OnExit(sm.table); err != nil {
        return err
    }

    sm.current = next

    return sm.current.OnEnter(sm.table)
}
```

### 6.5 工厂模式（Factory Pattern）

**应用场景**：创建不同类型的游戏对象

```go
// 抽象工厂
type GameFactory interface {
    CreateTable(config TableConfig) (*Table, error)
    CreateDeck() *Deck
    CreateRuleEngine() *RuleEngine
}

// 具体工厂
type SichuanMahjongFactory struct{}

func (f *SichuanMahjongFactory) CreateTable(config TableConfig) (*Table, error) {
    return &Table{
        config:     config,
        deck:       f.CreateDeck(),
        ruleEngine: f.CreateRuleEngine(),
    }, nil
}

func (f *SichuanMahjongFactory) CreateDeck() *Deck {
    return NewDeck(&StandardDeck{loop: 4})
}

func (f *SichuanMahjongFactory) CreateRuleEngine() *RuleEngine {
    return NewRuleEngine(RuleConfig{
        EnabledRules: []string{"clear_hand", "seven_pairs", "pong_pong"},
    })
}
```

---

## 7. 重构路线图

### 阶段一：基础重构（1-2周）

**目标**：提高代码质量，不改变功能

```
✅ 优先级：高
├─ 1. 添加单元测试（覆盖率 > 70%）
├─ 2. 提取常量和枚举
├─ 3. 添加注释和文档
├─ 4. 重命名不清晰的变量和方法
└─ 5. 提取重复代码为辅助函数
```

**检查清单**：
- [ ] 所有公开方法都有注释
- [ ] 核心算法有单元测试
- [ ] 消除所有魔法数字
- [ ] 代码通过 golangci-lint 检查

---

### 阶段二：结构优化（2-3周）

**目标**：改善对象设计，提高封装性

```
✅ 优先级：高
├─ 1. Seat 对象封装优化
│   ├─ 将公开字段改为私有
│   ├─ 添加 Getter/Setter 方法
│   └─ 添加状态验证逻辑
│
├─ 2. Table 对象拆分
│   ├─ 提取 TableConfig
│   ├─ 提取 GameState
│   └─ 提取 SeatManager
│
├─ 3. Algorithm 模块化
│   ├─ 按功能拆分文件
│   ├─ 提取公共算法
│   └─ 添加算法测试
│
└─ 4. 错误处理规范化
    ├─ 定义错误类型
    ├─ 添加错误上下文
    └─ 统一错误返回
```

---

### 阶段三：架构升级（3-4周）

**目标**：引入设计模式，提高可拓展性

```
✅ 优先级：中
├─ 1. 实现策略模式
│   ├─ 抽象 WinRule 接口
│   ├─ 实现具体规则
│   └─ 创建 RuleEngine
│
├─ 2. 实现观察者模式
│   ├─ 设计事件系统
│   ├─ 实现 EventBus
│   └─ 添加事件监听器
│
├─ 3. 实现状态模式
│   ├─ 定义游戏状态
│   ├─ 实现状态机
│   └─ 添加状态转换逻辑
│
└─ 4. 引入依赖注入
    ├─ 定义接口
    ├─ 使用 wire 或类似工具
    └─ 重构对象创建
```

---

### 阶段四：功能拓展（持续）

**目标**：支持新功能和新规则

```
✅ 优先级：中-低
├─ 1. 支持多种麻将规则
│   ├─ 四川麻将
│   ├─ 广东麻将
│   └─ 日本麻将
│
├─ 2. 支持不同玩家数量
│   ├─ 2人麻将
│   ├─ 3人麻将
│   └─ 动态座位管理
│
├─ 3. AI系统
│   ├─ 简单AI
│   ├─ 中级AI
│   └─ 高级AI（机器学习）
│
└─ 4. 性能优化
    ├─ 算法优化
    ├─ 内存优化
    └─ 并发优化
```

---

## 8. 总结与建议

### 8.1 立即行动项（本周）

1. **添加测试**：为核心算法添加单元测试
2. **提取常量**：消除魔法数字，定义常量
3. **改进注释**：为公开方法添加清晰的注释
4. **代码审查**：运行 golangci-lint，修复警告

### 8.2 短期目标（1个月）

1. **封装优化**：改善 Seat 和 Table 的封装性
2. **模块拆分**：将 Algorithm 拆分为多个文件
3. **错误处理**：统一错误处理方式
4. **文档完善**：编写使用文档和示例

### 8.3 长期目标（3-6个月）

1. **架构重构**：引入设计模式，提高可拓展性
2. **规则引擎**：实现灵活的规则配置系统
3. **插件系统**：支持自定义规则和功能
4. **性能优化**：优化算法和数据结构

---

## 附录

### A. 推荐资源

**书籍**：
- 《Clean Code》- Robert C. Martin
- 《Design Patterns》- Gang of Four
- 《Domain-Driven Design》- Eric Evans

**工具**：
- golangci-lint: 代码质量检查
- wire: 依赖注入工具
- testify: 测试框架
- pprof: 性能分析

**参考项目**：
- [gomoku](https://github.com/gomoku/gomoku) - 棋类游戏参考
- [mahjong](https://github.com/mahjong/mahjong) - 麻将实现参考

---

### B. 代码规范

**命名约定**：
- 接口：以 -er 结尾（如 `WinChecker`）
- 私有字段：小写开头
- 常量：大写字母和下划线
- 错误变量：Err 前缀

**注释规范**：
```go
// CheckWin 检查是否可以胡牌
//
// 参数：
//   - supplierType: 供牌方式（CATCH/OUT/GANG）
//   - seat: 当前座位
//   - tile: 要检查的牌
//
// 返回：
//   - bool: 是否可以胡牌
//
// 注意：
//   - 不能抢杠胡自己
//   - 需要考虑报听状态
func CheckWin(supplierType SupplierType, seat *Seat, tile Mahjong) bool {
    // ...
}
```

---

**文档版本**：v1.0
**最后更新**：2026-01-12
**维护者**：开发团队
