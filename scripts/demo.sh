#!/usr/bin/env bash
# ForgeX Demo Script
# 一键清理、构建并运行 Demo
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_DIR"

# Colors
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}      ForgeX Demo Script${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Step 1: Clean
echo -e "${YELLOW}[1/4]${NC} 清理旧构建和输出..."
rm -rf build/ forgex-output/
echo -e "${GREEN}  ✅ 清理完成${NC}"

# Step 2: Build
echo -e "${YELLOW}[2/4]${NC} 构建 ForgeX..."
go build -o build/forgex ./forgex-cli
echo -e "${GREEN}  ✅ 构建完成: build/forgex${NC}"

# Step 3: Version check
echo -e "${YELLOW}[3/4]${NC} 版本检查..."
./build/forgex version
echo ""

# Step 4: Run demo task
TASK="${1:-用Go写一个简单的TODO命令行工具，支持添加、列出和删除任务}"
echo -e "${YELLOW}[4/4]${NC} 运行 Demo 任务:"
echo -e "  ${CYAN}$TASK${NC}"
echo ""
echo -e "${YELLOW}━━━━━━━ Demo 开始 ━━━━━━━${NC}"
echo ""

./build/forgex run "$TASK"

echo ""
echo -e "${GREEN}━━━━━━━ Demo 完成 ━━━━━━━${NC}"

# Show output directory
if [ -d "forgex-output" ]; then
    echo ""
    echo -e "${CYAN}📂 生成的代码在:${NC}"
    find forgex-output -type f | head -20
fi
