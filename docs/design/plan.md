# IM-GO 低时延优先优化计划

## 最高准则

IM-GO 的优化必须以性能和低时延为最高准则。所有可靠性、扩展性、持久化、安全和观测改造，都不能让 Access/Logic/Web 的消息主链路等待慢速数据库、外部服务、无界队列或高延迟同步调用。

核心目标：
- 消息发送热路径保持短链路：认证、编解码、入队、路由、快速 ACK。
- 慢操作全部异步化：持久化、重试、离线、审计、指标上报、历史查询。
- 队列全部有界化：满队列快速失败或降级，不能无限等待。
- 优化必须能用 P50/P95/P99 时延、吞吐、队列积压、丢弃数、资源占用验证。

## 当前关键问题

1. Access 下行推送走连接写队列，但部分请求响应直接写 WebTransport stream，写路径不统一，可能影响 P99 稳定性。
2. Access 入站帧按客户端声明长度直接分配 body，缺少最大帧长、读超时和认证前限流。
3. Logic 消费 NATS 时缓冲区满会丢消息，当前保护了服务低时延，但缺少可观测、可补偿链路。
4. 消息批量写入采用先分配 serverMsgId 并快速 ACK 的低时延方案，但 ACK 语义未区分 accepted 和 persisted。
5. 房间和游戏状态在 Logic 单进程内存中，Logic 横向扩展后同一房间请求可能落到不同节点。
6. 群消息按成员 fan-out 并发查询 Redis，缺少并发上限和批量优化，大群会拖慢全局消息处理。
7. 前端 pending 消息使用本地 UUID，ACK 使用 reqId/serverMsgId，发送状态没有完整闭环。
8. 观测指标不足，无法量化 P99、队列积压、丢弃和慢依赖对主链路的影响。

## 阶段一：保护 Access 热路径

目标：降低连接层 P99 抖动，避免异常客户端拖慢主链路。

涉及改动：
- `project/access-go/internal/connection/connection.go`
  - 保持 `Send` 非阻塞语义。
  - 所有服务端写客户端的响应和推送统一进入单 writer 队列。
  - 写队列满时继续快速返回背压错误，并记录可观测指标。
- `project/access-go/internal/handler/handler.go`
  - 增加最大帧长校验，超过限制直接关闭连接或返回错误。
  - 增加认证后普通请求帧的基础校验，非法 payload 快速失败。
  - 避免请求响应直接 `stream.Write`，改为经 `conn.Send` 发送。
- `project/access-go/internal/handler/auth_handler.go`
  - 认证首包增加最大长度校验。
  - 认证失败仍快速返回，不做重型逻辑。
- `project/access-go/internal/server/server.go`
  - 接入 `max_connections` 约束。
  - 增加 Origin 白名单配置。
  - 对 0-RTT 保持谨慎，只允许幂等或可去重请求进入业务处理。
- `project/access-go/internal/config/config.go`
  - 增加 `max_frame_size`、`allowed_origins`、认证前限流相关配置。

验证指标：
- Access 入站请求 P99。
- 下行写队列长度和满队列次数。
- 单连接内存峰值。
- 非法帧拒绝数。

## 阶段二：低时延消息可靠性

目标：不牺牲发送时延的前提下补齐可恢复能力。

设计原则：
- 发送路径不等待 PostgreSQL 持久化。
- ACK 分层：`accepted` 表示 Logic 已接收并分配 serverMsgId；`persisted` 表示后台落库成功；`delivered/read` 作为后续投递状态。
- 幂等去重优先使用 `fromUserId + clientMsgId`，避免 0-RTT 或重连重发导致重复消息。

涉及改动：
- `env/schema.sql`
  - 给 `messages` 增加适合幂等的唯一约束或唯一索引，例如 `(from_user_id, client_msg_id)`。
  - 按查询模式补充会话维度索引，为后续历史消息分页做准备。
- `project/logic-go/internal/service/message_batcher.go`
  - 保持 `SaveMessage` 快速入队返回。
  - 增加批写失败的异步补偿记录，不阻塞发送路径。
  - 增加批写耗时、队列长度、队列满次数指标。
- `project/logic-go/internal/handler/chat_handler.go`
  - ACK 语义调整为 accepted。
  - 持久化成功后可选异步发送 persisted 事件。
- `project/shared/proto/messages.go`
  - 扩展 ACK 类型字段，例如 `Status: accepted/persisted`。
  - 增加 `TraceId` 或 `EventId`，用于链路追踪和补偿。
- `project/access-go/internal/handler/downstream_handler.go`
  - 解析并下发新的 ACK 状态。

验证指标：
- 客户端发送到 accepted ACK 的 P50/P95/P99。
- 批量落库 P95/P99。
- 批写失败补偿数量。
- 重复 clientMsgId 命中数。

## 阶段三：Logic 分片与房间/游戏低时延扩展

目标：保持房间/游戏内存态低延迟，同时支持横向扩展。

设计原则：
- 不把每次房间操作改成数据库事务或分布式锁同步等待。
- 用 roomId 粘性路由保证同一房间请求固定落到同一 Logic 分片。
- 内存态为主，Redis/PostgreSQL 异步快照用于恢复和补偿。

涉及改动：
- `project/shared/nats/subjects.go`
  - 增加房间/游戏分片 subject，例如 `im.logic.room.{shard}`。
  - 保留普通 IM 消息 subject，避免房间热点影响单聊。
- `project/access-go/internal/handler/room_handler.go`
  - 房间请求按 `roomId` 或创建请求分配 shard 后发布到对应 subject。
- `project/access-go/internal/handler/game_handler.go`
  - 游戏请求按 `roomId` 发布到同一 shard。
- `project/logic-go/internal/nats/subscriber.go`
  - 订阅当前节点负责的 room/game shard。
  - 普通消息和房间/游戏消息使用独立 worker 池，避免互相拖慢。
- `project/logic-go/internal/room/manager.go`
  - 保持内存态和 LRU。
  - 增加异步快照触发点。
- `project/logic-go/internal/game/manager.go`
  - 补充游戏状态异步快照。
  - 游戏结束事件异步持久化，不阻塞玩家操作。

验证指标：
- 房间操作 P99。
- 单 shard 房间数、游戏数、热点房间 QPS。
- room/game worker 队列积压。
- 快照耗时和恢复耗时。

## 阶段四：群消息高扇出优化

目标：大群消息不拖慢单聊和小群。

涉及改动：
- `project/logic-go/internal/service/router.go`
  - `fetchMultipleUserLocations` 增加并发上限。
  - 支持批量 MGET 用户位置，减少 Redis 往返。
  - 大群消息进入 fan-out worker，不占用普通消息 worker。
- `project/logic-go/internal/service/location.go`
  - 增加批量位置查询接口。
  - 保持短 TTL 缓存，但避免缓存长期污染路由。
- `project/logic-go/internal/service/group.go`
  - 群成员列表增加 Redis 缓存和失效机制。

验证指标：
- 大群 fan-out P95/P99。
- Redis MGET 耗时。
- fan-out worker 队列长度。
- 单聊消息在大群压力下的 P99。

## 阶段五：前端发送状态与离线队列

目标：客户端感知更快、更准，重连不丢用户操作。

涉及改动：
- `project/desktop-web/src/stores/messageStore.ts`
  - 维护 `reqId -> localMsgId -> serverMsgId` 映射。
  - accepted ACK 到达后立刻把 pending 改为 sent 或 accepted。
  - persisted ACK 到达后更新持久化状态。
- `project/desktop-web/src/services/transport/WebTransportManager.ts`
  - 重连后触发未确认消息恢复流程。
  - 保持心跳轻量，不阻塞 UI。
- `project/desktop-web/src/services/protocol/IMProtocol.ts`
  - 支持新的 ACK 状态字段。
- `project/desktop-web/src/services/mahjongRoomService.ts`
  - 补齐 `targetSeatIndex` 写入，避免换座位请求协议字段缺失。

验证指标：
- 用户点击发送到本地状态更新耗时。
- 发送到 accepted ACK 的端到端 P99。
- 重连后未确认消息恢复成功率。

## 阶段六：观测与性能门禁

目标：所有优化可度量，避免无感知退化。

涉及改动：
- `scripts/check-go-quality.sh`
  - 后续可加入轻量性能基准入口，但默认不生成编译产物。
- Access 指标：
  - 活跃连接数、认证耗时、帧解析耗时、写队列长度、背压次数、连接关闭原因。
- Logic 指标：
  - NATS 消费延迟、worker 队列长度、消息 accepted ACK 耗时、批写耗时、fan-out 耗时、Redis 查询耗时。
- Web 指标：
  - 登录耗时、Token Redis 查询耗时、API P95/P99。
- 前端指标：
  - WebTransport 连接耗时、认证耗时、消息 ACK 延迟、重连次数。

注意事项：
- 高 QPS 路径日志必须采样，不能每条消息都打 info 日志。
- 指标上报异步化，不能阻塞消息处理。
- 性能回归以 P99 为主要判断依据。

## 建议落地顺序

1. 先改 Access 写路径统一、最大帧长和连接限流。
2. 再改消息 ACK 语义、幂等索引和批写补偿。
3. 然后做 roomId 分片路由，隔离房间/游戏 worker。
4. 接着优化群消息 fan-out 和批量位置查询。
5. 最后补前端状态闭环和全链路指标。

这个顺序能优先保护主链路低时延，同时逐步补齐可靠性和横向扩展能力。
