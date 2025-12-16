# IM Logic Layer 架构设计

基于 Go 的即时通讯系统逻辑层架构设计文档。

---

## 1. 系统概述

Logic Layer（逻辑层）是 IM 系统的业务核心，负责消息处理、用户管理、群组管理等核心业务逻辑。

### 1.1 核心职责

| 职责 | 描述 |
|------|------|
| 消息处理 | 消息存储、转发、离线消息管理 |
| 用户管理 | 用户状态、在线信息、多端同步 |
| 群组管理 | 群消息扩散、成员管理 |
| 消息路由 | 根据用户位置路由到正确的 Access 节点 |

### 1.2 技术选型

```
┌─────────────────────────────────────────────────────────┐
│                    技术栈                                │
├─────────────────┬───────────────────────────────────────┤
│ 语言            │ Go 1.25.5                               │
│ 内部通信        │ NATS (nats.go 官方客户端)              │
│ 数据库          │ PostgreSQL                            │
│ ORM             │ sqlc / pgx                            │
│ 缓存            │ go-redis                              │
│ 并发            │ Goroutines + Channels                 │
│ 日志            │ log/slog 或 zerolog                   │
│ 配置            │ Viper                                  │
└─────────────────┴───────────────────────────────────────┘
```

> [!IMPORTANT]
> **无 HTTP 服务**：本服务不包含任何 HTTP 服务器，通过 NATS 与 Access 层通信。
> 所有 I/O 操作均为非阻塞，充分利用 Go 的并发特性。

### 1.3 与 Access 层通信

```mermaid
graph LR
    subgraph AccessLayer["Access Layer (Go)"]
        A1[Access-1]
        A2[Access-2]
        A3[Access-N]
    end

    subgraph NATS["NATS Cluster"]
        N[NATS]
    end

    subgraph LogicLayer["Logic Layer (Go)"]
        L1[Logic-1]
        L2[Logic-2]
    end

    A1 & A2 & A3 -->|Publish| N
    N -->|QueueSubscribe| L1 & L2
    L1 & L2 -->|Publish| N
    N -->|Subscribe| A1 & A2 & A3
```

---

## 2. 整体架构

```mermaid
graph TB
    subgraph AccessLayer["Access Layer"]
        A1[Access-1]
        A2[Access-2]
    end

    subgraph NATS["NATS Cluster"]
        N[NATS]
    end

    subgraph LogicLayer["Logic Layer"]
        subgraph LogicNode["Logic Node"]
            NS[NATS Subscriber]
            MS[Message Service]
            US[User Service]
            GRS[Group Service]
            RS[Router Service]
        end
    end

    subgraph Storage["存储层"]
        DB[(PostgreSQL)]
        CACHE[(Redis)]
    end

    subgraph External["外部服务"]
        PUSH[Push Service]
        AUDIT[Audit Service]
    end

    A1 & A2 <-->|Publish/Subscribe| N
    N <--> NS
    NS --> MS & US & GRS
    MS & US & GRS --> RS
    RS --> CACHE
    MS --> DB
    MS --> PUSH
    MS --> AUDIT
```

---

## 3. 模块设计

### 3.1 项目目录结构

```
logic-go/
├── cmd/
│   └── logic/
│       └── main.go                 # 程序入口
├── configs/
│   └── config.yaml                 # 配置文件
├── internal/
│   ├── config/
│   │   └── config.go               # 配置加载
│   ├── nats/
│   │   ├── client.go               # NATS 客户端
│   │   ├── subscriber.go           # 消息订阅器
│   │   └── publisher.go            # 消息发布器
│   ├── service/
│   │   ├── message.go              # 消息业务
│   │   ├── user.go                 # 用户业务
│   │   ├── group.go                # 群组业务
│   │   └── router.go               # 路由服务
│   ├── repository/
│   │   ├── message.go
│   │   ├── user.go
│   │   └── group.go
│   └── model/
│       ├── message.go
│       ├── user.go
│       └── group.go
├── pkg/
│   └── proto/                      # Protobuf 生成的代码
├── go.mod
└── go.sum
```

### 3.2 核心模块详解

#### 3.2.1 NATS 消息服务

```mermaid
classDiagram
    class MessageSubscriber {
        -nc: *nats.Conn
        -messageService: *MessageService
        -routerService: *RouterService
        +Start()
        +handleUpstreamMessage(data []byte)
    }

    class MessagePublisher {
        -nc: *nats.Conn
        +PublishToAccess(accessNodeId string, message *pb.DownstreamMessage)
        +Broadcast(message *pb.DownstreamMessage)
    }

    class RouterService {
        -redisClient: *redis.Client
        -publisher: *MessagePublisher
        +GetUserLocations(userId int64) []UserLocation
        +RouteMessage(userId int64, message *pb.PushMessage)
        +RouteToMultiple(userIds []int64, message *pb.PushMessage)
    }

    MessageSubscriber --> RouterService
    RouterService --> MessagePublisher
```

#### 3.2.2 消息路由服务

```mermaid
classDiagram
    class RouterService {
        -redisClient: *redis.Client
        -publisher: *MessagePublisher
        +GetUserLocation(userId int64) []UserLocation
        +RouteMessage(userId int64, message *pb.PushMessage)
        +RouteToMultiple(userIds []int64, message *pb.PushMessage)
    }

    class UserLocation {
        +UserId: int64
        +AccessNodeId: string
        +ConnId: int64
        +DeviceId: string
        +Platform: string
        +LoginTime: time.Time
    }

    RouterService --> UserLocation
```

---

## 5. 核心代码实现

### 5.1 NATS 订阅者实现

```go
package nats

import (
    "context"
    "log/slog"

    "github.com/nats-io/nats.go"
    "google.golang.org/protobuf/proto"
)

type MessageSubscriber struct {
    nc             *nats.Conn
    messageService *service.MessageService
    userService    *service.UserService
    routerService  *service.RouterService
    logger         *slog.Logger
}

func NewMessageSubscriber(
    nc *nats.Conn,
    messageService *service.MessageService,
    userService *service.UserService,
    routerService *service.RouterService,
) *MessageSubscriber {
    return &MessageSubscriber{
        nc:             nc,
        messageService: messageService,
        userService:    userService,
        routerService:  routerService,
        logger:         slog.Default(),
    }
}

func (s *MessageSubscriber) Start(ctx context.Context) error {
    // 订阅上行消息 - 使用队列组实现负载均衡
    _, err := s.nc.QueueSubscribe("im.logic.upstream", "logic-group", func(msg *nats.Msg) {
        go s.handleUpstreamMessage(ctx, msg.Data)
    })
    if err != nil {
        return err
    }

    s.logger.Info("NATS subscriber started, listening on im.logic.upstream")
    return nil
}

func (s *MessageSubscriber) handleUpstreamMessage(ctx context.Context, data []byte) {
    var message pb.UpstreamMessage
    if err := proto.Unmarshal(data, &message); err != nil {
        s.logger.Error("Failed to unmarshal message", "error", err)
        return
    }

    accessNodeId := message.GetAccessNodeId()

    switch {
    case message.GetUserMessage() != nil:
        s.handleUserMessage(ctx, message.GetUserMessage(), accessNodeId)
    case message.GetUserOnline() != nil:
        s.handleUserOnline(ctx, message.GetUserOnline(), accessNodeId)
    case message.GetUserOffline() != nil:
        s.handleUserOffline(ctx, message.GetUserOffline(), accessNodeId)
    }
}

func (s *MessageSubscriber) handleUserMessage(ctx context.Context, msg *pb.UserMessage, accessNodeId string) {
    // 1. 消息存储
    serverMsgId, err := s.messageService.SaveMessage(ctx, msg)
    if err != nil {
        s.logger.Error("Failed to save message", "error", err)
        return
    }

    // 2. 发送 ACK 给发送者
    if err := s.routerService.SendAckToUser(ctx, msg.GetFromUserId(), msg.GetMsgId(), serverMsgId); err != nil {
        s.logger.Error("Failed to send ack", "error", err)
    }

    // 3. 路由消息给接收者
    if msg.GetToUserId() > 0 {
        s.routerService.RouteMessage(ctx, msg.GetToUserId(), msg, serverMsgId)
    } else if msg.GetToGroupId() > 0 {
        members, _ := s.groupService.GetGroupMembers(ctx, msg.GetToGroupId())
        // 过滤发送者
        filteredMembers := filterOut(members, msg.GetFromUserId())
        s.routerService.RouteToMultiple(ctx, filteredMembers, msg, serverMsgId)
    }
}

func (s *MessageSubscriber) handleUserOnline(ctx context.Context, event *pb.UserOnline, accessNodeId string) {
    s.userService.RegisterUserLocation(ctx, &model.UserLocation{
        UserId:       event.GetUserId(),
        AccessNodeId: accessNodeId,
        ConnId:       event.GetConnId(),
        DeviceId:     event.GetDeviceId(),
        Platform:     event.GetPlatform(),
    })
}

func (s *MessageSubscriber) handleUserOffline(ctx context.Context, event *pb.UserOffline, accessNodeId string) {
    s.userService.UnregisterUserLocation(ctx, event.GetUserId(), event.GetConnId(), accessNodeId)
}

func (s *MessageSubscriber) Stop() {
    s.logger.Info("NATS subscriber stopped")
}

func filterOut(members []int64, excludeId int64) []int64 {
    result := make([]int64, 0, len(members))
    for _, m := range members {
        if m != excludeId {
            result = append(result, m)
        }
    }
    return result
}
```

### 5.2 NATS 发布者 (用于下行推送)

```go
package nats

import (
    "fmt"
    "log/slog"

    "github.com/nats-io/nats.go"
    "google.golang.org/protobuf/proto"
)

type MessagePublisher struct {
    nc     *nats.Conn
    logger *slog.Logger
}

func NewMessagePublisher(nc *nats.Conn) *MessagePublisher {
    return &MessagePublisher{
        nc:     nc,
        logger: slog.Default(),
    }
}

// PublishToAccess 推送消息到指定 Access 节点
func (p *MessagePublisher) PublishToAccess(accessNodeId string, message *pb.DownstreamMessage) error {
    subject := fmt.Sprintf("im.access.%s.downstream", accessNodeId)
    data, err := proto.Marshal(message)
    if err != nil {
        return err
    }
    return p.nc.Publish(subject, data)
}

// Broadcast 广播消息到所有 Access 节点
func (p *MessagePublisher) Broadcast(message *pb.DownstreamMessage) error {
    data, err := proto.Marshal(message)
    if err != nil {
        return err
    }
    return p.nc.Publish("im.access.broadcast", data)
}
```

### 5.3 路由服务

```go
package service

import (
    "context"
    "encoding/json"
    "fmt"
    "log/slog"
    "sync"
    "time"

    "github.com/redis/go-redis/v9"
)

const (
    userLocationKeyPrefix = "im:user:location:"
    locationTTL           = 24 * time.Hour
)

type RouterService struct {
    redisClient *redis.Client
    publisher   *nats.MessagePublisher
    logger      *slog.Logger
}

type UserLocation struct {
    UserId       int64     `json:"userId"`
    AccessNodeId string    `json:"accessNodeId"`
    ConnId       int64     `json:"connId"`
    DeviceId     string    `json:"deviceId"`
    Platform     string    `json:"platform"`
    LoginTime    time.Time `json:"loginTime"`
}

func NewRouterService(redisClient *redis.Client, publisher *nats.MessagePublisher) *RouterService {
    return &RouterService{
        redisClient: redisClient,
        publisher:   publisher,
        logger:      slog.Default(),
    }
}

// GetUserLocations 获取用户所在的 Access 节点
func (s *RouterService) GetUserLocations(ctx context.Context, userId int64) ([]UserLocation, error) {
    key := fmt.Sprintf("%s%d", userLocationKeyPrefix, userId)

    entries, err := s.redisClient.HGetAll(ctx, key).Result()
    if err != nil {
        return nil, err
    }

    locations := make([]UserLocation, 0, len(entries))
    for _, value := range entries {
        var loc UserLocation
        if err := json.Unmarshal([]byte(value), &loc); err != nil {
            continue
        }
        locations = append(locations, loc)
    }

    return locations, nil
}

// RegisterUserLocation 注册用户位置
func (s *RouterService) RegisterUserLocation(ctx context.Context, location *UserLocation) error {
    key := fmt.Sprintf("%s%d", userLocationKeyPrefix, location.UserId)
    field := fmt.Sprintf("%s:%d", location.AccessNodeId, location.ConnId)

    value, err := json.Marshal(location)
    if err != nil {
        return err
    }

    pipe := s.redisClient.Pipeline()
    pipe.HSet(ctx, key, field, value)
    pipe.Expire(ctx, key, locationTTL)
    _, err = pipe.Exec(ctx)

    s.logger.Debug("Registered user location",
        "userId", location.UserId,
        "accessNodeId", location.AccessNodeId)

    return err
}

// RemoveUserLocation 移除用户位置
func (s *RouterService) RemoveUserLocation(ctx context.Context, userId int64, accessNodeId string, connId int64) error {
    key := fmt.Sprintf("%s%d", userLocationKeyPrefix, userId)
    field := fmt.Sprintf("%s:%d", accessNodeId, connId)

    s.logger.Debug("Unregistered user location",
        "userId", userId,
        "connId", connId)

    return s.redisClient.HDel(ctx, key, field).Err()
}

// RouteMessage 路由消息到用户
func (s *RouterService) RouteMessage(ctx context.Context, userId int64, message *pb.PushMessage) error {
    locations, err := s.GetUserLocations(ctx, userId)
    if err != nil {
        return err
    }

    if len(locations) == 0 {
        s.logger.Debug("User is offline, saving to offline storage", "userId", userId)
        // TODO: offlineMessageService.Save(userId, message)
        return nil
    }

    // 按 Access 节点分组并行推送
    nodeLocations := make(map[string][]UserLocation)
    for _, loc := range locations {
        nodeLocations[loc.AccessNodeId] = append(nodeLocations[loc.AccessNodeId], loc)
    }

    var wg sync.WaitGroup
    for accessNodeId := range nodeLocations {
        wg.Add(1)
        go func(nodeId string) {
            defer wg.Done()
            downstreamMsg := &pb.DownstreamMessage{
                Payload: &pb.DownstreamMessage_PushMessage{
                    PushMessage: message,
                },
            }
            if err := s.publisher.PublishToAccess(nodeId, downstreamMsg); err != nil {
                s.logger.Warn("Failed to route message to access node",
                    "accessNodeId", nodeId,
                    "error", err)
            }
        }(accessNodeId)
    }
    wg.Wait()

    return nil
}

// RouteToMultiple 批量路由消息（群消息）- 并行处理
func (s *RouterService) RouteToMultiple(ctx context.Context, userIds []int64, message *pb.PushMessage) error {
    // 并行获取所有用户位置
    type userLoc struct {
        userId    int64
        locations []UserLocation
    }

    results := make(chan userLoc, len(userIds))
    var wg sync.WaitGroup

    for _, userId := range userIds {
        wg.Add(1)
        go func(uid int64) {
            defer wg.Done()
            locs, _ := s.GetUserLocations(ctx, uid)
            results <- userLoc{userId: uid, locations: locs}
        }(userId)
    }

    go func() {
        wg.Wait()
        close(results)
    }()

    // 按 Access 节点分组
    nodeToUsers := make(map[string][]int64)
    for result := range results {
        for _, loc := range result.locations {
            nodeToUsers[loc.AccessNodeId] = append(nodeToUsers[loc.AccessNodeId], result.userId)
        }
    }

    // 并行发送
    var sendWg sync.WaitGroup
    for accessNodeId, users := range nodeToUsers {
        sendWg.Add(1)
        go func(nodeId string, targetUsers []int64) {
            defer sendWg.Done()
            for _, userId := range targetUsers {
                downstreamMsg := &pb.DownstreamMessage{
                    Payload: &pb.DownstreamMessage_PushMessage{
                        PushMessage: message,
                    },
                }
                s.publisher.PublishToAccess(nodeId, downstreamMsg)
            }
        }(accessNodeId, users)
    }
    sendWg.Wait()

    return nil
}
```

---

## 6. 核心流程

### 6.1 Access 节点连接流程

```mermaid
sequenceDiagram
    participant A as Access Node
    participant NATS as NATS Cluster
    participant L as Logic Service

    A->>NATS: 连接并订阅 im.access.{nodeId}.downstream
    L->>NATS: 连接并订阅 im.logic.upstream (队列组)
    A->>NATS: Publish AccessNodeOnline
    NATS->>L: 消息分发
    L->>L: 记录 Access 节点信息

    loop 消息交互
        A->>NATS: Publish UpstreamMessage
        NATS->>L: 队列分发
        L->>NATS: Publish DownstreamMessage
        NATS->>A: 消息推送
    end

    A->>NATS: Publish AccessNodeOffline
    NATS->>L: 消息分发
    L->>L: 清理 Access 节点信息
```

### 6.2 单聊消息处理流程

```mermaid
sequenceDiagram
    participant C1 as Client A
    participant A1 as Access-1
    participant L as Logic
    participant A2 as Access-2
    participant C2 as Client B

    C1->>A1: ChatMessage (to: B)
    A1->>L: UpstreamMessage.UserMessage
    L->>L: 存储消息
    L-->>A1: DownstreamMessage.MessageAck
    A1-->>C1: MessageAck

    L->>L: 查询 B 的位置 (Redis)
    L->>A2: DownstreamMessage.PushMessage
    A2->>C2: ChatMessage
    C2-->>A2: MessageAck
    A2->>L: UpstreamMessage.MessageAck
    L->>L: 更新消息状态
```

### 6.3 群聊消息处理流程

```mermaid
sequenceDiagram
    participant C1 as Client A
    participant A1 as Access-1
    participant L as Logic
    participant A2 as Access-2
    participant A3 as Access-3

    C1->>A1: GroupMessage (group: 123)
    A1->>L: UpstreamMessage.UserMessage (toGroupId=123)
    L->>L: 存储消息

    L-->>A1: MessageAck
    A1-->>C1: MessageAck

    L->>L: 获取群成员列表
    L->>L: 按 Access 节点分组

    par 并行推送
        L->>A1: PushMessage (user B)
        L->>A2: PushMessage (user C, D)
        L->>A3: PushMessage (user E)
    end
```

---

## 7. 关键设计

### 7.1 连接管理

| 场景 | 处理方式 |
|------|----------|
| Access 断线 | NATS 自动清理订阅，用户位置保留到 TTL |
| Logic 重启 | NATS 队列组自动负载均衡 |
| 用户多端 | 同一用户多个 Location，按需推送 |

### 7.2 消息可靠性

```mermaid
flowchart TB
    A[收到消息] --> B[存储到 DB]
    B --> C[发送 ACK 给发送者]
    C --> D{接收者在线?}
    D -->|是| E[推送消息]
    D -->|否| F[存储离线消息]
    E --> G{推送成功?}
    G -->|是| H[等待客户端 ACK]
    G -->|否| I[重试/存离线]
    H -->|超时| I
```

### 7.3 群消息扩散策略

| 策略 | 适用场景 | 实现 |
|------|----------|------|
| 写扩散 | 小群 (<100人) | 为每个在线成员发送推送 |
| 读扩散 | 大群 (>100人) | 只推送通知，客户端拉取 |
| 混合 | 通用 | 在线成员写扩散，离线成员读扩散 |

---

## 8. 配置示例

### 8.1 config.yaml

```yaml
# Logic 服务配置
app:
  name: im-logic
  log_level: debug

# NATS 配置
nats:
  url: nats://localhost:4222
  # cluster:
  #   urls:
  #     - nats://nats-1:4222
  #     - nats://nats-2:4222
  #     - nats://nats-3:4222
  max_reconnects: -1  # 无限重连
  reconnect_wait: 2s

# PostgreSQL
database:
  host: localhost
  port: 5432
  name: im
  user: postgres
  password: password
  max_open_conns: 50
  max_idle_conns: 10
  conn_max_lifetime: 30m

# Redis
redis:
  host: localhost
  port: 6379
  password: ""
  db: 0
  pool_size: 50
  # cluster:
  #   addrs:
  #     - redis-1:6379
  #     - redis-2:6379
  #     - redis-3:6379
```

### 8.2 main.go

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/nats-io/nats.go"
    "github.com/redis/go-redis/v9"
    "github.com/spf13/viper"
)

func main() {
    // 初始化日志
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    slog.SetDefault(logger)

    // 加载配置
    cfg := loadConfig()

    // 连接 NATS
    nc, err := connectNATS(cfg.NATS)
    if err != nil {
        logger.Error("Failed to connect to NATS", "error", err)
        os.Exit(1)
    }
    defer nc.Close()

    // 连接 Redis
    redisClient := connectRedis(cfg.Redis)
    defer redisClient.Close()

    // 连接数据库
    db := connectDatabase(cfg.Database)
    defer db.Close()

    // 初始化服务
    publisher := nats.NewMessagePublisher(nc)
    routerService := service.NewRouterService(redisClient, publisher)
    userService := service.NewUserService(redisClient)
    messageService := service.NewMessageService(db)

    // 启动订阅者
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    subscriber := nats.NewMessageSubscriber(nc, messageService, userService, routerService)
    if err := subscriber.Start(ctx); err != nil {
        logger.Error("Failed to start subscriber", "error", err)
        os.Exit(1)
    }

    logger.Info("Logic service started")

    // 优雅退出
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    logger.Info("Shutting down...")
    subscriber.Stop()
}

func connectNATS(cfg NATSConfig) (*nats.Conn, error) {
    opts := []nats.Option{
        nats.MaxReconnects(cfg.MaxReconnects),
        nats.ReconnectWait(cfg.ReconnectWait),
        nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
            slog.Warn("Disconnected from NATS", "error", err)
        }),
        nats.ReconnectHandler(func(nc *nats.Conn) {
            slog.Info("Reconnected to NATS", "url", nc.ConnectedUrl())
        }),
    }
    return nats.Connect(cfg.URL, opts...)
}

func connectRedis(cfg RedisConfig) *redis.Client {
    return redis.NewClient(&redis.Options{
        Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
        Password: cfg.Password,
        DB:       cfg.DB,
        PoolSize: cfg.PoolSize,
    })
}
```

### 8.3 go.mod

```go
module github.com/yourorg/im-logic

go 1.21

require (
    github.com/nats-io/nats.go v1.31.0
    github.com/redis/go-redis/v9 v9.3.0
    github.com/jackc/pgx/v5 v5.5.0
    github.com/spf13/viper v1.17.0
    google.golang.org/protobuf v1.31.0
)
```

---

## 9. 监控与运维

### 9.1 关键指标

| 指标 | 描述 |
|------|------|
| `logic_nats_connections_active` | 当前活跃的 NATS 连接数 |
| `logic_messages_processed` | 处理的消息数 (按类型) |
| `logic_message_latency` | 消息处理延迟 |
| `logic_route_failures` | 路由失败次数 |
| `logic_db_query_time` | 数据库查询耗时 |

### 9.2 健康检查

```go
package health

import (
    "context"
    "net/http"
    "time"

    "github.com/nats-io/nats.go"
    "github.com/redis/go-redis/v9"
)

type HealthChecker struct {
    nc          *nats.Conn
    redisClient *redis.Client
}

func NewHealthChecker(nc *nats.Conn, redisClient *redis.Client) *HealthChecker {
    return &HealthChecker{nc: nc, redisClient: redisClient}
}

func (h *HealthChecker) Check(ctx context.Context) map[string]string {
    status := make(map[string]string)

    // 检查 NATS
    if h.nc.IsConnected() {
        status["nats"] = "connected"
    } else {
        status["nats"] = "disconnected"
    }

    // 检查 Redis
    ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
    defer cancel()

    if err := h.redisClient.Ping(ctx).Err(); err == nil {
        status["redis"] = "connected"
    } else {
        status["redis"] = "disconnected"
    }

    return status
}

// ServeHTTP 可选的 HTTP 健康检查端点
func (h *HealthChecker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    status := h.Check(r.Context())
    for k, v := range status {
        if v != "connected" {
            w.WriteHeader(http.StatusServiceUnavailable)
            break
        }
    }
    json.NewEncoder(w).Encode(status)
}
```

---

## 10. 后续演进

- [ ] 消息存储分库分表
- [ ] 群消息读扩散优化
- [ ] 消息搜索 (Elasticsearch)
- [ ] 已读回执批量处理
- [ ] 消息撤回机制
- [ ] 端到端加密支持
