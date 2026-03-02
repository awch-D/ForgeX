package commands

import (
	"fmt"

	"github.com/spf13/cobra"
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
		fmt.Println("🎯 Task received:", args[0])
		fmt.Println("⚙️ Analyzing complexity...")
		fmt.Println("🚧 [Phase 1+ required] Agent execution not yet implemented.")
		fmt.Println("   Stay tuned! 🚀")
	},
}
