---
name: 代码质量检查
description: IM-GO 项目 Go 和 TypeScript 代码质量检查工作流
---

# 代码质量检查技能

本技能提供 IM-GO 项目的代码质量检查工作流程,包括 Go 模块和 TypeScript 模块的质量检查。

## 适用场景

- 修改 Go 代码后 (access-go, logic-go, web-go, shared 模块)
- 修改 TypeScript/React 代码后 (desktop-web 模块)
- 提交代码前的必要检查
- CI/CD 流程中的质量门禁

## 核心原则

⚠️ **最高优先级规则**:
1. 所有 ERROR 必须修复才能提交代码
2. WARNING 建议修复以提高代码质量
3. 不要生成编译产物到代码库
4. 检查完成后删除所有 `.bak` 备份文件

---

## Go 代码质量检查

### 适用模块

- `project/access-go` - Access 层
- `project/logic-go` - Logic 层  
- `project/web-go` - Web 层
- `project/shared` - 共享模块

### 执行步骤

#### 1. 运行质量检查脚本

```bash
# 在项目根目录执行
./scripts/check-go-quality.sh
```

#### 2. 检查项说明

脚本会依次执行以下检查:

**a. 代码格式化 (`go fmt`)**
- 检查代码格式是否符合 Go 标准
- 自动修复格式问题

**b. 静态代码分析 (`go vet`)**
- 检查常见的代码错误
- 发现可能的 bug 和可疑的构造

**c. 代码质量检查 (`golangci-lint`)**
- 执行多个 linter 检查
- 检查代码规范、潜在问题、性能问题等
- 如果未安装,需要先安装:
  ```bash
  # macOS
  brew install golangci-lint
  
  # 或使用 go install
  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
  ```

**d. 单元测试 (`go test`)**
- 运行所有单元测试
- 确保代码修改没有破坏现有功能

**e. 编译检查 (`go build`)**
- 验证代码可以正常编译
- ⚠️ **注意**: 不会生成二进制文件(不使用 `-o` 参数)

#### 3. 修复问题

**ERROR 级别**:
```bash
# 示例错误
ERROR: undefined: someVariable
ERROR: syntax error: unexpected {

# 必须修复才能提交
```

**WARNING 级别**:
```bash
# 示例警告
WARNING: exported function should have comment
WARNING: unused variable or import

# 建议修复以提高代码质量
```

#### 4. 清理备份文件

```bash
# 查找并删除 .bak 文件
find . -name "*.bak" -delete

# 或手动删除
rm project/access-go/**/*.bak
rm project/logic-go/**/*.bak
rm project/web-go/**/*.bak
```

### 单独执行某项检查

如果只需要执行特定检查:

```bash
# 格式化代码
cd project/web-go
go fmt ./...

# 静态分析
go vet ./...

# linter 检查
golangci-lint run

# 运行测试
go test ./...

# 测试覆盖率
go test -cover ./...
```

### 常见 Go 问题修复

**1. 未使用的导入**
```go
// ❌ 错误
import (
    "fmt"  // unused
    "log"
)

// ✅ 修复: 删除未使用的导入
import (
    "log"
)
```

**2. 导出函数缺少注释**
```go
// ❌ 错误
func ProcessMessage(msg string) error {
    // ...
}

// ✅ 修复: 添加注释
// ProcessMessage 处理收到的消息并返回错误
func ProcessMessage(msg string) error {
    // ...
}
```

**3. 错误处理**
```go
// ❌ 错误: 忽略错误
result, _ := someFunction()

// ✅ 修复: 正确处理错误
result, err := someFunction()
if err != nil {
    return fmt.Errorf("failed to process: %w", err)
}
```

**4. 资源泄漏**
```go
// ❌ 错误: 未关闭资源
file, err := os.Open("file.txt")
// ... 使用 file

// ✅ 修复: 使用 defer 关闭
file, err := os.Open("file.txt")
if err != nil {
    return err
}
defer file.Close()
// ... 使用 file
```

---

## TypeScript/React 代码质量检查

### 适用模块

- `project/desktop-web` - 前端客户端

### 执行步骤

#### 1. 运行质量检查脚本

```bash
# 在项目根目录执行
./scripts/check-web-quality.sh
```

#### 2. 检查项说明

脚本会依次执行以下检查:

**a. ESLint 检查**
- 检查 JavaScript/TypeScript 代码规范
- 发现潜在的代码问题

**b. TypeScript 类型检查**
- 验证类型系统正确性
- 确保类型安全

**c. Prettier 格式检查**
- 检查代码格式一致性

#### 3. 自动修复

```bash
cd project/desktop-web

# 自动修复 ESLint 问题
npm run lint:fix

# 自动格式化代码
npm run format

# 类型检查(不能自动修复,需要手动修复)
npx tsc --noEmit
```

#### 4. 手动检查

```bash
cd project/desktop-web

# 只检查不修复
npm run lint

# 检查格式
npm run format:check

# 类型检查
npm run type-check
```

### 常见 TypeScript 问题修复

**1. 禁止使用 any 类型**
```typescript
// ❌ 错误
function processData(data: any) {
    return data.value;
}

// ✅ 修复: 使用具体类型
interface DataType {
    value: string;
}

function processData(data: DataType) {
    return data.value;
}
```

**2. 缺少返回类型**
```typescript
// ❌ 错误
function getUserName(userId: string) {
    return users.find(u => u.id === userId)?.name;
}

// ✅ 修复: 添加返回类型
function getUserName(userId: string): string | undefined {
    return users.find(u => u.id === userId)?.name;
}
```

**3. 未使用的变量**
```typescript
// ❌ 错误
const [count, setCount] = useState(0);
const unusedVar = 123;

// ✅ 修复: 删除未使用的变量
const [count, setCount] = useState(0);
```

**4. React Hooks 依赖**
```typescript
// ❌ 错误
useEffect(() => {
    fetchData(userId);
}, []); // userId 缺失

// ✅ 修复: 添加依赖
useEffect(() => {
    fetchData(userId);
}, [userId]);
```

**5. 严格空值检查**
```typescript
// ❌ 错误
function getLength(str: string | null) {
    return str.length; // str 可能为 null
}

// ✅ 修复: 添加空值检查
function getLength(str: string | null) {
    return str?.length ?? 0;
}
```

---

## 质量检查清单

### 提交前检查

在提交代码前,确保完成以下检查:

- [ ] 运行对应的质量检查脚本
- [ ] 所有 ERROR 已修复
- [ ] 所有 WARNING 已修复或有合理理由保留
- [ ] 单元测试全部通过
- [ ] 删除所有 `.bak` 备份文件
- [ ] 没有编译产物被提交
- [ ] 代码格式化完成
- [ ] 没有 `console.log` 或 `fmt.Println` 调试代码

### Go 模块特定检查

- [ ] 所有导出函数都有注释
- [ ] 错误处理正确,包含上下文信息
- [ ] 资源使用了 `defer` 关闭
- [ ] 没有 goroutine 泄漏
- [ ] 使用 `errors.Is()` 和 `errors.As()` 判断错误
- [ ] 共享数据有适当的锁保护

### TypeScript 模块特定检查

- [ ] 没有使用 `any` 类型
- [ ] 所有类型定义正确
- [ ] React Hooks 依赖数组正确
- [ ] 组件使用 `React.memo` 优化(如需要)
- [ ] 列表渲染使用稳定的 `key`
- [ ] 异步操作有错误处理

---

## 持续集成建议

### 在 CI/CD 中集成质量检查

**GitHub Actions 示例**:

```yaml
name: Code Quality Check

on: [push, pull_request]

jobs:
  go-quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.25'
      - name: Run Go Quality Check
        run: ./scripts/check-go-quality.sh

  web-quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '18'
      - name: Run Web Quality Check
        run: ./scripts/check-web-quality.sh
```

---

## 故障排查

### 问题: golangci-lint 未安装

```bash
# macOS
brew install golangci-lint

# Linux
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# 验证安装
golangci-lint --version
```

### 问题: 依赖包缺失

```bash
# Go 模块
go mod download
go mod tidy

# TypeScript 模块
cd project/desktop-web
npm install
```

### 问题: 测试失败

```bash
# 查看详细测试输出
go test -v ./...

# 运行特定测试
go test -run TestFunctionName ./...

# 查看测试覆盖率
go test -cover -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## 总结

代码质量检查是开发流程中的关键环节,遵循以下原则:

1. **每次修改后都运行质量检查**
2. **提交前修复所有 ERROR**
3. **保持代码库整洁,无编译产物和备份文件**
4. **养成良好的代码规范习惯**

质量检查不仅能发现问题,还能帮助团队保持一致的代码风格和最佳实践。
