package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/awch-D/ForgeX/forgex-core/config"
	"github.com/awch-D/ForgeX/forgex-core/logger"
	"github.com/awch-D/ForgeX/forgex-intent/clarifier"
	"github.com/awch-D/ForgeX/forgex-intent/confirmation"
	"github.com/awch-D/ForgeX/forgex-llm/cost"
	"github.com/awch-D/ForgeX/forgex-llm/litellm"
)

const version = "0.1.0-alpha"

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

		// Initialize LiteLLM Provider
		provider := litellm.NewClient(&cfg.LLM)

		// Run the Intent Clarifier Interactive Logic
		ctx := cmd.Context()
		analysis, _, err := clarifier.Run(ctx, provider, initialPrompt)
		if err != nil {
			logger.L().Fatalw("Clarifier failed", "error", err)
		}

		// Render the parsed and finalized intent card
		confirmation.RenderCard(analysis)

		// Check Costs
		_, totalCost := cost.Global().Summary()
		logger.L().Infof("🧠 意图分析完成 | 预估 LLM 本轮花费: 约 $%.4f", totalCost)
	},
}
