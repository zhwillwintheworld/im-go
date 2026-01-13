---
trigger: always_on
---

# IM 项目开发规范

## ⚠️ 核心规则（必须严格遵守）

1. **非阻塞要求（最高优先级）**: 在 access、logic、web 这三个模块中，绝对不允许阻塞消息处理，只需要打印日志即可。如果修改或生成的代码阻塞了消息处理，必须立即通知用户
2. **质量检查要求**:
   - Go 模块（access-go/logic-go/web-go）修改后必须运行 `scripts/check-go-quality.sh` 并解决所有错误与警告
   - desktop-web 模块修改后必须运行 `scripts/check-web-quality.sh` 并解决所有错误与警告
   - 校验代码时不要生成编译产物
3. **文件清理**: 生成的 .bak 备份文件必须删除
4. **Redis 键管理**: 所有 Redis 键值操作必须在 shared/redis/keys.go 中定义，绝不在应用代码中硬编码
5. **JSON 字段规范**:
   - Go 后端输出的 JSON 字段使用驼峰格式（camelCase）
   - web-go 返回的 response 对象中 id 字段必须为 string（因为前端 JS 会丢精度）
6. **协议同步**: 修改 schema/message.fbs 后，必须运行 `npm run flatc` 重新生成 TypeScript 代码

## 数据库规范

7. **表结构标准字段**: 每个表必须包含以下字段
   - `id`: 使用雪花 ID
   - `created_at`: 创建时间（timestamp）
   - `updated_at`: 修改时间（timestamp）
   - `deleted`: 逻辑删除标志（0=正常，1=已删除）

8. **Schema 与 Model 同步**:
   - 修改 schema.sql 必须同步更新对应的 model
   - 修改 model 必须同步更新 schema.sql
   - 新增 model 必须在 schema 中新增表
   - 新增表必须生成对应的 model

9. **外键禁令**: 绝对不允许使用数据库外键，可以在字段注释中说明关系

10. **字段要求**:
    - 每个字段必须有注释描述其用途
    - 字符串字段必须设置为 NOT NULL，默认值为空字符串（`DEFAULT ''`）
    - 判断字符串是否为空使用 `column != ''`，不要使用 `column = NULL`

## Go 代码规范

### 错误处理

11. 使用 `errors.Is()` 和 `errors.As()` 进行错误判断，不要使用 `==` 比较错误
12. 不要忽略错误，至少要记录日志
13. 关键路径的错误必须包含上下文信息
    ```go
    return fmt.Errorf("failed to save message: %w", err)
    ```

### 并发安全

14. 共享数据必须使用互斥锁（sync.Mutex/RWMutex）或 channel 保护
15. 避免在 goroutine 中直接使用闭包变量，应该作为参数传递
    ```go
    // 正确做法
    for _, item := range items {
        go func(i Item) {
            process(i)
        }(item)
    }
    ```
16. 使用 `context.Context` 传递取消信号和超时控制

### 资源管理

17. 使用 `defer` 确保资源释放（文件、连接、锁等）
18. 数据库连接、文件句柄、HTTP 响应体必须正确关闭
    ```go
    resp, err := http.Get(url)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    ```

### 命名规范

19. 导出的函数和类型使用大写字母开头（如 `SaveMessage`）
20. 私有的函数和类型使用小写字母开头（如 `validateInput`）
21. 接口名称通常以 `er` 结尾（如 `Reader`、`Writer`、`MessageHandler`）
22. 避免使用下划线分隔，使用驼峰命名
23. 常量使用全大写加下划线（如 `MAX_RETRY_COUNT`）

### 日志规范

24. 使用结构化日志（slog），不要使用 fmt.Println
25. 日志级别：Debug、Info、Warn、Error
26. 包含必要的上下文信息（userId、requestId、messageId 等）
    ```go
    slog.Error("failed to send message", "userId", userId, "error", err)
    ```

### 函数设计

27. 函数保持简洁，单一职责，建议不超过 50 行
28. 参数不要超过 5 个，超过则使用结构体
29. 返回值统一使用 `(result, error)` 模式
30. 避免使用 panic，除非是真正不可恢复的错误

### 测试规范

31. 单元测试文件以 `_test.go` 结尾
32. 测试函数以 `Test` 开头（如 `TestSaveMessage`）
33. 使用 table-driven tests 进行多场景测试
34. Mock 外部依赖（数据库、Redis、HTTP 等）

## React/TypeScript 开发规范

### 类型安全

35. 使用 TypeScript 严格模式（tsconfig.json 中 `strict: true`）
36. 禁止使用 `any` 类型，除非有充分理由并添加注释说明
37. 优先使用类型推断
38. 使用类型守卫（type guard）处理联合类型

### 组件规范

39. 使用函数组件和 Hooks，不要使用类组件
40. 组件文件名使用 PascalCase（如 `UserProfile.tsx`）
41. 每个组件一个文件
42. 复杂组件拆分为更小的子组件，保持单一职责

### 状态管理

43. 全局状态使用 Zustand 管理
44. 局部状态使用 `useState` 和 `useReducer`
45. 避免 prop drilling，超过 3 层使用 Context 或全局状态
46. 状态更新保持不可变性（使用扩展运算符或 immer）

### 性能优化

47. 使用 `React.memo` 避免不必要的重渲染（但不要过度使用）
48. 使用 `useMemo` 缓存计算结果，`useCallback` 缓存函数引用
49. 列表渲染必须使用稳定的 `key`（不要使用 index）
50. 大列表使用虚拟滚动（如 react-window）

### 代码风格

51. 使用 ESLint 和 Prettier 自动格式化代码
52. 使用解构赋值提取 props 和 state
53. 避免嵌套三元表达式，超过一层使用 if-else 或提取函数
54. 函数保持简洁，单一职责，建议不超过 30 行

### 异步处理

55. 使用 `async/await` 处理异步操作，不要使用 `.then()`
56. 正确处理 Loading、Error、Success 三种状态
57. 使用 try-catch 捕获异常并显示友好的错误信息
58. 组件卸载时取消未完成的异步请求（使用 AbortController）

### 样式规范

59. 优先使用 Ant Design 组件库，不要重复造轮子
60. 自定义样式使用 CSS Modules（.module.css）
61. 避免内联样式，除非是动态样式
62. 使用 CSS 变量定义主题色、间距等

### Hooks 使用规范

63. Hooks 必须在组件顶层调用，不能在条件语句中
64. 自定义 Hooks 以 `use` 开头（如 `useWebTransport`）
65. useEffect 依赖数组必须正确声明所有依赖
66. 清理副作用使用 useEffect 返回清理函数

### 目录结构

67. desktop-web 项目目录结构：
    ```
    src/
    ├── components/      # 通用可复用组件
    ├── pages/          # 页面级组件
    ├── stores/         # Zustand 全局状态
    ├── hooks/          # 自定义 Hooks
    ├── services/       # API 服务层
    ├── utils/          # 工具函数
    ├── types/          # TypeScript 类型定义
    └── constants/      # 常量定义
    ```

## Git 提交规范

68. **提交信息格式**:
    ```
    <type>(<scope>): <subject>

    <body>

    <footer>
    ```

69. **Type 类型**:
    - `feat`: 新功能
    - `fix`: 修复 Bug
    - `docs`: 文档更新
    - `style`: 代码格式（不影响代码运行）
    - `refactor`: 重构（既不是新增功能，也不是修复 Bug）
    - `perf`: 性能优化
    - `test`: 测试相关
    - `chore`: 构建过程或辅助工具的变动

70. **提交示例**:
    ```
    feat(access): 添加 WebTransport 连接支持

    - 实现 QUIC 连接处理
    - 添加心跳机制
    - 支持多端登录

    Closes #123
    ```

## 开发流程注意事项

71. 修改代码前先阅读相关的架构文档（docs/ 目录）
72. 提交代码前运行对应的质量检查脚本
73. 重要功能必须编写单元测试
74. API 变更必须更新 Swagger 文档
75. 协议变更必须同步更新前后端代码

---

## 项目架构说明

76. 项目分为四个模块：
    - **access-go**: 持有用户长连接
    - **logic-go**: 处理 IM 逻辑
    - **web-go**: REST API 服务
    - **desktop-web**: 客户端应用

77. **access-go**: go + webtransport + flatbuffers 构建用户交互服务器
    - 持有用户连接
    - 使用 redis 存储用户 location
    - 使用 nats 作为 broker 分发数据到 logic
    - 接收 logic 下发到 nats 的 message 再发往用户

78. **logic-go**: go + redis + postgresql
    - redis 存储路由热点数据
    - pg 存储消息

79. **web-go**: gin + redis + pg
    - redis 存储 token 信息
    - pg 存储业务数据（好友、用户、战绩等）

80. **desktop-web**: react + tsx + typescript + vite + webtransport + antd
    - 前端客户端应用
    - 与 IM 系统通信
