package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/awch-D/ForgeX/forgex-agent/coder"
	"github.com/awch-D/ForgeX/forgex-agent/pool"
	"github.com/awch-D/ForgeX/forgex-agent/protocol"
	"github.com/awch-D/ForgeX/forgex-agent/reviewer"
	"github.com/awch-D/ForgeX/forgex-agent/supervisor"
	"github.com/awch-D/ForgeX/forgex-agent/tester"
	"github.com/awch-D/ForgeX/forgex-cli/display"
	"github.com/awch-D/ForgeX/forgex-cognition/graph"
	"github.com/awch-D/ForgeX/forgex-core/config"
	"github.com/awch-D/ForgeX/forgex-core/logger"
	evolution "github.com/awch-D/ForgeX/forgex-evolution"
	gear "github.com/awch-D/ForgeX/forgex-gear"
	"github.com/awch-D/ForgeX/forgex-governance/audit"
	"github.com/awch-D/ForgeX/forgex-governance/budget"
	"github.com/awch-D/ForgeX/forgex-governance/safety"
	"github.com/awch-D/ForgeX/forgex-intent/archaeology"
	"github.com/awch-D/ForgeX/forgex-intent/clarifier"
	"github.com/awch-D/ForgeX/forgex-intent/confirmation"
	"github.com/awch-D/ForgeX/forgex-llm/cost"
	"github.com/awch-D/ForgeX/forgex-llm/litellm"
	"github.com/awch-D/ForgeX/forgex-llm/provider"
	"github.com/awch-D/ForgeX/forgex-llm/router"
	"github.com/awch-D/ForgeX/forgex-mcp/tools"
	localsandbox "github.com/awch-D/ForgeX/forgex-sandbox/local"
)

// version can be overridden at build time via ldflags:
//
//	go build -ldflags "-X commands.version=$(git describe --tags)"
var version = "0.3.0-alpha"

func showBanner() {
	pterm.DefaultBigText.WithLetters(
		pterm.NewLettersFromStringWithStyle("Forge", pterm.NewStyle(pterm.FgCyan)),
		pterm.NewLettersFromStringWithStyle("X", pterm.NewStyle(pterm.FgLightRed)),
	).Render()
	pterm.DefaultCenter.Println(pterm.LightCyan("⚡ Local-First AI Programmer • v" + version))
	fmt.Println()
}

var rootCmd = &cobra.Command{
	Use:   "forgex",
	Short: "ForgeX — Local-First AI Programmer",
	Long: `ForgeX is an advanced, local-first AI programmer featuring
elastic agent orchestration, dual-metered sandboxing,
and autonomous code evolution.

Usage:
  forgex run "帮我写一个带登录的 Web 服务"
  forgex version`,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(runCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of ForgeX",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("⚡ ForgeX %s\n", version)
		fmt.Println("  Local-First AI Programmer")
		fmt.Println("  https://github.com/awch-D/ForgeX")
	},
}

// Flags
var (
	outputDir    string
	dryRun       bool
	gearOverride int
)

func init() {
	runCmd.Flags().StringVarP(&outputDir, "output", "o", "", "Output directory (default: ./forgex-output)")
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Only parse intent and evaluate gear level, don't execute agents")
	runCmd.Flags().IntVar(&gearOverride, "gear-level", 0, "Manually override gear level (1-4), bypassing automatic evaluation")
}

var runCmd = &cobra.Command{
	Use:   "run [task description]",
	Short: "Run a coding task",
	Long:  "Describe what you want ForgeX to build, and it will autonomously create the code.",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		initialPrompt := args[0]

		showBanner()

		cfg, err := config.Load()
		if err != nil {
			logger.L().Fatalw("Failed to load config", "error", err)
		}

		pterm.Info.Printf("🤖 模型: %s\n", pterm.LightCyan(cfg.LLM.Model))
		pterm.Info.Printf("📝 任务: %s\n", pterm.LightYellow(initialPrompt))
		fmt.Println()

		logger.L().Infow("ForgeX initialized", "LLM", cfg.LLM.Provider, "Model", cfg.LLM.Model, "Task", initialPrompt)

		// Initialize LLM Provider (with optional Router)
		var llmProvider provider.Provider
		if cfg.LLM.Router != nil && len(cfg.LLM.Router.Models) > 0 {
			llmProvider = router.New(cfg.LLM.Router, &cfg.LLM)
			pterm.Info.Println("🔀 LLM 路由器已启用")
		} else {
			llmProvider = litellm.NewClient(&cfg.LLM)
		}

		// Phase 1: Intent Clarification
		ctx := cmd.Context()
		analysis, _, err := clarifier.Run(ctx, llmProvider, initialPrompt)
		if err != nil {
			logger.L().Fatalw("Clarifier failed", "error", err)
		}

		confirmed := confirmation.RenderCard(analysis)
		if !confirmed {
			return
		}

		_, intentCost := cost.Global().Summary()
		logger.L().Infof("🧠 意图分析花费: 约 $%.4f", intentCost)

		// Determine output directory
		workDir := outputDir
		if workDir == "" {
			workDir = filepath.Join(".", "forgex-output")
		}
		absWorkDir, _ := filepath.Abs(workDir)
		os.MkdirAll(absWorkDir, 0755)

		// Create tool registry
		registry := tools.NewRegistry(absWorkDir)

		// ===== Governance Setup =====

		// 1. Audit Logger: record all tool invocations to SQLite
		auditDir := filepath.Join(absWorkDir, ".forgex")
		os.MkdirAll(auditDir, 0755)
		auditLogger, err := audit.NewLogger(filepath.Join(auditDir, "audit.db"))
		if err != nil {
			logger.L().Warnw("Failed to initialize audit logger, continuing without audit", "error", err)
		} else {
			defer auditLogger.Close()
			registry.SetAuditLogger(auditLogger)
			pterm.Info.Println("📋 审计日志已启用")
		}

		// 2. Safety auto-approve level from config
		approveLevel := safety.ParseLevel(cfg.Governance.AutoApproveLevel)
		registry.SetAutoApproveLevel(approveLevel)

		// 3. Sandbox executor: process group isolation + memory limits
		executor := localsandbox.New(cfg.Sandbox.TimeoutSec, cfg.Sandbox.MemoryMB)
		registry.SetExecutor(executor)

		// 4. Budget guard: enforce cost ceiling
		guard := budget.NewGuard(cfg.Governance.MaxBudget, cost.Global())
		pterm.Info.Printf("💰 预算上限: $%.2f\n", cfg.Governance.MaxBudget)

		// Gear Evaluation: decide execution strategy
		fmt.Println()
		level := gear.Evaluate(analysis)

		// Allow manual gear level override via CLI flag
		if gearOverride >= 1 && gearOverride <= 4 {
			level = gear.Level(gearOverride)
			pterm.Info.Printf("⚙️ 任务评级: %s (手动覆盖)\n", level.String())
		} else {
			pterm.Info.Printf("⚙️ 任务评级: %s\n", level.String())
		}

		// Dry-run mode: output analysis results and exit without executing agents
		if dryRun {
			pterm.Success.Println("🔍 Dry-run 完成，已跳过 Agent 执行")
			display.PrintCostSummary()
			return
		}

		// Wrap provider with gear level so Router can apply gear-aware model selection.
		gearProvider := provider.WithGear(llmProvider, int(level))

		// Setup event bus
		eventBus := protocol.NewEventBus()
		defer eventBus.Close()

		// Start terminal live monitor (replaces Web Dashboard)
		display.StartLiveMonitor(eventBus)

		// Build architecture graph (AST + Git History)
		graphStore := graph.NewStore()
		pterm.Info.Println("🏗 正在扫描代码架构图谱与 Git 考古历史...")
		archBuilder := archaeology.NewBuilder(graphStore)
		_ = archBuilder.Build(absWorkDir)

		gitAnalyzer := archaeology.NewGitAnalyzer(graphStore)
		// Analyze recent 100 commits to find hot spots and implicit dependencies
		_ = gitAnalyzer.Analyze(ctx, absWorkDir, 100)

		// Budget check before agent execution
		if err := guard.Check(); err != nil {
			pterm.Error.Printf("💰 %v\n", err)
			return
		}

		if level.NeedsMultiAgent() {
			// ===== Phase 3: Multi-Agent Mode =====
			pterm.DefaultSection.Println("🚀 多 Agent 协作模式")
			pterm.Info.Printf("📂 代码输出目录: %s\n\n", absWorkDir)

			agentPool := pool.NewPool(eventBus)

			// Register agents
			sup := supervisor.New(gearProvider, eventBus, analysis, graphStore)
			coderAgent := coder.NewWithBus(gearProvider, registry, eventBus, graphStore)
			testerAgent := tester.New(eventBus, registry)
			reviewerAgent := reviewer.New(gearProvider, eventBus, registry)

			agentPool.Register(sup)
			agentPool.Register(coderAgent)
			agentPool.Register(testerAgent)
			agentPool.Register(reviewerAgent)

			// Start all agents
			agentPool.Start(ctx)

			// Wait for Supervisor (first registered agent) to finish,
			// then gracefully shut down the remaining agents.
			agentPool.WaitForRole(protocol.RoleSupervisor)
			agentPool.Shutdown()

		} else {
			// ===== Phase 2: Single Agent Mode =====
			pterm.DefaultSection.Println("🚀 单 Agent 模式")
			pterm.Info.Printf("📂 代码输出目录: %s\n\n", absWorkDir)

			agent := coder.NewWithBus(gearProvider, registry, eventBus, graphStore)
			if err := agent.RunStandalone(ctx, analysis); err != nil {
				logger.L().Fatalw("Coder Agent failed", "error", err)
			}
		}

		// Evolution: evaluate code quality with retry loop
		evolver := evolution.NewEvolver()
		pterm.DefaultSection.Println("📊 进化引擎: 评估代码质量")

		for attempt := 0; attempt <= evolver.MaxRetries; attempt++ {
			evoScore := evolver.Evaluate(absWorkDir)

			if !evolver.ShouldRetry(evoScore) {
				pterm.Success.Printf("✅ 代码质量评分: %.0f%%\n", evoScore.Total*100)
				break
			}

			if attempt == evolver.MaxRetries {
				pterm.Warning.Printf("⚠️ 已达最大重试次数，最终评分: %.0f%%\n", evoScore.Total*100)
				break
			}

			// Budget check before retry
			if err := guard.Check(); err != nil {
				pterm.Warning.Printf("💰 预算已耗尽，停止重试: %v\n", err)
				break
			}

			// Auto-retry: inject error context and re-run the Coder Agent.
			retryPrompt := evolver.BuildRetryPrompt(evoScore)
			pterm.Warning.Printf("⚠️ 评分 %.0f%% 低于阈值，第 %d 次自动修复中...\n", evoScore.Total*100, attempt+1)
			logger.L().Warnw("Evolution retry triggered",
				"attempt", attempt+1,
				"score", evoScore.Total,
				"compile_pass", evoScore.CompilePass,
			)

			// Clone analysis and inject error context as the new intent.
			retryAnalysis := *analysis
			retryAnalysis.CoreIntent = retryPrompt

			retryAgent := coder.NewWithBus(gearProvider, registry, eventBus, graphStore)
			if err := retryAgent.RunStandalone(ctx, &retryAnalysis); err != nil {
				logger.L().Warnw("Evolution retry coder failed", "attempt", attempt+1, "error", err)
			}
		}

		// Final cost and budget summary
		display.PrintCostSummary()
		pterm.Info.Printf("💰 剩余预算: $%.4f\n", guard.Remaining())

		// Audit summary
		if auditLogger != nil {
			if stats, err := auditLogger.Stats(); err == nil {
				pterm.Info.Printf("📋 审计统计: 共 %d 次工具调用, %d 次成功, %d 次被阻止\n",
					stats.Total, stats.Approved, stats.Blocked)
			}
		}
	},
}
