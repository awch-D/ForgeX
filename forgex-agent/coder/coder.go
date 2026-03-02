// Package coder implements the single Coder Agent for Phase 2.
// It drives the LLM in a tool-use loop: analyze → call tools → reflect → repeat.
package coder

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pterm/pterm"

	"github.com/awch-D/ForgeX/forgex-core/logger"
	"github.com/awch-D/ForgeX/forgex-intent/parser"
	"github.com/awch-D/ForgeX/forgex-llm/provider"
	"github.com/awch-D/ForgeX/forgex-mcp/tools"
)

const maxIterations = 15

var coderSystemPrompt = `You are ForgeX Coder Agent — an expert autonomous programmer.
You have been given a task analysis with a clear intent, tech stack, and execution plan.
Your job is to implement the code by calling tools.

## Response Format
You MUST respond in JSON with exactly one of these two formats:

### When you need to call tools:
{
  "thought": "Brief reasoning about what to do next",
  "tool_calls": [
    {"name": "write_file", "args": {"path": "main.go", "content": "package main..."}},
    {"name": "run_command", "args": {"command": "go build ./..."}}
  ]
}

### When you are completely done:
{
  "thought": "Summary of what was accomplished",
  "done": true,
  "summary": "Created X files implementing Y feature"
}

## Rules
- Write COMPLETE, PRODUCTION-QUALITY code. No placeholders, no TODOs.
- Create all necessary files including go.mod if needed.
- After writing code, run tests or build commands to verify.
- If a build/test fails, read the error and fix the code.
- Maximum iterations: 15. Plan efficiently.
`

// Agent is the Coder Agent that drives the tool-use loop.
type Agent struct {
	llm      provider.Provider
	registry *tools.Registry
}

// New creates a new Coder Agent.
func New(llm provider.Provider, registry *tools.Registry) *Agent {
	return &Agent{llm: llm, registry: registry}
}

type agentResponse struct {
	Thought   string           `json:"thought"`
	ToolCalls []tools.ToolCall  `json:"tool_calls,omitempty"`
	Done      bool             `json:"done,omitempty"`
	Summary   string           `json:"summary,omitempty"`
}

// Run executes the full coding loop for the given task analysis.
func (a *Agent) Run(ctx context.Context, analysis *parser.TaskAnalysis) error {
	// Build the initial context message
	taskJSON, _ := json.MarshalIndent(analysis, "", "  ")

	history := []provider.Message{
		{Role: provider.RoleSystem, Content: coderSystemPrompt + "\n\n" + a.registry.ToolsForLLM()},
		{Role: provider.RoleUser, Content: fmt.Sprintf("Please implement the following task:\n\n%s", string(taskJSON))},
	}

	for i := 0; i < maxIterations; i++ {
		iterLabel := fmt.Sprintf("[Iter %d/%d]", i+1, maxIterations)

		spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("🤖 %s Agent 思考中...", iterLabel))

		resp, err := a.llm.Generate(ctx, history, &provider.Options{
			Temperature: 0.2,
			JSONMode:    true,
			MaxTokens:   8192,
		})
		if err != nil {
			spinner.Fail("LLM 调用失败")
			return fmt.Errorf("LLM generation failed at iteration %d: %w", i+1, err)
		}

		spinner.Success(fmt.Sprintf("%s Agent 响应就绪", iterLabel))

		// Parse the structured response
		cleanJSON := extractJSON(resp.Content)
		var agentResp agentResponse
		if err := json.Unmarshal([]byte(cleanJSON), &agentResp); err != nil {
			logger.L().Warnw("Failed to parse agent response, retrying", "error", err, "raw", cleanJSON[:min(len(cleanJSON), 300)])
			// Add the raw content as assistant and ask to fix format
			history = append(history, provider.Message{Role: provider.RoleAssistant, Content: resp.Content})
			history = append(history, provider.Message{Role: provider.RoleUser, Content: "Your response was not valid JSON. Please respond with valid JSON matching the required format."})
			continue
		}

		// Show thought
		pterm.Info.Printf("💭 %s\n", agentResp.Thought)

		// Check if done
		if agentResp.Done {
			fmt.Println()
			pterm.DefaultBox.WithTitle("✅ 任务完成").Println(agentResp.Summary)
			return nil
		}

		// Execute tool calls
		if len(agentResp.ToolCalls) == 0 {
			logger.L().Warn("Agent returned no tool calls and not done, nudging...")
			history = append(history, provider.Message{Role: provider.RoleAssistant, Content: resp.Content})
			history = append(history, provider.Message{Role: provider.RoleUser, Content: "You didn't call any tools and didn't mark done. Please either call tools or set done=true."})
			continue
		}

		// Execute each tool call and collect results
		var toolResultsSB strings.Builder
		toolResultsSB.WriteString("Tool execution results:\n")

		for j, tc := range agentResp.ToolCalls {
			pterm.Success.Printf("  🔧 [%d] %s\n", j+1, tc.Name)
			result := a.registry.Execute(tc.Name, tc.Args)
			
			status := "✅ success"
			if !result.Success {
				status = "❌ failed: " + result.Error
			}

			// Truncate long output for context window management
			output := result.Output
			if len(output) > 2000 {
				output = output[:2000] + "\n... (truncated)"
			}

			toolResultsSB.WriteString(fmt.Sprintf("\n--- Tool: %s [%s] ---\n%s\n", tc.Name, status, output))
		}

		// Append assistant message and tool results to history
		history = append(history, provider.Message{Role: provider.RoleAssistant, Content: resp.Content})
		history = append(history, provider.Message{Role: provider.RoleUser, Content: toolResultsSB.String()})
	}

	return fmt.Errorf("agent exceeded max iterations (%d)", maxIterations)
}

// extractJSON strips markdown code fences.
func extractJSON(raw string) string {
	s := strings.TrimSpace(raw)
	if strings.HasPrefix(s, "```") {
		idx := strings.Index(s, "\n")
		if idx != -1 {
			s = s[idx+1:]
		}
		if lastIdx := strings.LastIndex(s, "```"); lastIdx != -1 {
			s = s[:lastIdx]
		}
		s = strings.TrimSpace(s)
	}
	return s
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
