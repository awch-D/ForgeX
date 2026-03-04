# ForgeX Architecture Documentation

> **Last Updated**: 2026-03-04  
> **Version**: v1.1 (Phase 3 — Production Ready)

---

## 1. Overview

ForgeX is a **Local-First, Multi-Agent AI Programmer** built entirely in Go. It accepts a natural language task description, decomposes it, generates code, tests, and reviews — all autonomously within the developer's local machine.

**Design Principles:**
- 🏠 **Local-First**: Single Go binary, zero external services, data never leaves the machine
- ⚡ **Goroutine-native concurrency**: Agent pool expands dynamically from 1 to 50+ coroutines
- 🛡️ **Defense-in-depth**: Four-layer safety belt + process group sandbox + error firewall
- 🧩 **Modular Monorepo**: Each concern is an independent Go module under `go.work`

---

## 2. System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                  👤 User Input (forgex run "task")               │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────── 🎯 Intent Layer ─────────────────────────────────┐
│  Parser → Multi-round Clarifier → Visual Confirmation Card       │
│  + Code Archaeology (Git history hot spots + co-modification)    │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────── ⚙️ Control Plane ────────────────────────────────┐
│                                                                   │
│  ┌──────────────┐    ┌──────────────────────────────────────┐    │
│  │  Gear Engine  │──▶│        Agent Goroutine Pool           │    │
│  │  (L1 – L4)   │    │  ┌──────────┐  ┌──────┐  ┌───────┐  │    │
│  │  5-dim score  │    │  │Supervisor│  │Coder │  │Tester │  │    │
│  └──────────────┘    │  └────┬─────┘  └──┬───┘  └───┬───┘  │    │
│                       │       │           │          │       │    │
│  ┌──────────────┐    │  ┌────▼───────────▼──────────▼────┐  │    │
│  │  LLM Router   │◀──│  │          EventBus              │  │    │
│  │gear/cheapest/ │    │  └───────────────────────────────┘  │    │
│  │   fallback    │    └──────────────────────────────────────┘    │
│  └──────────────┘                                                  │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────── 🧠 Cognition Plane ──────────────────────────────┐
│                                                                   │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────────────┐  │
│  │  Fact Store  │  │  Draft Store  │  │    Knowledge Graph      │  │
│  │ (SQLite FTS5)│  │(in-memory,   │  │ (AST: Go/TS/Python +   │  │
│  │              │  │ per-agent    │  │  Git co-mod edges)      │  │
│  │              │  │ isolation)   │  │                         │  │
│  └─────────────┘  └──────────────┘  └────────────────────────┘  │
│                                                                   │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────────────┐  │
│  │   Inference   │  │Error Firewall │  │   Evolution Engine     │  │
│  │ (vector/cosine│  │(draft→fact   │  │(compile+test score,    │  │
│  │  similarity) │  │ validation)  │  │ LLM retry prompt)      │  │
│  └─────────────┘  └──────────────┘  └────────────────────────┘  │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────── 🛡️ Governance + Execution Layer ─────────────────┐
│                                                                   │
│  Safety Belt (🟢green/🟡yellow/🔴red/⚫black)                   │
│  Budget Controller (real-time USD cap + circuit breaker)         │
│  Audit Logger                                                     │
│                                                                   │
│  ┌───────────────────────────────────────────────────────────┐   │
│  │                  Sandbox Executor                          │   │
│  │  - Process Group Isolation (Setpgid + SIGKILL tree)       │   │
│  │  - Memory Limit (ulimit -v injection)                     │   │
│  │  - Configurable timeout per operation                     │   │
│  └───────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. Module Reference

| Module | Path | Responsibility |
|--------|------|----------------|
| `forgex-cli` | `forgex-cli/` | CLI entry + terminal event renderer (real-time colored output) |
| `forgex-core` | `forgex-core/` | Config (Viper), Logger (Zap), Errors, Types |
| `forgex-intent` | `forgex-intent/` | Intent parser, Clarifier, Confirmation card, Code Archaeology |
| `forgex-gear` | `forgex-gear/` | 5-dimension complexity scorer + L1~L4 dispatch |
| `forgex-llm` | `forgex-llm/` | Provider interface, LiteLLM adapter, Cost ledger, LLM Router |
| `forgex-agent` | `forgex-agent/` | EventBus, Goroutine Pool, Supervisor / Coder / Tester / Reviewer |
| `forgex-cognition` | `forgex-cognition/` | Fact store, Draft store, Vector inference, Error firewall, AST graph |
| `forgex-governance` | `forgex-governance/` | Safety belt, Budget controller, Audit log |
| `forgex-evolution` | `forgex-evolution/` | Build+test quality scorer, retry prompt generator |
| `forgex-mcp` | `forgex-mcp/` | Tool registry (file R/W, shell exec, list dir) |
| `forgex-sandbox` | `forgex-sandbox/` | Executor interface + local exec backend (Unix process group) |

---

## 4. Key Data Flows

### 4.1 Task Execution Flow

```
forgex run "task"
   │
   ├─ Intent: parse → clarify → confirm card
   │
   ├─ Archaeology: git log analysis → hot files → co-mod edges ─► graph.Store
   │
   ├─ AST Scan: .go / .ts / .py → struct/class/func nodes ──────► graph.Store
   │
   ├─ Gear: score complexity (L1–L4)
   │    └─ L1/L2: single Coder Agent
   │    └─ L3/L4: Supervisor → Coder + Tester in parallel
   │
   ├─ LLM Router: route to high/low tier model based on gear level
   │
   ├─ Agent Loop: think → tool_call → result → iterate
   │    └─ Tools: write_file / run_command / read_file / list_dir
   │    └─ Safety belt check before each tool execution
   │    └─ Audit logger records every invocation
   │
   ├─ Reviewer Agent: code review → score + feedback → iterate if needed
   │
   ├─ Evolution Engine: compile score + test score → retry if below threshold
   │    └─ Budget guard check before each retry
   │    └─ GOWORK=off for isolated Go builds
   │
   └─ Terminal Output: real-time EventBus → colored role-labeled display
        └─ Cost summary + remaining budget + audit stats
```

### 4.2 Knowledge Graph Construction

The graph is populated **before** agents start working:

```
git log ──► GitAnalyzer ──► file:X commit_count=N
                        ──► CoModifiedWith edge (A ↔ B, weight=co-occurrence)

AST scan ──► Builder ──► file:X
                     ──► func:X.Method → DefinedIn → file:X
                     ──► func:X.Method → Calls → func:Y
                     ──► type:Struct → DefinedIn → file:X
```

`graph.Store.BuildSummary()` serializes this graph into LLM-readable markdown, injected into the Coder's system prompt.

### 4.3 Agent Isolation

Each agent has its own `DraftStore` partition (keyed by `AgentRole`). Drafts are promoted to `FactStore` only after passing the **Error Firewall** (heuristic + optional LLM validation).

---

## 5. LLM Router

The router (`forgex-llm/router`) sits in front of multiple downstream `litellm.Client` instances.

| Strategy | Behavior |
|----------|----------|
| `gear` | Low-tier models for L1/L2 tasks, high-tier for L3+ (default) |
| `cheapest` | Always route to the cheapest configured model |
| `fallback` | Try cheapest first; fall back to high-tier on failure |

Configuration (in `forgex.yaml`):

```yaml
llm:
  router:
    strategy: "gear"
    models:
      - name: "deepseek-coder-v2"
        tier: "low"
      - name: "claude-opus-4-20250514"
        tier: "high"
```

---

## 6. Sandbox Security Model

The local executor (`forgex-sandbox/local`) provides two independent layers of protection:

| Layer | Mechanism | Protection |
|-------|-----------|------------|
| **Memory cap** | `ulimit -v <KB>` injected before command | Prevents OOM / memory bomb |
| **Process group kill** | `SysProcAttr{Setpgid:true}` + `SIGKILL(-pgid)` | Kills entire process tree on timeout, no orphan escape |
| **Timeout** | `context.WithTimeout` + goroutine `select` | Hard deadline on any command |

---

## 7. Multi-Language AST Support

The knowledge graph builder supports three languages. All parsers produce the same node/edge schema for graph homogeneity:

| Language | File Types | Parser | Accuracy |
|----------|-----------|--------|----------|
| Go | `.go` | `go/ast` stdlib (full parse) | High |
| TypeScript / JavaScript | `.ts`, `.js`, `.tsx`, `.jsx` | Heuristic regex (Classs, Interface, Function, Arrow fn) | ~90% |
| Python | `.py` | Heuristic regex (class, def + indentation context) | ~90% |

Files larger than **1 MB** or matching test patterns (`.test.ts`, `_test.go`) are skipped.

---

## 8. Configuration Reference

See [`docs/user-guide.md`](user-guide.md) for full configuration details.
