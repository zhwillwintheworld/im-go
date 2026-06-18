# Access 接入领域

## 领域职责

Access 是实时连接网关，负责客户端 WebTransport/QUIC 长连接、认证首包、FlatBuffers 帧处理、用户位置注册、上行消息转发和下行消息推送。

Access 不负责复杂业务判断和持久化，必须保持无状态或轻状态，方便横向扩展。

## 核心流程

1. 客户端创建 WebTransport session。
2. 客户端创建首个双向 stream。
3. 首帧必须是 AuthRequest。
4. Access 通过 Redis 校验 token、deviceId 和 platform。
5. 认证成功后绑定连接和用户。
6. Access 写入 Redis 在线位置。
7. 后续请求通过同一双向 stream 收发。
8. 心跳刷新 Redis 位置 TTL。
9. 连接关闭或心跳超时后注销位置并通知 Logic。

## 接入保护配置

Access 连接保护由 `project/access-go/configs/config.yaml` 的 `server` 配置控制：

- `max_connections`：最大活跃连接数，`<= 0` 表示不限制；达到上限时，WebTransport upgrade 前快速返回 `503`，并在竞态场景下关闭已建立但尚未进入认证流程的 session。
- `max_frame_size`：最大帧体长度，默认 `1048576` 字节；认证首包和认证后的普通请求都必须先校验长度，再分配 body 内存，避免异常客户端造成内存放大。
- `allowed_origins`：浏览器 Origin 白名单；为空时兼容开发环境允许所有 Origin，配置 `*` 时显式允许所有 Origin，非空白名单按完整 Origin 精确匹配。无 Origin 的非浏览器请求允许通过，便于本地探测和集成测试。

## 在线位置

用户位置 key 由 `shared/redis/keys.go` 管理。

位置内容必须包含：
- `accessNodeId`
- `connId`
- `deviceId`
- `platform`
- `version`

注销和心跳续期必须校验当前连接版本，避免旧连接误删或续期新连接。

## 低时延要求

- NATS 回调不能阻塞。
- 下行写队列满时快速返回背压错误。
- 请求分发进入有界 worker pool，提交失败快速丢弃或降级。
- 入站帧必须限制最大长度，避免异常客户端造成内存放大。
- 所有服务端写客户端路径应统一为单 writer 队列，避免并发写 stream 造成 P99 抖动。
- 客户端请求响应和 Logic 下行推送都必须先封装完整 frame，再进入连接写队列；写队列满时快速丢弃并记录背压日志，不允许阻塞读循环或 worker。

## 安全边界

- 首帧必须认证。
- 认证失败快速关闭连接。
- 生产环境必须配置 `allowed_origins`，不能依赖开发环境的空白名单放行语义。
- 0-RTT 数据只允许幂等或可去重请求进入业务处理。
- token 不得明文记录到日志。

## 关键代码

- `project/access-go/internal/server/server.go`
- `project/access-go/internal/connection/connection.go`
- `project/access-go/internal/connection/manager.go`
- `project/access-go/internal/handler/auth_handler.go`
- `project/access-go/internal/handler/handler.go`
- `project/access-go/internal/handler/downstream_handler.go`
- `project/access-go/internal/redis/client.go`

## 相关指标

- 活跃连接数
- 认证耗时 P95/P99
- 帧解析耗时
- worker queue 长度和满队列次数
- 写队列长度和容量可通过 Connection 快照读取，背压通过 `ErrConnectionBackpressure` 日志和计数入口统计。
- 心跳超时数量
- Redis 位置注册、续期、注销耗时
