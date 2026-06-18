# IM-GO 业务总览

## 系统定位

IM-GO 是一个面向高并发、低时延场景的企业级即时通讯系统，目标是提供实时聊天、用户认证、好友关系、房间和游戏互动能力。

## 核心模块

- Access 层：维护 WebTransport/QUIC 长连接，处理认证、协议编解码、上行转发、下行推送和在线位置注册。
- Logic 层：处理 IM 业务逻辑，包括消息路由、会话更新、群消息扩散、用户上下线事件、房间和游戏编排。
- Web 层：提供 REST API，包括注册、登录、登出、用户信息、好友请求和好友列表。
- Desktop Web 客户端：React 客户端，负责登录、实时连接、聊天、好友、房间和麻将界面。
- Shared 模块：承载跨服务共享模型、Redis key、NATS subject、JWT、雪花 ID 等公共能力。

## 最高业务原则

- 实时性优先：消息主链路必须短、快、可背压。
- 非阻塞优先：慢操作异步化，不能阻塞消息读取、NATS 回调或连接写循环。
- 有界队列优先：队列满快速失败或降级，禁止无限等待。
- 状态新鲜优先：在线位置和路由缓存必须短 TTL，并在上线/下线事件中失效。
- 业务一致性服从主链路低时延：可靠性通过 accepted/persisted 分层 ACK、幂等去重和后台补偿实现。

## 业务链路概览

1. Web 登录后获取 accessToken 和 refreshToken。
2. 客户端用 accessToken 连接 Access 的 WebTransport endpoint。
3. Access 认证通过后注册用户在线位置到 Redis。
4. 客户端消息经 FlatBuffers 帧上行到 Access。
5. Access 将上行事件发布到 NATS。
6. Logic 消费消息、分配 serverMsgId、快速 ACK、异步持久化并路由接收方。
7. Access 根据 Logic 下行消息推送给对应连接。

## 相关代码入口

- `project/access-go/cmd/access/main.go`
- `project/logic-go/cmd/logic/main.go`
- `project/web-go/cmd/web/main.go`
- `project/desktop-web/src/App.tsx`
- `project/shared/`

## 相关架构文档

- `docs/access-layer-architecture.md`
- `docs/logic-layer-architecture.md`
- `docs/web-layer-architecture.md`
- `docs/web-client-architecture.md`
- `docs/game-module-design.md`
