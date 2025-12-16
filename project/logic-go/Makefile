# Logic-Go Makefile

.PHONY: build run clean test tidy migrate help

# 默认目标
all: build

# 构建
build:
	go build -o bin/logic ./cmd/logic

# 运行
run:
	go run ./cmd/logic/main.go

# 清理
clean:
	rm -rf bin/

# 测试
test:
	go test -v ./...

# 整理依赖
tidy:
	go mod tidy

# 数据库迁移
migrate:
	@echo "执行数据库迁移..."
	@if command -v psql >/dev/null 2>&1; then \
		psql -h localhost -U postgres -d im -f migrations/001_init.sql; \
	else \
		echo "psql 未安装，请手动执行迁移"; \
	fi

# 格式化代码
fmt:
	go fmt ./...

# 代码检查
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint 未安装"; \
	fi

# 帮助
help:
	@echo "Logic-Go 可用命令:"
	@echo "  make build   - 构建二进制文件"
	@echo "  make run     - 运行服务"
	@echo "  make clean   - 清理构建产物"
	@echo "  make test    - 运行测试"
	@echo "  make tidy    - 整理依赖"
	@echo "  make migrate - 执行数据库迁移"
	@echo "  make fmt     - 格式化代码"
	@echo "  make lint    - 代码检查"
