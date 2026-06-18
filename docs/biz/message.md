# 消息领域

## 领域职责

消息领域负责聊天消息的上行接收、serverMsgId 分配、ACK、持久化、路由、会话更新、未读数和后续离线补偿。

## 主链路

1. 客户端发送 ChatSendReq。
2. Access 转换为 UpstreamMessage 并发布 NATS。
3. Logic 消费 UserMessage。
4. Logic 快速分配 serverMsgId。
5. 消息进入批量写队列。
6. Logic 立即向发送连接返回 accepted ACK。
7. Logic 查询接收方在线位置并下发 PushMessage。
8. 后台批量写 PostgreSQL。
9. 批写成功后发送 persisted ACK；批写失败记录失败计数并释放幂等占位，允许客户端重试。

## ACK 语义

当前已落地以下状态：

- `accepted`：Logic 已接收并分配 serverMsgId，发送路径成功。
- `persisted`：后台持久化成功。

以下状态是后续扩展方向：

- `delivered`：已推送到接收方在线连接。
- `read`：接收方已读到指定消息。

发送路径不得等待 persisted。

## 幂等与去重

- 客户端必须生成稳定 `clientMsgId` 或 reqId。
- 服务端使用 `fromUserId + clientMsgId` 去重。
- PostgreSQL `messages` 表使用 `from_user_id + client_msg_id` 的部分唯一索引防止重复落库。
- Logic 批写队列前有进程内轻量幂等表：重复 `clientMsgId` 复用同一个 `serverMsgId`，只重复 ACK，不重复路由。
- 重连重发、0-RTT 重放、客户端重试都必须命中幂等逻辑。

## 前端发送状态

- 前端发送消息时维护 `reqId -> localMsgId` 映射。
- 收到 `accepted` ACK 后，本地消息由 `pending` 更新为 `accepted`，并记录 `serverMsgId`。
- 收到 `persisted` ACK 后，本地消息更新为 `persisted`，并清理未确认映射。
- 若发送失败且没有 ACK，本地消息更新为 `failed`。

## 会话与未读

- 会话索引存 Redis ZSet。
- 会话详情存 Redis Hash。
- 收消息方未读数递增。
- 当前打开会话时前端应发送 ConversationReadReq。
- MarkRead 将未读数归零，并记录 lastReadMsgId。

## 群消息 fan-out

- 群消息仍先完成消息入队和 accepted ACK，fan-out 不得影响发送方 ACK。
- 群成员数达到 `fanout.large_group_threshold` 后，路由进入 Logic 内部有界 fan-out 队列，由独立 worker 异步扩散。
- fan-out 队列满时快速返回 `ErrFanoutQueueFull` 并记录日志，禁止无界排队。
- 小群或普通批量推送直接执行，但下行分发受 `fanout.dispatch_concurrency` 限制。
- 在线位置查询按用户批量 MGET，单批用户数由 `fanout.location_batch_size` 控制，批次并发由 `fanout.location_query_concurrency` 控制。
- 群成员列表使用短 TTL 内存缓存，默认 `500ms`；成员变更时必须调用缓存失效入口。

## 低时延要求

- SaveMessage 必须快速入队，队列满快速失败。
- accepted ACK 只依赖 Logic 入队成功，不等待 PostgreSQL。
- persisted ACK 由后台 batcher 在批写成功后异步发送。
- 群消息 fan-out 必须有 worker、队列、位置查询和下行分发并发上限。
- Redis 位置查询优先批量 MGET 和短 TTL 缓存。
- 会话更新可异步执行，不能阻塞发送 ACK。

## 关键代码

- `project/logic-go/internal/handler/chat_handler.go`
- `project/logic-go/internal/service/message_batcher.go`
- `project/logic-go/internal/service/router.go`
- `project/logic-go/internal/service/location.go`
- `project/logic-go/internal/service/conversation.go`
- `project/access-go/internal/handler/chat_handler.go`
- `project/access-go/internal/handler/downstream_handler.go`
- `project/desktop-web/src/stores/messageStore.ts`

## 相关指标

- 客户端发送到 accepted ACK 的 P50/P95/P99
- `MessageBatcher.Stats()` 提供批写队列长度、容量、队列满次数、失败批写次数、最近批写耗时和批大小。
- `RouterService.Stats()` 提供 fan-out 队列长度、容量和队列满丢弃次数。
- Redis 位置查询耗时
- fan-out 耗时
- 重复 clientMsgId 命中数
