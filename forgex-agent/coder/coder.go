// Package coder implements the Coder Agent.
// In Phase 3, the Coder listens on the EventBus for task assignments from
// the Supervisor, executes the LLM tool-use loop, and reports results back.
package coder

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pterm/pterm"

	"github.com/awch-D/ForgeX/forgex-agent/protocol"
	"github.com/awch-D/ForgeX/forgex-core/logger"
	"github.com/awch-D/ForgeX/forgex-intent/parser"
	"github.com/awch-D/ForgeX/forgex-llm/provider"
	"github.com/awch-D/ForgeX/forgex-mcp/tools"
)

const maxIterations = 15

var coderSystemPrompt = `You are ForgeX Coder Agent — an expert autonomous programmer.
You receive a specific coding task and implement it by calling tools.

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
  "summary": "Created X files implementing Y feature",
  "files_created": ["file1.go", "file2.go"]
}

## Rules
- Write COMPLETE, PRODUCTION-QUALITY code. No placeholders, no TODOs.
- Create all necessary files including go.mod if needed.
- IMPORTANT: Always use GOWORK=off prefix for go build/test/vet/get commands (e.g. "GOWORK=off go build ./...") because there may be a parent go.work file that interferes.
- After writing code, run build commands to verify.
- If a build fails, read the error and fix the code.
- Maximum iterations: 15.
`

// Agent is the Coder Agent that writes code via LLM tool-use loops.
type Agent struct {
	llm      provider.Provider
	registry *tools.Registry
	bus      *protocol.EventBus
	inbox    <-chan protocol.Message
}

// New creates a new Coder Agent.
// If bus is nil, runs in standalone (Phase 2) mode.
func New(llm provider.Provider, registry *tools.Registry) *Agent {
	return &Agent{llm: llm, registry: registry}
}

// NewWithBus creates a Coder Agent wired to the EventBus (Phase 3 mode).
func NewWithBus(llm provider.Provider, registry *tools.Registry, bus *protocol.EventBus) *Agent {
	inbox := bus.Subscribe(protocol.RoleCoder, 50)
	return &Agent{llm: llm, registry: registry, bus: bus, inbox: inbox}
}

func (a *Agent) Role() protocol.AgentRole { return protocol.RoleCoder }

type agentResponse struct {
	Thought      string           `json:"thought"`
	ToolCalls    []tools.ToolCall `json:"tool_calls,omitempty"`
	Done         bool             `json:"done,omitempty"`
	Summary      string           `json:"summary,omitempty"`
	FilesCreated []string         `json:"files_created,omitempty"`
}

// Run starts the Coder in EventBus mode: listen for tasks and execute them.
func (a *Agent) Run(ctx context.Context) error {
	if a.inbox == nil {
		return fmt.Errorf("coder: no inbox configured, use RunStandalone for Phase 2 mode")
	}

	for {
		select {
		case msg, ok := <-a.inbox:
			if !ok {
				return nil
			}
			if msg.Type == protocol.MsgTask {
				a.handleTask(ctx, msg)
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (a *Agent) handleTask(ctx context.Context, msg protocol.Message) {
	payloadJSON, _ := json.Marshal(msg.Payload)
	var task protocol.TaskPayload
	json.Unmarshal(payloadJSON, &task)

	logger.L().Infow("🔨 Coder: received task", "task_id", task.TaskID)

	prompt := fmt.Sprintf("Task: %s\n\nContext: %s", task.Description, task.Context)
	files, summary, err := a.execute(ctx, prompt)

	result := protocol.ResultPayload{
		TaskID:       task.TaskID,
		FilesCreated: files,
		Summary:      summary,
		Success:      err == nil,
	}
	if err != nil {
		result.Error = err.Error()
	}

	a.bus.Publish(ctx, protocol.Message{
		Sender:   protocol.RoleCoder,
		Receiver: protocol.RoleSupervisor,
		Type:     protocol.MsgResult,
		Payload:  result,
	})
}

// RunStandalone executes the full coding loop in Phase 2 (standalone) mode.
func (a *Agent) RunStandalone(ctx context.Context, analysis *parser.TaskAnalysis) error {
	taskJSON, _ := json.MarshalIndent(analysis, "", "  ")
	prompt := fmt.Sprintf("Please implement the following task:\n\n%s", string(taskJSON))
	_, _, err := a.execute(ctx, prompt)
	return err
}

// execute is the core LLM tool-use loop.
func (a *Agent) execute(ctx context.Context, taskPrompt string) (filesCreated []string, summary string, err error) {
	history := []provider.Message{
		{Role: provider.RoleSystem, Content: coderSystemPrompt + "\n\n" + a.registry.ToolsForLLM()},
		{Role: provider.RoleUser, Content: taskPrompt},
	}

	for i := 0; i < maxIterations; i++ {
		iterLabel := fmt.Sprintf("[Iter %d/%d]", i+1, maxIterations)

		spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("🤖 %s Coder 思考中...", iterLabel))

		resp, err := a.llm.Generate(ctx, history, &provider.Options{
			Temperature: 0.2,
			JSONMode:    true,
			MaxTokens:   8192,
		})
		if err != nil {
			spinner.Fail("LLM 调用失败")
			return nil, "", fmt.Errorf("LLM generation failed at iteration %d: %w", i+1, err)
		}

		spinner.Success(fmt.Sprintf("%s Coder 响应就绪", iterLabel))

		cleanJSON := extractJSON(resp.Content)
		var agentResp agentResponse
		if err := json.Unmarshal([]byte(cleanJSON), &agentResp); err != nil {
			logger.L().Warnw("Failed to parse agent response, retrying",
				"error", err, "raw", cleanJSON[:min(len(cleanJSON), 300)])
			history = append(history, provider.Message{Role: provider.RoleAssistant, Content: resp.Content})
			history = append(history, provider.Message{Role: provider.RoleUser, Content: "Your response was not valid JSON. Please respond with valid JSON matching the required format."})
			continue
		}

		pterm.Info.Printf("💭 %s\n", agentResp.Thought)

		if agentResp.Done {
			fmt.Println()
			pterm.DefaultBox.WithTitle("✅ 任务完成").Println(agentResp.Summary)
			// Render file tree
			if len(agentResp.FilesCreated) > 0 {
				renderFileTree(agentResp.FilesCreated)
			}
			return agentResp.FilesCreated, agentResp.Summary, nil
		}

		if len(agentResp.ToolCalls) == 0 {
			history = append(history, provider.Message{Role: provider.RoleAssistant, Content: resp.Content})
			history = append(history, provider.Message{Role: provider.RoleUser, Content: "You didn't call any tools and didn't mark done. Please either call tools or set done=true."})
			continue
		}

		var toolResultsSB strings.Builder
		toolResultsSB.WriteString("Tool execution results:\n")

		for j, tc := range agentResp.ToolCalls {
			// Rich tool output
			switch tc.Name {
			case "write_file":
				bytes := len(tc.Args["content"])
				pterm.Success.Printf("  📄 [%d] 写入 %s (%d bytes)\n", j+1, pterm.LightCyan(tc.Args["path"]), bytes)
			case "run_command":
				cmd := tc.Args["command"]
				if len(cmd) > 60 {
					cmd = cmd[:60] + "..."
				}
				pterm.Success.Printf("  ⚡ [%d] 执行 %s\n", j+1, pterm.LightYellow(cmd))
			case "read_file":
				pterm.Success.Printf("  📖 [%d] 读取 %s\n", j+1, pterm.LightCyan(tc.Args["path"]))
			case "list_dir":
				pterm.Success.Printf("  📂 [%d] 列目录 %s\n", j+1, tc.Args["path"])
			default:
				pterm.Success.Printf("  🔧 [%d] %s\n", j+1, tc.Name)
			}
			result := a.registry.Execute(tc.Name, tc.Args)

			status := "✅ success"
			if !result.Success {
				status = "❌ failed: " + result.Error
			}

			output := result.Output
			if len(output) > 2000 {
				output = output[:2000] + "\n... (truncated)"
			}
			toolResultsSB.WriteString(fmt.Sprintf("\n--- Tool: %s [%s] ---\n%s\n", tc.Name, status, output))
		}

		history = append(history, provider.Message{Role: provider.RoleAssistant, Content: resp.Content})
		history = append(history, provider.Message{Role: provider.RoleUser, Content: toolResultsSB.String()})
	}

	return nil, "", fmt.Errorf("agent exceeded max iterations (%d)", maxIterations)
}

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

// renderFileTree renders a tree of created files using pterm.
func renderFileTree(files []string) {
	if len(files) == 0 {
		return
	}

	sort.Strings(files)

	// Build tree structure
	root := pterm.TreeNode{Text: "📂 生成文件", Children: []pterm.TreeNode{}}
	dirs := make(map[string]*pterm.TreeNode)

	for _, f := range files {
		dir := filepath.Dir(f)
		base := filepath.Base(f)

		if dir == "." || dir == "" {
			root.Children = append(root.Children, pterm.TreeNode{Text: "📄 " + base})
		} else {
			if _, ok := dirs[dir]; !ok {
				dirNode := &pterm.TreeNode{Text: "📁 " + dir + "/", Children: []pterm.TreeNode{}}
				dirs[dir] = dirNode
				root.Children = append(root.Children, *dirNode)
			}
			// Find and update the dir node in root.Children
			for k := range root.Children {
				if root.Children[k].Text == "📁 "+dir+"/" {
					root.Children[k].Children = append(root.Children[k].Children, pterm.TreeNode{Text: "📄 " + base})
				}
			}
		}
	}

	fmt.Println()
	pterm.DefaultTree.WithRoot(root).Render()
}
