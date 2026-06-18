# 存储与协议领域

## 领域职责

本领域记录数据库表、Redis key、NATS subject 和 FlatBuffers 协议边界，确保跨模块约定一致。

## PostgreSQL

核心表：
- `users`
- `friend_requests`
- `friends`
- `messages`
- `groups`
- `group_members`

表结构规则：
- 每张表必须包含 `id`、`created_at`、`updated_at`、`deleted`。
- `id` 使用雪花 ID。
- 禁止数据库外键，关系由应用层保证。
- 字符串字段 NOT NULL，默认空字符串。
- 修改 schema 必须同步 model。

消息幂等：
- `messages(from_user_id, client_msg_id)` 存在部分唯一索引，条件为 `client_msg_id != '' AND deleted = 0`。
- 客户端重试同一 `client_msg_id` 时，Logic 应复用同一 `serverMsgId`，不能重复路由或重复落库。

## Redis

Redis key 必须统一定义在 `project/shared/redis/keys.go`。

核心 key：
- `im:user:location:{userId}:{platform}`：用户在线位置。
- `conv:{userId}:idx`：会话索引。
- `conv:{userId}:p:{peerId}`：私聊会话详情。
- `conv:{userId}:g:{groupId}`：群聊会话详情。
- `user:info:{userId}`：用户基础信息。
- `user:token:{userId}:{platform}`：当前平台 token。
- `token:info:{accessToken}`：token 认证信息。
- `room:{roomId}`、`user_room:{userId}`、`room_users:{roomId}`、`room_lock:{roomId}`：房间扩展 key。

位置查询：
- Logic 查询多个用户在线位置时应按 `userId * platform` 构造 key 后使用 Redis MGET。
- 单批查询用户数由 `fanout.location_batch_size` 控制，批次并发由 `fanout.location_query_concurrency` 控制。
- 用户位置在 Logic 内有短 TTL 缓存，用于降低热会话 Redis 往返；Access 心跳和下线路径仍负责位置 key 生命周期。

群成员缓存：
- 群成员列表在 Logic 内使用短 TTL 内存缓存，默认 `fanout.group_member_cache_ttl = 500ms`。
- 查询群成员时必须过滤 `deleted = 0`。
- 群成员增删、退群、踢人、解散群等写路径必须失效对应 groupId 缓存。

## NATS

核心 subject：
- `im.logic.upstream`：Access 到 Logic 的普通上行消息。
- `im.access.{node_id}.downstream`：Logic 到指定 Access 节点的下行消息。
- `im.access.broadcast`：广播给所有 Access。
- `im.logic.room.{shard}`：Access 到 Logic 的房间有状态请求。
- `im.logic.game.{shard}`：Access 到 Logic 的游戏有状态请求。

room/game shard 规则：
- shard 由 `project/shared/nats/subjects.go` 统一计算和构造。
- 优先使用 `roomId` 计算；没有 `roomId` 时使用 `userId` fallback。
- `room_shard_count <= 1` 时 shard 固定为 `0`。
- Logic 对 room/game shard 使用普通订阅而不是队列组，保证同一 shard 的有状态请求由指定节点处理。
- 普通聊天和 room/game worker 隔离，避免房间热点影响消息主链路。

## FlatBuffers

Client 与 Access 使用 FlatBuffers。

帧格式：
- 4 bytes body length，大端。
- 1 byte frame type。
- body 为 FlatBuffers payload。

帧类型：
- `1`：AuthRequest。
- `2`：ClientRequest。
- `3`：AuthAck。
- `4`：ClientResponse。

修改 `schema/message.fbs` 后必须运行 `pnpm run flatc` 重新生成 TypeScript 代码。

ACK 状态：
- `ChatSendAck.status` 使用 `AckStatus`。
- `ACCEPTED` 表示 Logic 已接收并成功入队，客户端可先展示发送成功态。
- `PERSISTED` 表示后台批写已完成，客户端可清理未持久化标记。
- 空状态兼容为 `ACCEPTED`，未知状态映射为 `UNKNOWN`。

## JSON 内部事件

Access 与 Logic 当前使用 `shared/proto/messages.go` 中的 JSON struct。

注意：
- 内部 JSON 字段目前使用 PascalCase。
- 外部 REST JSON 使用 camelCase。
- 若迁移到 Protobuf 或 FlatBuffers，需要单独设计兼容方案。

## 低时延要求

- Redis 查询优先批量操作。
- 数据库写入优先批处理。
- 内部协议演进不能让主链路同步等待慢存储。
- 协议字段扩展需要兼容旧客户端。

## 关键文件

- `env/schema.sql`
- `schema/message.fbs`
- `project/shared/redis/keys.go`
- `project/shared/nats/subjects.go`
- `project/shared/proto/messages.go`
- `schema/generate-ts.sh`
- `schema/generate-go.sh`
