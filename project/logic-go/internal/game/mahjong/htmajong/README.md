# HTMajong - 海淘麻将游戏引擎

> Go语言实现的海淘麻将游戏核心逻辑

---

## 📋 目录结构

```
htmajong/
├── DESIGN.md                 # 设计文档（架构设计和优化方案）
├── OPTIMIZATION_SUMMARY.md   # 优化总结（已完成的改进）
├── README.md                 # 本文件
│
├── errors.go                 # 错误定义
├── mahjong.go                # 麻将牌对象
├── hand.go                   # 手牌管理器
├── tiles.go                  # 公开牌和出牌堆管理器
├── seat.go                   # 座位对象
├── table.go                  # 牌桌对象
├── lease.go                  # 租约系统（多人响应）
├── algorithm.go              # 胡牌算法
│
├── color.go                  # 颜色枚举
├── position.go               # 位置枚举
├── hu.go                     # 胡牌类型
├── supplier_type.go          # 供牌方式枚举
└── task.go                   # 任务类型枚举
```

---

## 🎯 核心概念

### 对象层次

```
Table (牌桌)
  ├── Seat (座位) x 4
  │     ├── Hand (手牌管理器)
  │     ├── PublicTiles (公开牌管理器)
  │     ├── DiscardPile (出牌堆)
  │     └── ListeningState (报听状态)
  │
  ├── Deck (牌堆)
  └── Lease (租约系统)
```

### 核心对象

| 对象 | 说明 | 文件 |
|------|------|------|
| **Mahjong** | 麻将牌值对象 | `mahjong.go` |
| **Hand** | 手牌管理器 | `hand.go` |
| **PublicTiles** | 公开牌管理器（碰、杠） | `tiles.go` |
| **DiscardPile** | 出牌堆管理器 | `tiles.go` |
| **Seat** | 座位（玩家状态） | `seat.go` |
| **Table** | 牌桌（游戏容器） | `table.go` |
| **LeaseInfo** | 租约（多人响应机制） | `lease.go` |

---

## 🚀 快速开始

### 1. 创建麻将牌

```go
// 生成一副麻将（108张牌，每种4张）
tiles := htmajong.Generate(4)

// 根据数字生成单张牌
tile, err := htmajong.GenerateByNumber(5)  // 5万
```

### 2. 创建座位

```go
user := &model.User{
    UserID:   12345,
    Username: "player1",
    Nickname: "玩家1",
}

seat := htmajong.NewSeat(user, htmajong.EAST)
```

### 3. 手牌操作

```go
// 摸牌
err := seat.DrawTile(tile)

// 出牌
err := seat.DiscardTile(tile)

// 获取手牌
handTiles := seat.GetHandTiles()

// 判断是否可以报听
canListen := seat.CanDeclareListening()
```

### 4. 检查胡牌

```go
// 检查是否可以胡牌
canWin := htmajong.CheckHu(
    htmajong.CATCH,      // 供牌方式
    supplierSeat,        // 供牌座位
    seat,                // 当前座位
    tile,                // 牌
)

// 获取胡牌类型
winTypes := htmajong.CheckHUType(table, htmajong.CATCH, seat, tile)
// 返回：[CLEAR, PENG_PENG_HU] 等
```

### 5. 检查碰、杠

```go
// 检查是否可以碰
canPong := htmajong.CheckPeng(htmajong.OUT, supplierSeat, seat, tile)

// 检查是否可以杠
// 返回：0-不能杠，1-杠，2-公杠，3-暗杠
gangType := htmajong.CheckGang(table, htmajong.OUT, supplierSeat, seat, tile)
```

---

## 📖 详细文档

### 设计文档
查看 [DESIGN.md](./DESIGN.md) 了解：
- 完整的架构设计
- 设计模式应用
- 可拓展性方案
- 重构路线图

### 优化总结
查看 [OPTIMIZATION_SUMMARY.md](./OPTIMIZATION_SUMMARY.md) 了解：
- 已完成的优化
- 代码对比
- 改进效果
- 下一步计划

---

## 🎮 游戏规则

### 支持的胡牌类型

| 类型 | 说明 | 代码 |
|------|------|------|
| 普通胡 | 标准牌型 | `GENERAL` |
| 清一色 | 全是同一花色 | `CLEAR` |
| 碰碰胡 | 全是刻子+一对将 | `PENG_PENG_HU` |
| 七小对 | 七个对子 | `SEVEN_PAIR` |
| 龙七对 | 七小对中有四张 | `LOONG_SEVEN_PAIR` |
| 258 | 全是2、5、8 | `TWO_FIVE_EIGHT` |
| 缺一门 | 只有两种花色 | `TWO_COLOR` |
| 无将糊 | 没有2、5、8 | `NO_JIANG` |
| 报听 | 报听后胡牌 | `BAO_TING` |

### 牌型编码

```go
万子: 1-9
条子: 11-19
饼子: 21-29
```

### 游戏流程

1. **发牌**: 每人13张牌
2. **摸牌**: 从牌堆摸一张
3. **出牌**: 打出一张牌
4. **响应**: 其他玩家可以碰/杠/胡
5. **报听**: 第一手可以报听（可选规则）
6. **胡牌**: 符合胡牌条件即可胡牌

---

## 🔧 API 参考

### Seat (座位)

```go
// 创建座位
func NewSeat(user *model.User, position Position) *Seat

// 手牌操作
func (s *Seat) DrawTile(tile Mahjong) error
func (s *Seat) DiscardTile(tile Mahjong) error
func (s *Seat) GetHandTiles() []Mahjong
func (s *Seat) GetHandSize() int

// 碰杠操作
func (s *Seat) Pong(tiles []Mahjong) error
func (s *Seat) Kong(tiles []Mahjong, kongType TileGroupType) error

// 报听操作
func (s *Seat) DeclareListening(tiles []Mahjong) error
func (s *Seat) CanDeclareListening() bool
func (s *Seat) IsListening() bool

// 状态查询
func (s *Seat) GetPoints() int
func (s *Seat) IsFirstRound() bool
```

### Algorithm (算法)

```go
// 检查胡牌
func CheckHu(supplierType SupplierType, supplierUser *Seat, seat *Seat, mahjong Mahjong) bool

// 获取胡牌类型
func CheckHUType(table *Table, supplierType SupplierType, seat *Seat, mahjong Mahjong) []HuType

// 检查碰
func CheckPeng(supplierType SupplierType, supplierUser *Seat, seat *Seat, mahjong Mahjong) bool

// 检查杠
func CheckGang(table *Table, supplierType SupplierType, supplierUser *Seat, seat *Seat, mahjong Mahjong) int

// 检查报听
func CheckPublic(table *Table, seat *Seat) bool
```

### Hand (手牌管理器)

```go
// 创建手牌管理器
func NewHand() *Hand

// 基本操作
func (h *Hand) Add(tile Mahjong) error
func (h *Hand) Remove(tile Mahjong) error
func (h *Hand) Contains(tile Mahjong) bool
func (h *Hand) Count(number int) int

// 状态查询
func (h *Hand) Size() int
func (h *Hand) IsEmpty() bool
func (h *Hand) IsFull() bool

// 高级操作
func (h *Hand) Sort()
func (h *Hand) Clone() *Hand
func (h *Hand) ToCountMap() map[int]int
func (h *Hand) IsAllSameColor() bool
```

---

## ⚠️ 注意事项

### 并发安全

- `Seat` 对象使用 `sync.RWMutex` 保证并发安全
- 所有公开方法都已加锁
- 不要直接访问私有字段

### 向后兼容

当前保留了一些向后兼容的方法（标记为 TODO）：
```go
// 向后兼容方法（待移除）
func (s *Seat) ExtraList() []Mahjong
func (s *Seat) PublicList() []Mahjong
func (s *Seat) IsPublic() bool
```

新代码应使用新的方法：
```go
// 推荐使用
seat.GetHandTiles()
seat.GetPublicTiles()
seat.IsListening()
```

---

## 🧪 测试

```bash
# 运行测试
go test ./...

# 运行基准测试
go test -bench=. ./...

# 查看覆盖率
go test -cover ./...
```

---

## 📊 性能特点

- ✅ 零内存分配的牌型判断算法
- ✅ 使用对象池减少GC压力
- ✅ 优化的递归算法（`CanFormWinningHand`）
- ✅ 预分配切片容量

---

## 🛠️ 开发指南

### 添加新的胡牌类型

1. 在 `hu.go` 中添加新的 `HuType` 常量
2. 在 `algorithm.go` 中实现检查函数
3. 在 `commonCheckHu` 中调用检查函数
4. 编写单元测试

### 拓展座位功能

1. 在 `Seat` 中添加新的状态字段
2. 实现相关的方法
3. 添加并发保护（mutex）
4. 更新文档

---

## 📝 代码规范

### 命名约定

- 接口：以 `-er` 结尾
- 私有字段：小写开头
- 常量：大写字母和下划线
- 错误变量：`Err` 前缀

### 注释规范

```go
// CheckHu 检查是否可以胡牌
//
// 参数：
//   - supplierType: 供牌方式（CATCH/OUT/GANG）
//   - seat: 当前座位
//   - tile: 要检查的牌
//
// 返回：
//   - bool: 是否可以胡牌
func CheckHu(supplierType SupplierType, seat *Seat, tile Mahjong) bool
```

---

## 🔗 相关链接

- [设计文档](./DESIGN.md)
- [优化总结](./OPTIMIZATION_SUMMARY.md)
- [项目主页](../../)

---

## 📄 许可证

Copyright © 2026 Sudooom Team

---

**维护者**: 开发团队
**版本**: v1.0
**最后更新**: 2026-01-12
