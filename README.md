<div align="center">

# ⚡ ForgeX

**Local-First AI Programmer — 本地优先的 AI 自动编程系统**

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Tests](https://img.shields.io/badge/Tests-16%2F16%20PASS-brightgreen)](Makefile)

*用一句话描述你想要什么，ForgeX 自动帮你写代码、测试、审查，一条龙交付。*

</div>

---

## 🎬 快速体验

```bash
# 克隆并编译
git clone https://github.com/awch-D/ForgeX.git
cd ForgeX && make build

# 运行你的第一个任务
./build/forgex run "用 Go 写一个 HTTP 文件服务器"
```

ForgeX 会自动完成：意图分析 → 需求确认 → 挡位评估 → 代码生成 → 质量评分。

---

## ✨ 核心特性

| 特性 | 描述 |
|------|------|
| 🏠 **Local-First** | 纯 Go 单二进制，零外部服务依赖，数据不出本机 |
| 🤖 **多 Agent 协作** | Supervisor 分解任务 → Coder 并发编码 → Tester 验证 → Reviewer 审查 |
| ⚙️ **智能挡位** | 5 维评估自动识别任务复杂度，简单任务 1 个 Agent，复杂任务多 Agent 并发 |
| 🔀 **LLM 路由** | 多模型配置 + 3 种策略（gear/cheapest/fallback），简单任务用便宜模型 |
| 🧠 **代码知识图谱** | AST 解析项目结构，Agent 修改代码前自动感知依赖影响半径 |
| 📊 **进化引擎** | 自动编译+测试评分，低于阈值提示优化 |
| 🛡️ **四级安全带** | 绿/黄/红/黑操作分级拦截，危险操作需人工确认 |
| 💰 **成本控制** | 实时 Token 计费，预算超限自动熔断 |

---

## 🏗 架构总览

```
forgex run "你的任务"
     │
     ▼
┌─ 意图感知层 ──────────────────────────────┐
│  多轮追问 → 需求解析 → 可视化确认卡       │
└────────────────────┬──────────────────────┘
                     ▼
┌─ 控制面 ──────────────────────────────────┐
│  挡位评估 → LLM 路由 → Agent 协程池       │
│  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────────┐ │
│  │ SUP  │→│ DEV  │→│ TST  │→│ REVIEWER │ │
│  └──────┘ └──────┘ └──────┘ └──────────┘ │
└────────────────────┬──────────────────────┘
                     ▼
┌─ 认知层 ──────────────────────────────────┐
│  事实库(SQLite) │ 向量检索 │ 代码图谱(AST) │
│  草稿区 │ 错误防火墙 │ 进化引擎评分       │
└────────────────────┬──────────────────────┘
                     ▼
┌─ 治理层 ──────────────────────────────────┐
│  安全带 │ 预算控制 │ 审计日志 │ 终端监控   │
└───────────────────────────────────────────┘
```

---

## 📦 模块一览

| 模块 | 职责 | 测试状态 |
|------|------|----------|
| `forgex-cli` | CLI 入口 + 终端事件渲染器 | ✅ |
| `forgex-core` | 配置 / 日志 / 错误体系 | ✅ |
| `forgex-intent` | 意图解析 / 多轮追问 / AST 考古 | ✅ |
| `forgex-gear` | 5 维智能挡位评估器 | ✅ |
| `forgex-llm` | LLM Provider 接口 + 多模型路由 | ✅ |
| `forgex-agent` | Supervisor / Coder / Tester / Reviewer + EventBus + 协程池 | ✅ |
| `forgex-cognition` | 事实库 / 草稿区 / 向量检索 / 防火墙 / 知识图谱 | ✅ |
| `forgex-governance` | 安全带 / 预算控制 / 审计日志 | ✅ |
| `forgex-evolution` | 编译+测试自动评分引擎 | ✅ |
| `forgex-mcp` | MCP 工具层（文件读写 / 终端执行） | ✅ |
| `forgex-sandbox` | 沙箱执行器接口 | ✅ |

---

## ⚙️ 配置参考

编辑项目根目录下的 `forgex.yaml`：

```yaml
log_level: info
dev_mode: true

llm:
  provider: litellm
  endpoint: "https://your-api-endpoint"
  api_key: "sk-your-key"
  model: "claude-sonnet-4-20250514"
  max_tokens: 4096
  temperature: 0.7

  # 可选：多模型智能路由
  router:
    strategy: "gear"  # gear | cheapest | fallback
    models:
      - name: "deepseek-coder-v2"
        endpoint: "https://your-api-endpoint"
        api_key: "sk-xxx"
        tier: "low"    # 简单任务用
      - name: "claude-opus-4-20250514"
        endpoint: "https://your-api-endpoint"
        api_key: "sk-xxx"
        tier: "high"   # 复杂任务用

sandbox:
  backend: exec
  timeout_sec: 30
  memory_mb: 512

governance:
  auto_approve_level: yellow  # green | yellow
  max_budget: 10.0            # 最大预算 (USD)
```

### 路由策略说明

| 策略 | 行为 |
|------|------|
| `gear` | L1/L2 简单任务用 low-tier，L3+ 用 high-tier（默认） |
| `cheapest` | 永远选最便宜的模型 |
| `fallback` | 先尝试便宜模型，失败自动切换高端模型 |

---

## 🧪 测试

```bash
make test        # 运行全部单元测试
make test-e2e    # 运行端到端集成测试
make test-all    # 全量测试
```

---

## 🛠 开发

```bash
# 前置条件
go 1.25+

# 编译
make build

# 整理依赖
make tidy

# 清理构建产物
make clean
```

### 项目结构

```
ForgeX/
├── forgex-cli/          # CLI 入口 + 终端渲染器
├── forgex-core/         # 配置 / 日志 / 错误
├── forgex-intent/       # 意图解析 / AST 考古
├── forgex-gear/         # 挡位评估器
├── forgex-llm/          # LLM 适配 + 路由
├── forgex-agent/        # 多 Agent 体系
├── forgex-cognition/    # 认知存储层
├── forgex-governance/   # 治理层
├── forgex-evolution/    # 进化引擎
├── forgex-mcp/          # MCP 工具
├── forgex-sandbox/      # 沙箱接口
├── test/e2e/            # 集成测试
├── forgex.yaml          # 配置文件
├── go.work              # Go workspace
└── Makefile             # 构建脚本
```

---

## 📄 License

[MIT](LICENSE) © ForgeX Contributors
