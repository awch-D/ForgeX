package commands

import (
	"fmt"
	"net/http"
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
	gear "github.com/awch-D/ForgeX/forgex-gear"
	"github.com/awch-D/ForgeX/forgex-intent/clarifier"
	"github.com/awch-D/ForgeX/forgex-intent/confirmation"
	"github.com/awch-D/ForgeX/forgex-llm/cost"
	"github.com/awch-D/ForgeX/forgex-llm/litellm"
	"github.com/awch-D/ForgeX/forgex-mcp/tools"
	"github.com/awch-D/ForgeX/forgex-server/bus"
)

const version = "0.3.0-alpha"

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
	outputDir string
	dashPort  int
)

func init() {
	runCmd.Flags().StringVarP(&outputDir, "output", "o", "", "Output directory (default: ./forgex-output)")
	runCmd.Flags().IntVar(&dashPort, "dash", 0, "Start a local websocket dashboard server on this port (e.g. 8080)")
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

		// Initialize LLM Provider
		llmProvider := litellm.NewClient(&cfg.LLM)

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

		// Gear Evaluation: decide execution strategy
		fmt.Println()
		level := gear.Evaluate(analysis)
		pterm.Info.Printf("⚙️ 任务评级: %s\n", level.String())

		// Setup event bus globally so dash server can listen
		eventBus := protocol.NewEventBus()
		defer eventBus.Close()

		if dashPort > 0 {
			hub := bus.NewHub(eventBus)
			go hub.Run()

			http.HandleFunc("/ws", hub.HandleWebSocket)
			pterm.Info.Printf("🖥 Dashboard WebSocket Server starting on ws://localhost:%d/ws\n", dashPort)
			go func() {
				// Don't care about the error, background service
				_ = http.ListenAndServe(fmt.Sprintf(":%d", dashPort), nil)
			}()
		}

		if level.NeedsMultiAgent() {
			// ===== Phase 3: Multi-Agent Mode =====
			pterm.DefaultSection.Println("🚀 多 Agent 协作模式")
			pterm.Info.Printf("📂 代码输出目录: %s\n\n", absWorkDir)

			agentPool := pool.NewPool(eventBus)

			// Register agents
			sup := supervisor.New(llmProvider, eventBus, analysis)
			coderAgent := coder.NewWithBus(llmProvider, registry, eventBus)
			testerAgent := tester.New(eventBus, registry)

			agentPool.Register(sup)
			agentPool.Register(coderAgent)
			agentPool.Register(testerAgent)

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

			agent := coder.NewWithBus(llmProvider, registry, eventBus)
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
