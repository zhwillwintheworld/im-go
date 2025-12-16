# Access-Go Makefile

.PHONY: build run clean test tidy gen-proto gen-certs help

# 默认目标
all: build

# 构建
build:
	go build -o bin/access ./cmd/access

# 运行
run:
	go run ./cmd/access/main.go

# 清理
clean:
	rm -rf bin/

# 测试
test:
	go test -v ./...

# 整理依赖
tidy:
	go mod tidy

# 生成 FlatBuffers 代码
gen-flatbuf:
	@if command -v flatc >/dev/null 2>&1; then \
		./schema/generate.sh; \
	else \
		echo "flatc 未安装，请执行: brew install flatbuffers"; \
	fi

# 生成 TLS 证书
gen-certs:
	./scripts/gen-certs.sh

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
	@echo "Access-Go 可用命令:"
	@echo "  make build       - 构建二进制文件"
	@echo "  make run         - 运行服务"
	@echo "  make clean       - 清理构建产物"
	@echo "  make test        - 运行测试"
	@echo "  make tidy        - 整理依赖"
	@echo "  make gen-flatbuf - 生成 FlatBuffers 代码"
	@echo "  make gen-certs   - 生成 TLS 证书"
	@echo "  make fmt         - 格式化代码"
	@echo "  make lint        - 代码检查"
