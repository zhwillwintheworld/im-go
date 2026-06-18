# 房间与游戏领域

## 领域职责

房间与游戏领域负责房间创建、加入、离开、准备、换座位、开始游戏、多轮游戏、单局游戏、麻将动作和游戏事件推送。

## 房间流程

1. 用户创建房间。
2. 房主自动加入房间。
3. 其他用户加入并分配座位。
4. 用户准备或取消准备。
5. 房主发起开始游戏。
6. Logic 校验玩家人数、房主权限和准备状态。
7. GameManager 创建游戏并启动第一局。
8. 房间内用户收到房间或游戏事件推送。

## 房间状态

- `waiting`：等待玩家和准备。
- `playing`：游戏中。
- 后续可扩展 `settling`、`closed`。

## 游戏状态

游戏采用分层模型：
- GameManager：管理房间到游戏实例的映射。
- MultiRoundGame：管理多局、总分和结束条件。
- Round：管理单局状态、操作串行化和任务超时。
- RoundEngine：具体玩法引擎，例如麻将。

## 低时延要求

- 房间和游戏互动优先使用内存态，避免每次操作同步写数据库。
- Logic 横向扩展时必须使用 roomId 粘性路由。
- 房间和游戏状态异步快照到 Redis/PostgreSQL，用于恢复和补偿。
- 房间/游戏 worker 应与普通聊天 worker 隔离，避免热点房间拖慢单聊。

## 粘性路由

- Access 将房间请求发布到 `im.logic.room.{shard}`，将游戏请求发布到 `im.logic.game.{shard}`。
- `shard` 由 `roomId` 稳定哈希得到；创建房间等尚无 `roomId` 的请求使用 `userId` 作为 fallback key。
- `room_shard_count <= 1` 时统一落到 shard `0`，兼容单 Logic 节点部署。
- Logic 节点使用 `room_shard_index` 订阅自身负责的 room/game shard，不使用普通队列组抢占这些有状态请求。
- 普通聊天上行仍走 `im.logic.upstream`，room/game worker 使用独立队列和 worker 数，避免热点房间拖慢单聊 P99。

相关配置：
- Access：`server.room_shard_count`，环境变量 `ROOM_SHARD_COUNT`。
- Logic：`nats.room_shard_count`、`nats.room_shard_index`、`nats.room_worker_count`、`nats.room_buffer_size`。

## 异步快照

- RoomManager 和 GameManager 都提供非阻塞 `TriggerSnapshot(roomId)`。
- 房间创建、加入、离开、准备、换座、开始游戏成功后触发房间快照。
- 游戏启动首局、玩家动作处理、单局结算成功后触发游戏快照。
- 快照队列满时快速丢弃并记录日志；不得让房间/游戏请求等待 Redis、PostgreSQL 或无界重试。
- 当前阶段快照 worker 只记录触发点，后续接入持久化时必须保持异步、有界、可降级。

## 当前边界

- 房间和游戏状态当前仍以内存态为主，已具备 roomId 粘性路由和异步快照触发入口。
- 完整游戏持久化、恢复和补偿逻辑仍需后续补齐。
- 麻将 GamePayload 到具体动作处理仍有 TODO。
- 业务扩展时必须先明确 roomId 路由归属。

## 关键代码

- `project/logic-go/internal/room/manager.go`
- `project/logic-go/internal/room/service.go`
- `project/logic-go/internal/room/operations.go`
- `project/logic-go/internal/room/room.go`
- `project/logic-go/internal/game/manager.go`
- `project/logic-go/internal/game/multi_round_game.go`
- `project/logic-go/internal/game/round.go`
- `project/logic-go/internal/handler/room_handler.go`
- `project/access-go/internal/handler/room_handler.go`
- `project/access-go/internal/handler/game_handler.go`
- `project/desktop-web/src/services/mahjongRoomService.ts`

## 相关指标

- 房间操作 P99
- 每个 Logic shard 的房间数
- 游戏操作 P99
- room/game worker 队列长度
- 房间淘汰数量
- 游戏快照耗时
