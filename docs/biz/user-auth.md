# 用户与认证领域

## 领域职责

用户与认证领域负责注册、登录、Token 签发、Token 存储、登出、多端登录和 Access 连接认证。

## 注册

用户注册通过 Web 层 REST API 完成。

核心规则：
- 用户名唯一。
- 密码使用 bcrypt 哈希。
- 用户 ID 使用雪花 ID。
- 用户基础信息可缓存到 Redis，供房间、好友或展示信息读取。

## 登录

登录成功后 Web 层生成 accessToken 和 refreshToken。

Redis 中需要维护：
- `user:token:{userId}:{platform}` 到当前 accessToken 的映射。
- `token:info:{accessToken}` 到 userId、deviceId、platform 的映射。

同一用户同一 platform 只保留一个当前 token，新登录会替换旧 token。

## Access 认证

客户端连接 Access 后首帧发送 AuthRequest。

Access 校验：
- token 是否存在于 Redis。
- deviceId 是否匹配。
- platform 是否匹配。
- token 是否是该用户该 platform 当前有效 token。

认证成功后 Access 才绑定连接并注册在线位置。

## 多端登录

- 同一用户不同 platform 可以同时在线。
- 同一用户同一 platform 新连接会踢掉旧连接。
- 位置 key 按 userId + platform 维护。

## 低时延要求

- Access 认证只做 Redis 快速校验，不访问 PostgreSQL。
- 登录接口可以访问数据库，但连接认证路径不能访问慢存储。
- Token 自动续期不能阻塞主链路。

## 安全要求

- Token 不记录明文日志。
- REST Authorization 应兼容 Bearer 格式。
- refresh token 换新 access token 后必须写回 Redis，否则 Access 无法识别新 token。
- 登出必须删除 token info，并保证当前 token 映射语义一致。

## 关键代码

- `project/web-go/internal/service/auth.go`
- `project/web-go/internal/repository/token.go`
- `project/web-go/internal/middleware/auth.go`
- `project/web-go/internal/handler/auth.go`
- `project/access-go/internal/handler/auth_handler.go`
- `project/shared/jwt/jwt.go`
- `project/shared/redis/keys.go`

## 相关指标

- 登录耗时：Web 层登录超过 `100ms` 时记录慢日志。
- Redis token 查询耗时：TokenAuth 中间件查询超过 `20ms` 时记录慢日志。
- Access 认证耗时
- token 不匹配拒绝数
- 同平台踢旧连接次数
