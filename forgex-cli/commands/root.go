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
	"github.com/awch-D/ForgeX/forgex-agent/supervisor"
	"github.com/awch-D/ForgeX/forgex-agent/tester"
	"github.com/awch-D/ForgeX/forgex-core/config"
	"github.com/awch-D/ForgeX/forgex-core/logger"
	"github.com/awch-D/ForgeX/forgex-gear"
	"github.com/awch-D/ForgeX/forgex-intent/clarifier"
	"github.com/awch-D/ForgeX/forgex-intent/confirmation"
	"github.com/awch-D/ForgeX/forgex-llm/cost"
	"github.com/awch-D/ForgeX/forgex-llm/litellm"
	"github.com/awch-D/ForgeX/forgex-mcp/tools"
)

const version = "0.3.0-alpha"

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
var outputDir string

func init() {
	runCmd.Flags().StringVarP(&outputDir, "output", "o", "", "Output directory (default: ./forgex-output)")
}

var runCmd = &cobra.Command{
	Use:   "run [task description]",
	Short: "Run a coding task",
	Long:  "Describe what you want ForgeX to build, and it will autonomously create the code.",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		initialPrompt := args[0]

		cfg, err := config.Load()
		if err != nil {
			logger.L().Fatalw("Failed to load config", "error", err)
		}

		logger.L().Infow("ForgeX initialized", "LLM", cfg.LLM.Provider, "Model", cfg.LLM.Model, "Task", initialPrompt)

		// Initialize LLM Provider
		llmProvider := litellm.NewClient(&cfg.LLM)

		// Phase 1: Intent Clarification
		ctx := cmd.Context()
		analysis, _, err := clarifier.Run(ctx, llmProvider, initialPrompt)
		if err != nil {
			logger.L().Fatalw("Clarifier failed", "error", err)
		}

		confirmation.RenderCard(analysis)

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

		// Gear Evaluation: decide execution strategy
		fmt.Println()
		level := gear.Evaluate(analysis)
		pterm.Info.Printf("⚙️ 任务评级: %s\n", level.String())

		if level.NeedsMultiAgent() {
			// ===== Phase 3: Multi-Agent Mode =====
			pterm.DefaultSection.Println("🚀 多 Agent 协作模式")
			pterm.Info.Printf("📂 代码输出目录: %s\n\n", absWorkDir)

			bus := protocol.NewEventBus()
			defer bus.Close()

			agentPool := pool.NewPool(bus)

			// Register agents
			sup := supervisor.New(llmProvider, bus, analysis)
			coderAgent := coder.NewWithBus(llmProvider, registry, bus)
			testerAgent := tester.New(bus, registry)

			agentPool.Register(sup)
			agentPool.Register(coderAgent)
			agentPool.Register(testerAgent)

			// Start all agents
			agentPool.Start(ctx)
			agentPool.Wait()

		} else {
			// ===== Phase 2: Single Agent Mode =====
			pterm.DefaultSection.Println("🚀 单 Agent 模式")
			pterm.Info.Printf("📂 代码输出目录: %s\n\n", absWorkDir)

			agent := coder.New(llmProvider, registry)
			if err := agent.RunStandalone(ctx, analysis); err != nil {
				logger.L().Fatalw("Coder Agent failed", "error", err)
			}
		}

		// Final cost summary
		totalTokens, totalCost := cost.Global().Summary()
		fmt.Println()
		pterm.DefaultBox.WithTitle("💰 成本汇总").Println(
			fmt.Sprintf("Token 总消耗: %d\n预估花费: $%.4f", totalTokens, totalCost),
		)
	},
}
