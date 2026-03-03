# ForgeX 用户手册

## 目录
- [安装](#安装)
- [快速开始](#快速开始)
- [命令参考](#命令参考)
- [配置详解](#配置详解)
- [Agent 工作模式](#agent-工作模式)
- [安全与治理](#安全与治理)
- [常见问题](#常见问题)

---

## 安装

### 从源码编译

```bash
# 前置条件: Go 1.25+
git clone https://github.com/awch-D/ForgeX.git
cd ForgeX
make build

# 验证安装
./build/forgex version
```

### 环境变量

ForgeX 支持通过环境变量覆盖配置（前缀 `FORGEX_`）：

```bash
export FORGEX_LLM_API_KEY="sk-your-key"
export FORGEX_LLM_MODEL="claude-sonnet-4-20250514"
```

---

## 快速开始

### 1. 配置 LLM

编辑 `forgex.yaml`，设置你的 API Endpoint 和 Key：

```yaml
llm:
  endpoint: "https://your-api-endpoint"
  api_key: "sk-your-key"
  model: "claude-sonnet-4-20250514"
```

### 2. 运行任务

```bash
# 简单任务
./build/forgex run "用 Python 写一个计算器"

# 指定输出目录
./build/forgex run -o ./my-project "创建一个 REST API 服务"
```

### 3. 交互流程

```
1. ForgeX 分析你的需求
2. 显示需求确认卡（任务类型、技术栈、预计文件数）
3. 按 Y 确认执行
4. Agent 自动编码，终端实时显示进度
5. 进化引擎评分，输出成本汇总
```

---

## 命令参考

| 命令 | 说明 | 示例 |
|------|------|------|
| `forgex run <task>` | 执行编码任务 | `forgex run "写一个 TODO 工具"` |
| `forgex run -o <dir> <task>` | 指定输出目录 | `forgex run -o ./out "写一个 API"` |
| `forgex version` | 打印版本信息 | `forgex version` |

---

## 配置详解

### LLM 配置

```yaml
llm:
  provider: litellm       # 提供商标识
  endpoint: "https://..."  # API 端点
  api_key: "sk-..."       # API Key
  model: "claude-opus-4-20250514"     # 默认模型
  max_tokens: 4096         # 每次请求最大 Token
  temperature: 0.7         # 温度参数
```

### 多模型路由

当你在 `llm.router` 中配置多个模型时，ForgeX 会根据任务复杂度自动选择：

```yaml
llm:
  router:
    strategy: "gear"
    models:
      - name: "deepseek-coder-v2"
        endpoint: "https://..."
        api_key: "sk-..."
        tier: "low"     # 用于 L1/L2 简单任务
      - name: "claude-opus-4-20250514"
        endpoint: "https://..."
        api_key: "sk-..."
        tier: "high"    # 用于 L3+ 复杂任务
```

### 沙箱配置

```yaml
sandbox:
  backend: exec       # 执行后端 (exec)
  timeout_sec: 30     # 命令超时秒数
  memory_mb: 512      # 内存限制 MB
```

### 治理配置

```yaml
governance:
  auto_approve_level: yellow  # 自动批准等级
  max_budget: 10.0            # 最大预算 USD
```

**安全等级说明**：
- 🟢 `green` — 只读操作（读文件、列目录），自动放行
- 🟡 `yellow` — 写操作（创建/修改文件），按配置决定
- 🔴 `red` — 危险操作（删除文件、执行命令），需人工确认
- ⚫ `black` — 禁止操作（修改系统文件），永远拦截

---

## Agent 工作模式

ForgeX 根据任务复杂度自动选择工作模式：

### 单 Agent 模式（L1/L2 简单任务）
```
Coder Agent → 直接编码 → 输出文件
```

### 多 Agent 协作模式（L3+ 复杂任务）
```
Supervisor → 分解子任务
    ├── Coder #1 → 子任务 A
    ├── Coder #2 → 子任务 B
    └── Tester → 验证结果
```

**挡位评估维度**：
1. 关键词复杂度权重
2. 文件数量预估
3. 技术栈受限度
4. LLM 参考评估
5. 待修改文件列表

---

## 安全与治理

### 成本控制
- 实时 Token 计费和 USD 换算
- 超过 `max_budget` 自动熔断
- 终端结尾显示成本汇总

### 审计日志
- 所有 Agent 操作写入审计日志
- 包含时间戳、操作类型、影响范围

### 错误防火墙
- Agent 草稿中的错误逻辑被拦截
- 只有通过验证的代码才能进入事实库

---

## 常见问题

### Q: 支持哪些 LLM？
A: 任何 OpenAI 兼容接口或 Anthropic Claude API。推荐使用 Claude Sonnet 4 或 GPT-4o。

### Q: 需要网络吗？
A: 只有 LLM API 调用需要网络。代码分析、图谱构建、测试评分等全部在本地完成。

### Q: 如何节省 API 费用？
A: 配置多模型路由，让简单任务使用 DeepSeek 等低价模型，仅复杂任务使用高端模型。

### Q: 支持哪些编程语言？
A: ForgeX 可以生成任何语言的代码（取决于 LLM 能力）。代码知识图谱目前仅支持 Go AST 解析。
