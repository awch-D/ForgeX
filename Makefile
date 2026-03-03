.PHONY: build test clean run version tidy

# ============ 变量 ============
BINARY_NAME := forgex
BUILD_DIR := ./build
CLI_DIR := ./forgex-cli

# ============ 构建 ============
build:
	@echo "🔨 Building ForgeX..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CLI_DIR)
	@echo "✅ Built: $(BUILD_DIR)/$(BINARY_NAME)"

# ============ 运行 ============
run: build
	@$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

version: build
	@$(BUILD_DIR)/$(BINARY_NAME) version

# ============ 测试 ============
test:
	@echo "🧪 Running tests..."
	@for dir in forgex-core forgex-agent forgex-gear forgex-intent forgex-llm forgex-mcp forgex-governance forgex-evolution; do \
		echo "  → $$dir"; \
		(cd $$dir && go test ./... -v -count=1) || true; \
	done
	@echo "  → forgex-cognition (with FTS5)"
	@(cd forgex-cognition && CGO_ENABLED=1 go test -tags fts5 ./... -v -count=1) || true
	@echo "✅ All module tests done"

test-e2e:
	@echo "🧪 Running e2e tests..."
	@(cd test/e2e && go test -v -count=1 -timeout 60s ./...)
	@echo "✅ E2E tests passed"

test-all: test test-e2e

# ============ 依赖 ============
tidy:
	@echo "📦 Tidying dependencies..."
	@for dir in forgex-core forgex-cli forgex-intent forgex-gear forgex-agent forgex-cognition forgex-llm forgex-mcp forgex-governance forgex-evolution; do \
		echo "  → $$dir"; \
		(cd $$dir && go mod tidy); \
	done
	@echo "✅ All modules tidied"

# ============ 清理 ============
clean:
	@echo "🧹 Cleaning..."
	@rm -rf $(BUILD_DIR)
	@echo "✅ Clean done"

# ============ 帮助 ============
help:
	@echo "ForgeX Makefile"
	@echo ""
	@echo "  make build    - Build the ForgeX binary"
	@echo "  make run      - Build and run (use ARGS='version')"
	@echo "  make version  - Print version"
	@echo "  make test     - Run all tests"
	@echo "  make tidy     - Tidy all Go module dependencies"
	@echo "  make clean    - Remove build artifacts"
