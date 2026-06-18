# Desktop Web 客户端领域

## 领域职责

desktop-web 负责用户登录、实时连接、消息展示、好友入口、房间和麻将交互。

## 技术边界

- React + TypeScript + Vite。
- Ant Design 作为 UI 组件库。
- Zustand 管理全局状态。
- FlatBuffers 编解码实时消息。
- WebTransport 维护实时连接。

## 连接流程

1. 用户登录 Web API。
2. 保存 accessToken 和 refreshToken。
3. IMProvider 监听认证状态。
4. WebTransportManager 建立连接。
5. 创建双向 stream。
6. 发送 AuthRequest。
7. 认证成功后进入 authenticated 状态。
8. 启动心跳。

## 消息发送状态

建议状态：
- `pending`：本地已创建，尚未收到 ACK。
- `accepted`：Logic 已接收并分配 serverMsgId。
- `persisted`：服务端已持久化。
- `failed`：发送或服务端处理失败。

前端需要维护 `reqId -> localMsgId -> serverMsgId` 映射，ACK 到达后更新对应消息。发送时保留原始 frame，直到收到 `persisted` ACK 后删除。

## 重连恢复

- WebTransport 重连成功进入 `connected` 后，messageStore 会检查未收到 `persisted` ACK 的消息。
- 未确认消息使用原始 frame 和同一 `reqId` 重发，服务端应按同一 `clientMsgId` 命中幂等逻辑。
- 恢复发送失败时保留 pending 信息，等待下一次重连继续恢复；不要在瞬时网络抖动时立即标记 failed。
- 当前恢复范围是内存态会话内的未确认消息；刷新页面后的 IndexedDB 恢复属于后续扩展。

## 房间和游戏

- 房间请求通过 FlatBuffers RoomReq 发送。
- 换座位必须写入 `targetSeatIndex`，`-1` 表示不指定座位。
- 房间推送更新房间状态。
- 游戏推送进入麻将页面状态机。

## 低时延要求

- 用户点击发送后立即本地显示 pending。
- WebTransport 写入失败才标记 failed。
- accepted ACK 到达后立刻更新状态。
- IndexedDB、历史消息加载、离线恢复不能阻塞 UI。
- 重连逻辑采用指数退避，但不能无限堆积发送任务。
- 重连恢复只重发已有 pending frame，不重新生成 reqId。

## 关键代码

- `project/desktop-web/src/components/IMProvider.tsx`
- `project/desktop-web/src/services/transport/WebTransportManager.ts`
- `project/desktop-web/src/services/protocol/IMProtocol.ts`
- `project/desktop-web/src/services/messageDispatcher.ts`
- `project/desktop-web/src/stores/authStore.ts`
- `project/desktop-web/src/stores/imStore.ts`
- `project/desktop-web/src/stores/messageStore.ts`
- `project/desktop-web/src/stores/chatStore.ts`
- `project/desktop-web/src/services/mahjongRoomService.ts`

## 相关指标

- WebTransportLatencyAnalyzer 记录 WebTransport 连接耗时、认证耗时和消息发送到 ACK 延迟。
- LatencyMonitor 展示 ACK 平均/P95/P99、连接平均、认证平均和待确认请求数。
- 重连次数
- 未确认消息数量
