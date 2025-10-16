# CNET Agent Makefile

.PHONY: build run clean test deps install

# 变量定义
BINARY_NAME=cnet-agent
BINARY_DIR=bin
GO=go
GOFLAGS=-v

# 默认目标
all: build

# 编译
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BINARY_DIR)
	$(GO) build $(GOFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME) main.go
	@echo "Build complete: $(BINARY_DIR)/$(BINARY_NAME)"
	@echo "Building inference servers..."
	$(GO) build $(GOFLAGS) -o $(BINARY_DIR)/cnet-inference-yolo cmd/inference/yolo/main.go
	@echo "Build complete: $(BINARY_DIR)/cnet-inference-yolo"
	$(GO) build $(GOFLAGS) -o $(BINARY_DIR)/cnet-inference-opencv cmd/inference/opencv/main.go
	@echo "Build complete: $(BINARY_DIR)/cnet-inference-opencv"
	$(GO) build $(GOFLAGS) -o $(BINARY_DIR)/cnet-gateway-data cmd/gateway/main.go
	@echo "Build complete: $(BINARY_DIR)/cnet-gateway-data"

# 运行
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_DIR)/$(BINARY_NAME) -config config.yaml

# 清理
clean:
	@echo "Cleaning..."
	rm -rf $(BINARY_DIR)/
	rm -f coverage.out
	rm -f *.log
	@echo "Clean complete"

# 安装依赖
deps:
	@echo "Installing dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "Dependencies installed"

# 测试
test:
	@echo "Running tests..."
	$(GO) test -v ./...

# 测试覆盖率
coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out

# 运行示例
demo: build
	@echo "Running demo..."
	./test_agent.sh

# 格式化代码
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

# 代码检查
lint:
	@echo "Running linter..."
	golangci-lint run

# 帮助
help:
	@echo "CNET Agent Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make build     - Build the application"
	@echo "  make run       - Build and run the application"
	@echo "  make clean     - Remove build artifacts"
	@echo "  make deps      - Install dependencies"
	@echo "  make test      - Run tests"
	@echo "  make coverage  - Run tests with coverage"
	@echo "  make demo      - Run demo script"
	@echo "  make fmt       - Format code"
	@echo "  make lint      - Run linter"
	@echo "  make help      - Show this help message"
