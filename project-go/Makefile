.PHONY: build run test lint clean docker-build docker-run fmt vet

APP_NAME := ai-agent-go
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS  := -ldflags "-s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# 构建二进制
build:
	@echo ">>> 构建 $(APP_NAME)..."
	CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(APP_NAME) ./cmd/server

# 本地运行
run:
	@echo ">>> 启动开发服务器..."
	go run ./cmd/server

# 运行测试
test:
	@echo ">>> 运行测试..."
	go test -v -race -coverprofile=coverage.out ./...

# 代码格式化
fmt:
	@echo ">>> 格式化代码..."
	gofmt -s -w .
	goimports -w .

# 静态检查
vet:
	@echo ">>> 静态分析..."
	go vet ./...

# lint 检查
lint:
	@echo ">>> Lint 检查..."
	golangci-lint run ./...

# 清理产物
clean:
	@echo ">>> 清理构建产物..."
	rm -rf bin/ coverage.out

# 构建 Docker 镜像
docker-build:
	@echo ">>> 构建 Docker 镜像..."
	docker build -t $(APP_NAME):$(VERSION) .

# 运行 Docker 容器
docker-run:
	@echo ">>> 运行 Docker 容器..."
	docker run -d --name $(APP_NAME) \
		-p 8080:8080 \
		-e APP_ENV=production \
		$(APP_NAME):$(VERSION)

# 生成模拟依赖（用于测试）
mock:
	@echo ">>> 生成 Mock..."
	mockgen -source=internal/llm/client.go -destination=internal/llm/mock_client.go -package=llm
	mockgen -source=internal/tool/base.go -destination=internal/tool/mock_tool.go -package=tool
