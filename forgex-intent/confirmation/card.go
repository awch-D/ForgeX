// Package confirmation renders the final parsed intent as a CLI card.
package confirmation

import (
	"fmt"
	"strings"

	"github.com/pterm/pterm"

	"github.com/awch-D/ForgeX/forgex-core/types"
	"github.com/awch-D/ForgeX/forgex-intent/parser"
)

// RenderCard prints a beautiful summary of the task analysis to the console.
// Returns true if user confirms, false if user declines.
func RenderCard(analysis *parser.TaskAnalysis) bool {
	fmt.Println()

	// Gear level representation
	levelStr := analysis.EstimatedLevel.String()
	if analysis.EstimatedLevel == 0 {
		levelStr = types.L2.String() // Default fallback
	}

	// Level-based emoji
	levelEmoji := "🟢"
	switch {
	case analysis.EstimatedLevel >= 4:
		levelEmoji = "🔴"
	case analysis.EstimatedLevel >= 3:
		levelEmoji = "🟠"
	case analysis.EstimatedLevel >= 2:
		levelEmoji = "🟡"
	}

	// Tech Stack
	techStack := strings.Join(analysis.TechStack, ", ")
	if techStack == "" {
		techStack = "未指定 / 自动推断"
	}

	// 1. Header
	pterm.DefaultSection.Println("🎯 ForgeX 需求确认")

	// 2. Basic Info Table
	tableData := pterm.TableData{
		{"🎯 核心意图", pterm.LightCyan(analysis.CoreIntent)},
		{levelEmoji + " 任务挡位", pterm.Bold.Sprint(levelStr)},
		{"🛠️ 技术栈", techStack},
		{"📁 预计文件数", fmt.Sprintf("%d 个", len(analysis.ExecutionPlan))},
	}
	pterm.DefaultTable.WithHasHeader(false).WithData(tableData).WithBoxed().Render()

	// 3. Execution Plan bullet points
	if len(analysis.ExecutionPlan) > 0 {
		fmt.Println()
		pterm.Info.Println("📝 执行计划:")

		var listItems []pterm.BulletListItem
		for i, step := range analysis.ExecutionPlan {
			prefix := fmt.Sprintf("%d.", i+1)
			listItems = append(listItems, pterm.BulletListItem{Level: 0, Text: prefix + " " + step})
		}
		pterm.DefaultBulletList.WithItems(listItems).Render()
	}

	// 4. Files to modify
	if len(analysis.FilesToModify) > 0 {
		fmt.Println()
		pterm.Info.Println("📂 预计变更文件:")

		for _, file := range analysis.FilesToModify {
			fmt.Printf("  %s %s\n", pterm.Green("→"), file)
		}
	}

	// 5. Execution mode hint
	fmt.Println()
	if analysis.EstimatedLevel >= 3 {
		pterm.DefaultBox.WithTitle("🚀 执行模式").Println(
			"多 Agent 协作模式\nSupervisor → Coder × N → Tester → Reviewer")
	} else {
		pterm.DefaultBox.WithTitle("🚀 执行模式").Println(
			"单 Agent 高效模式\nCoder Agent → 自动构建验证")
	}

	// 6. User confirmation
	fmt.Println()
	result, _ := pterm.DefaultInteractiveConfirm.
		WithDefaultText("确认执行此任务?").
		WithDefaultValue(true).
		Show()

	if !result {
		pterm.Warning.Println("已取消执行")
	} else {
		fmt.Println()
		pterm.Success.Println("任务已确认，开始执行...")
	}

	return result
}
