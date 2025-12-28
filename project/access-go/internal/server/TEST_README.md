# WebTransport 认证功能测试

本目录包含 access-go 服务的 WebTransport 认证功能测试。

## 测试文件

### 1. auth_test.go - 单元测试

测试 FlatBuffers 消息构建、帧格式和解析功能，不需要外部依赖。

**测试用例：**

- `TestBuildAuthRequest` - 测试构建认证请求
- `TestBuildAuthFrame` - 测试构建认证帧（包含帧头）
- `TestParseAuthResponse` - 测试解析认证响应
- `TestAuthFrameReadWrite` - 测试认证帧的读写流程

**运行方法：**

```bash
# 运行所有单元测试
go test -v ./internal/server -run Test

# 运行特定测试
go test -v ./internal/server -run TestBuildAuthRequest
go test -v ./internal/server -run TestBuildAuthFrame
go test -v ./internal/server -run TestParseAuthResponse
go test -v ./internal/server -run TestAuthFrameReadWrite
```

### 2. server_test.go - 集成测试

测试完整的 WebTransport 连接和认证流程，需要 Redis 和 NATS 服务。

**测试用例：**

- `TestWebTransportAuth` - 测试认证成功场景
- `TestWebTransportAuthFail` - 测试认证失败场景

**前置条件：**

1. Redis 服务运行在 `localhost:6379`，密码为 `xhxxygwl`
2. NATS 服务运行在 `localhost:4222`
3. 设置环境变量 `INTEGRATION_TEST=1`

**运行方法：**

```bash
# 设置环境变量并运行集成测试
INTEGRATION_TEST=1 go test -v ./internal/server -run TestWebTransport

# 运行特定集成测试
INTEGRATION_TEST=1 go test -v ./internal/server -run TestWebTransportAuth
INTEGRATION_TEST=1 go test -v ./internal/server -run TestWebTransportAuthFail
```

## 测试覆盖范围

### 认证流程测试

1. **消息构建** - 使用 FlatBuffers 构建 AuthRequest
2. **帧格式** - 验证帧头（4 bytes 长度 + 1 byte 类型）和帧体
3. **连接建立** - 建立 WebTransport 连接
4. **认证请求** - 发送认证帧到服务器
5. **响应解析** - 读取并解析认证响应
6. **状态验证** - 验证用户位置在 Redis 中注册

### 认证失败场景

- 使用无效 token 连接
- 验证服务器返回正确的错误码

## 协议格式

### 认证请求帧

```
[4 bytes: 长度] [1 byte: 帧类型=1] [N bytes: AuthRequest FlatBuffers]
```

### AuthRequest (FlatBuffers)

- `token`: string - 用户 token
- `device_id`: string - 设备 ID
- `platform`: Platform枚举 - 平台类型（WEB, ANDROID, IOS, DESKTOP, WECHAT）
- `app_version`: string - 应用版本

### 认证响应帧

```
[4 bytes: 长度] [1 byte: 帧类型=4] [N bytes: ClientResponse FlatBuffers]
```

### ClientResponse (FlatBuffers)

- `req_id`: string - 请求 ID（认证响应为空）
- `timestamp`: int64 - 时间戳
- `code`: ErrorCode枚举 - 错误码（SUCCESS=0 表示成功）
- `msg`: string - 消息
- `payload_type`: ResponsePayload枚举 - 响应负载类型
- `payload`: bytes - 响应负载数据

## 注意事项

1. **测试端口** - 集成测试使用端口 `18081` 避免与开发服务器冲突
2. **证书文件** - 测试使用 `localhost+2.pem` 和 `localhost+2-key.pem` 证书
3. **测试数据** - 集成测试会在 Redis 中创建临时数据，测试结束后会清理
4. **跳过集成测试** - 默认跳过集成测试，需要设置 `INTEGRATION_TEST=1` 环境变量

## 示例输出

### 单元测试成功

```
=== RUN   TestBuildAuthRequest
    auth_test.go:60: 认证请求构建成功，大小: 84 bytes
--- PASS: TestBuildAuthRequest (0.00s)
```

### 集成测试成功

```
=== RUN   TestWebTransportAuth
    server_test.go:92: WebTransport 连接建立成功
    server_test.go:109: 认证请求已发送
    server_test.go:120: 认证成功！错误码: SUCCESS
    server_test.go:132: 用户位置已注册: test-access-1
--- PASS: TestWebTransportAuth (2.15s)
```

## 扩展测试

如果需要添加更多测试场景，可以考虑：

1. **设备不匹配** - token 对应的设备 ID 与请求不符
2. **平台不匹配** - token 对应的平台与请求不符
3. **Token 过期** - token 已被替换（用户在其他设备登录）
4. **并发连接** - 测试同一用户多个连接的情况
5. **心跳测试** - 测试认证后的心跳请求
