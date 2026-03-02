# ForgeX ⚡

> **Local-First AI Programmer** — 融合双重隔离沙箱与流体协程架构的终极 AI 程序员

[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Phase_0_Alpha-orange)](https://github.com/awch-D/ForgeX)

---

## ✨ 项目简介

**ForgeX** 是一款旨在"完全接管开发工作流"的前沿 AI 程序员系统。它摒弃了传统的臃肿微服务，采用 Go 构建极速单机控制面，并以 Rust/Wasm 双重防爆沙箱为执行底座。

### 核心特性

- ⚡ **流体 Agent 协程池** — 基于 Go Goroutine，毫秒级调度数十个 AI 角色并行探索最优解
- ⚙️ **智能挡位器 (L1~L4)** — 根据任务复杂度自动变速，节约 Token 开销
- 🛡️ **Dual-Metered 安全沙箱** — CPU 燃料计量 + 内存硬隔离 + 防 SSRF 网络拦截
- 🧠 **分级认知与错误防火墙** — 隔离 Agent 错误推断，仅验证后事实可共享
- 🔍 **存量代码考古** — 理解、适配并融入您的现有代码库
- 📺 **玄武岩 Dashboard** — 实时监控 Agent 活动、成本和进度

---

## 🚀 快速开始

### 安装

```bash
# 克隆仓库
git clone https://github.com/awch-D/ForgeX.git
cd ForgeX

# 构建
make build

# 验证
./build/forgex version
```

### 使用

```bash
# 运行一个编码任务 (Phase 1+ 实现)
./build/forgex run "帮我写一个带用户登录的 Go HTTP 服务"
```

---

## 🏗️ 项目结构

```
ForgeX/
├── forgex-cli/         ⚡ CLI 入口 (Cobra)
├── forgex-core/        📦 公共基础库 (日志/错误/配置/类型)
├── forgex-intent/      🎯 意图对齐引擎
├── forgex-gear/        ⚙️ 智能挡位器
├── forgex-agent/       🌊 流体 Agent 引擎
├── forgex-cognition/   🧠 集体认知系统
├── forgex-llm/         🧠 LLM 多模型路由
├── forgex-mcp/         🔧 MCP 工具层
├── forgex-governance/  🛡️ 守护与治理
├── forgex-evolution/   🧬 引导式进化引擎
├── forgex-sandbox/     🛡️ Rust 执行沙箱 (Phase 4)
└── forgex-dashboard/   📺 玄武岩 Dashboard (Phase 5)
```

---

## 📋 开发路线图

| 阶段 | 周期 | 核心产出 | 状态 |
|------|------|---------|------|
| **Phase 0** 骨架基建 | 2 周 | CLI + 日志 + 配置 + CI | 🟢 进行中 |
| **Phase 1** LLM + 意图 | 3 周 | 多模型路由 + 需求对话 | ⏳ 待启动 |
| **Phase 2** 单 Agent + MCP | 3 周 | 能自动写代码的 MVP | ⏳ 待启动 |
| **Phase 3** 多 Agent + 认知 | 6 周 | 并行协作 + 记忆隔离 | ⏳ 待启动 |
| **Phase 4** 安全沙箱 + 治理 | 4 周 | Wasm 双重防爆 + 审计 | ⏳ 待启动 |
| **Phase 5** Dashboard + 进化 | 8 周 | 完整产品化 | ⏳ 待启动 |

---

## 🛠️ 技术栈

| 层级 | 技术 |
|------|------|
| 控制面 | Go (Goroutines) |
| 执行沙箱 | Rust + Wasmtime (Phase 4) |
| LLM 路由 | LiteLLM 代理 → 自研路由 |
| 事实存储 | SQLite + FTS5 |
| 前端 | Vite + React + TypeScript (Phase 5) |

---

## 📄 许可证

[Apache License 2.0](LICENSE)
