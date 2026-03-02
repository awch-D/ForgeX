// Package clarifier implements the interactive CLI loop to clarify requirements.
package clarifier

import (
	"context"
	"fmt"
	"strings"

	"github.com/pterm/pterm"

	fxerr "github.com/awch-D/ForgeX/forgex-core/errors"
	"github.com/awch-D/ForgeX/forgex-intent/parser"
	"github.com/awch-D/ForgeX/forgex-llm/provider"
)

// Run starts the interactive clarification loop until the LLM reports "ready".
func Run(ctx context.Context, llm provider.Provider, initialPrompt string) (*parser.TaskAnalysis, []provider.Message, error) {
	history := []provider.Message{
		{Role: provider.RoleUser, Content: initialPrompt},
	}

	spinner, _ := pterm.DefaultSpinner.Start("🤔 正在分析需求...")
	defer func() {
		if spinner.IsActive {
			spinner.Stop()
		}
	}()

	for {
		// 1. Ask LLM to parse the current state
		analysis, err := parser.Parse(ctx, llm, history)
		if err != nil {
			spinner.Fail("分析失败")
			return nil, nil, err
		}

		if analysis.Status == "ready" || len(analysis.MissingContext) == 0 {
			spinner.Success("需求已明确！")
			return analysis, history, nil
		}

		// 2. We need more info. Stop spinner, ask user.
		spinner.Warning("需要更多细节")
		fmt.Println()
		pterm.DefaultBox.WithTitle("ForgeX 还需要了解以下细节:").WithTitleBottomRight().Println(
			strings.Join(analysis.MissingContext, "\n"),
		)

		// Ask for input
		input, err := DefaultInteractiveTextInput.Show("👉 请补充")
		if err != nil {
			return nil, nil, fxerr.Wrap(fxerr.ErrInvalidInput, "failed to read user input", err)
		}

		if strings.TrimSpace(input) == "" {
			// If user just presses Enter without providing info, we push them and try again or assume they want to proceed.
			// Let's add that to context
			input = "(用户已跳过提供更多信息，请尽力利用现有信息继续或者做出默认假设)"
			pterm.Info.Println("已跳过补充信息，ForgeX 将基于默认最佳实践推进...")
		}

		// 3. Append to history and restart loop
		history = append(history, provider.Message{
			Role:    provider.RoleAssistant,
			Content: fmt.Sprintf("Missing Context Questions:\n%s", strings.Join(analysis.MissingContext, "\n")),
		})
		history = append(history, provider.Message{
			Role:    provider.RoleUser,
			Content: input,
		})

		spinner, _ = pterm.DefaultSpinner.Start("🤔 重新评估需求...")
	}
}

// Wrapper for Pterm interactive input to allow testing/mocking if needed
var DefaultInteractiveTextInput = pterm.DefaultInteractiveTextInput.WithMultiLine(false)
