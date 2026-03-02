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
func RenderCard(analysis *parser.TaskAnalysis) {
	fmt.Println()

	// Gear level representation
	levelStr := analysis.EstimatedLevel.String()
	if analysis.EstimatedLevel == 0 {
		levelStr = types.L2.String() // Default fallback
	}

	// Tech Stack
	techStack := strings.Join(analysis.TechStack, ", ")
	if techStack == "" {
		techStack = "未指定 / 自动推断"
	}

	// 1. Basic Info Table
	tableData := pterm.TableData{
		{"🎯 核心意图", pterm.LightCyan(analysis.CoreIntent)},
		{"⚙️ 任务挡位", levelStr},
		{"🛠️ 技术栈", techStack},
	}
	pterm.DefaultTable.WithHasHeader(false).WithData(tableData).WithBoxed().Render()

	// 2. Execution Plan bullet points
	if len(analysis.ExecutionPlan) > 0 {
		fmt.Println()
		pterm.DefaultSection.Println("📝 执行计划预览")
		
		var listItems []pterm.BulletListItem
		for _, step := range analysis.ExecutionPlan {
			listItems = append(listItems, pterm.BulletListItem{Level: 0, Text: step})
		}
		pterm.DefaultBulletList.WithItems(listItems).Render()
	}

	// 3. Files to modify
	if len(analysis.FilesToModify) > 0 {
		fmt.Println()
		pterm.DefaultSection.Println("📂 预计变更文件")
		
		for _, file := range analysis.FilesToModify {
			pterm.Success.Println(file)
		}
	}

	fmt.Println()
	pterm.DefaultHeader.WithFullWidth().WithMargin(1).Println("准备移交执行引擎 (Phase 2)")
}
