# 业务领域文档索引

本目录沉淀 IM-GO 的业务领域知识。`AGENTS.md` 只保留协作规则、开发规范和文档入口；涉及具体业务前必须阅读对应领域文档。

## 领域文档

- `overview.md`：系统业务全局、模块边界、低时延原则
- `access.md`：Access 接入层、连接生命周期、在线位置
- `message.md`：聊天消息、ACK、会话、未读、离线方向
- `user-auth.md`：用户、登录、Token、多端登录
- `friend.md`：好友请求和好友关系
- `room-game.md`：房间、游戏、麻将领域
- `client.md`：desktop-web 客户端体验和状态
- `storage-protocol.md`：数据存储、Redis key、NATS subject、FlatBuffers

## 使用规则

- 修改某个业务领域前，先阅读对应文档和相关架构文档。
- 若对话中确认了新的业务结论，及时更新对应领域文档。
- 每个领域文档保持 300 行以内，避免把实现细节无限堆进文档。
- 性能与低时延是最高准则，所有领域设计都必须保护消息主链路 P99。
