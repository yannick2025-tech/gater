.PHONY: build run test clean tidy lint

# 项目名称
APP_NAME := nts-gater

# 构建输出目录
BUILD_DIR := build

# Go 参数
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOFMT := gofmt
GOLINT := golangci-lint

# 主入口
MAIN_PKG := ./cmd/server

all: build

## build: 编译项目
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PKG)
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"

## run: 直接运行
run:
	$(GOCMD) run $(MAIN_PKG)

## test: 运行所有单元测试
test:
	$(GOTEST) -v -count=1 ./...

## test-coverage: 运行单元测试并生成覆盖率报告
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## tidy: 整理依赖
tidy:
	$(GOCMD) mod tidy

## fmt: 格式化代码
fmt:
	$(GOFMT) -w .

## lint: 代码检查
lint:
	$(GOLINT) run ./...

## clean: 清理构建产物
clean:
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

## help: 显示帮助
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //'
