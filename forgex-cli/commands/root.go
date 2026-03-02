package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/awch-D/ForgeX/forgex-agent/coder"
	"github.com/awch-D/ForgeX/forgex-core/config"
	"github.com/awch-D/ForgeX/forgex-core/logger"
	"github.com/awch-D/ForgeX/forgex-intent/clarifier"
	"github.com/awch-D/ForgeX/forgex-intent/confirmation"
	"github.com/awch-D/ForgeX/forgex-llm/cost"
	"github.com/awch-D/ForgeX/forgex-llm/litellm"
	"github.com/awch-D/ForgeX/forgex-mcp/tools"
)

const version = "0.2.0-alpha"

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
	runCmd.Flags().StringVarP(&outputDir, "output", "o", "", "Output directory for generated code (default: ./forgex-output)")
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

		// Render the intent card
		confirmation.RenderCard(analysis)

		_, intentCost := cost.Global().Summary()
		logger.L().Infof("🧠 意图分析花费: 约 $%.4f", intentCost)

		// Phase 2: Coder Agent Execution
		fmt.Println()
		pterm.DefaultSection.Println("🚀 启动 Coder Agent")

		// Determine output directory
		workDir := outputDir
		if workDir == "" {
			workDir = filepath.Join(".", "forgex-output")
		}
		absWorkDir, _ := filepath.Abs(workDir)
		os.MkdirAll(absWorkDir, 0755)
		pterm.Info.Printf("📂 代码输出目录: %s\n\n", absWorkDir)

		// Create tool registry and agent
		registry := tools.NewRegistry(absWorkDir)
		agent := coder.New(llmProvider, registry)

		// Run the agent
		if err := agent.Run(ctx, analysis); err != nil {
			logger.L().Fatalw("Coder Agent failed", "error", err)
		}

		// Final cost summary
		totalTokens, totalCost := cost.Global().Summary()
		fmt.Println()
		pterm.DefaultBox.WithTitle("💰 成本汇总").Println(
			fmt.Sprintf("Token 总消耗: %d\n预估花费: $%.4f", totalTokens, totalCost),
		)
	},
}
