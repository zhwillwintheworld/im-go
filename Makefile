# IM-Go 项目根目录 Makefile

.PHONY: all build run clean test env-up env-down help

# 默认目标
all: build

# 构建所有服务
build:
	@echo "构建 Access-Go..."
	cd project/access-go && go build -o bin/access ./cmd/access
	@echo "构建 Logic-Go..."
	cd project/logic-go && go build -o bin/logic ./cmd/logic
	@echo "构建完成!"

# 运行服务 (需要先启动依赖)
run-access:
	cd project/access-go && go run ./cmd/access/main.go

run-logic:
	cd project/logic-go && go run ./cmd/logic/main.go

# 清理
clean:
	rm -rf project/access-go/bin/
	rm -rf project/logic-go/bin/

# 测试
test:
	cd project/access-go && go test -v ./...
	cd project/logic-go && go test -v ./...

# 整理依赖
tidy:
	cd project/access-go && go mod tidy
	cd project/logic-go && go mod tidy

# 使用 Podman 启动依赖环境
env-up:
	./env/start-all.sh

# 停止依赖环境
env-down:
	./env/stop-all.sh

# 使用 Docker Compose 启动依赖
docker-up:
	docker-compose up -d

# 停止 Docker Compose
docker-down:
	docker-compose down

# 格式化代码
fmt:
	cd project/access-go && go fmt ./...
	cd project/logic-go && go fmt ./...

# 帮助
help:
	@echo "IM-Go 项目命令:"
	@echo ""
	@echo "构建:"
	@echo "  make build       - 构建所有服务"
	@echo "  make clean       - 清理构建产物"
	@echo "  make tidy        - 整理所有依赖"
	@echo ""
	@echo "运行:"
	@echo "  make run-access  - 运行 Access 服务"
	@echo "  make run-logic   - 运行 Logic 服务"
	@echo ""
	@echo "测试:"
	@echo "  make test        - 运行所有测试"
	@echo "  make fmt         - 格式化代码"
	@echo ""
	@echo "环境 (Podman):"
	@echo "  make env-up      - 启动依赖 (NATS/Redis/PostgreSQL)"
	@echo "  make env-down    - 停止依赖"
	@echo ""
	@echo "环境 (Docker Compose):"
	@echo "  make docker-up   - 启动依赖"
	@echo "  make docker-down - 停止依赖"
