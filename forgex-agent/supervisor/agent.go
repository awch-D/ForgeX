// Package supervisor implements the project manager agent.
// It decomposes tasks, dispatches to Coder agents, coordinates Tester/Reviewer,
// and aggregates final results.
package supervisor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pterm/pterm"

	"github.com/awch-D/ForgeX/forgex-agent/protocol"
	"github.com/awch-D/ForgeX/forgex-cognition/graph"
	"github.com/awch-D/ForgeX/forgex-core/logger"
	"github.com/awch-D/ForgeX/forgex-intent/parser"
	"github.com/awch-D/ForgeX/forgex-llm/provider"
)

const supervisorPrompt = `You are ForgeX Supervisor Agent — a senior technical project manager.
Given a task analysis, you decompose it into independent sub-tasks that can be executed in parallel by Coder agents.

Respond in JSON:
{
  "plan": "Brief architecture overview",
  "tasks": [
    {
      "id": "task-1",
      "description": "Detailed description of what code to write",
      "priority": 1,
      "context": "Any helpful context (related APIs, data structures, etc.)"
    }
  ]
}

Rules:
- Each task should create 1-3 files maximum.
- Tasks should be as independent as possible.
- Order by dependency: foundational tasks first (models, DB layer), then handlers/routes, then main entry.
- Include a final task for writing go.mod and wiring everything in main.go.
- Maximum 6 tasks for any project.`

// Agent is the Supervisor that orchestrates the multi-agent workflow.
type Agent struct {
	llm      provider.Provider
	bus      *protocol.EventBus
	inbox    <-chan protocol.Message
	analysis *parser.TaskAnalysis
	graph    *graph.Store
}

// New creates a supervisor agent.
func New(llm provider.Provider, bus *protocol.EventBus, analysis *parser.TaskAnalysis, g *graph.Store) *Agent {
	inbox := bus.Subscribe(protocol.RoleSupervisor, 50)
	return &Agent{
		llm:      llm,
		bus:      bus,
		inbox:    inbox,
		analysis: analysis,
		graph:    g,
	}
}

func (a *Agent) Role() protocol.AgentRole { return protocol.RoleSupervisor }

// SubTask is a decomposed unit of work.
type SubTask struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Priority    int    `json:"priority"`
	Context     string `json:"context"`
}

type planResponse struct {
	Plan  string    `json:"plan"`
	Tasks []SubTask `json:"tasks"`
}

// Run orchestrates the full multi-agent workflow.
func (a *Agent) Run(ctx context.Context) error {
	pterm.DefaultSection.Println("📋 Supervisor: 正在制定执行计划...")

	// Step 1: Decompose the task via LLM
	taskJSON, _ := json.MarshalIndent(a.analysis, "", "  ")

	systemPrompt := supervisorPrompt
	if a.graph != nil {
		systemPrompt += a.graph.BuildSummary()
	}

	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: systemPrompt},
		{Role: provider.RoleUser, Content: fmt.Sprintf("Decompose this task:\n\n%s", string(taskJSON))},
	}

	resp, err := a.llm.Generate(ctx, messages, &provider.Options{
		Temperature: 0.3,
		JSONMode:    true,
		MaxTokens:   8192,
	})
	if err != nil {
		return fmt.Errorf("supervisor LLM call: %w", err)
	}

	var plan planResponse
	cleanJSON := protocol.ExtractJSON(resp.Content)
	if err := json.Unmarshal([]byte(cleanJSON), &plan); err != nil {
		return fmt.Errorf("parse plan: %w (raw: %s)", err, cleanJSON[:min(len(cleanJSON), 300)])
	}

	// Display the plan
	pterm.Info.Printf("📐 架构方案: %s\n", plan.Plan)
	pterm.Info.Printf("📦 子任务数: %d\n\n", len(plan.Tasks))

	for i, t := range plan.Tasks {
		pterm.Success.Printf("  [%d] %s (Priority: %d)\n", i+1, t.Description[:min(len(t.Description), 80)], t.Priority)
	}
	fmt.Println()

	// Step 2: Dispatch tasks to Coder(s)
	pendingTasks := make(map[string]*SubTask)
	for i := range plan.Tasks {
		t := &plan.Tasks[i]
		pendingTasks[t.ID] = t

		a.bus.Publish(ctx, protocol.Message{
			ID:       t.ID,
			Sender:   protocol.RoleSupervisor,
			Receiver: protocol.RoleCoder,
			Type:     protocol.MsgTask,
			Payload: protocol.TaskPayload{
				TaskID:      t.ID,
				Description: t.Description,
				Context:     t.Context,
				Priority:    t.Priority,
			},
		})
		logger.L().Infow("📤 Task dispatched", "task_id", t.ID)
	}

	// Step 3: Wait for all results
	completedTasks := 0
	totalTasks := len(plan.Tasks)
	var allResults []protocol.ResultPayload

	timeout := time.After(10 * time.Minute)

	for completedTasks < totalTasks {
		select {
		case msg, ok := <-a.inbox:
			if !ok {
				return fmt.Errorf("supervisor inbox closed")
			}
			switch msg.Type {
			case protocol.MsgResult:
				payloadJSON, _ := json.Marshal(msg.Payload)
				var result protocol.ResultPayload
				json.Unmarshal(payloadJSON, &result)

				completedTasks++
				allResults = append(allResults, result)
				status := "✅"
				if !result.Success {
					status = "❌"
				}
				pterm.Info.Printf("  %s [%d/%d] %s: %s\n", status, completedTasks, totalTasks, result.TaskID, result.Summary)

			case protocol.MsgError:
				logger.L().Warnw("Supervisor received error", "from", msg.Sender, "payload", msg.Payload)
			}

		case <-timeout:
			return fmt.Errorf("supervisor timed out waiting for %d/%d tasks", completedTasks, totalTasks)

		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Step 4: Dispatch test request
	pterm.DefaultSection.Println("🧪 Supervisor: 请求 Tester 检验代码")
	a.bus.Publish(ctx, protocol.Message{
		Sender:   protocol.RoleSupervisor,
		Receiver: protocol.RoleTester,
		Type:     protocol.MsgTest,
		Payload: protocol.TaskPayload{
			TaskID:      "test-all",
			Description: "Build and test the entire generated project",
		},
	})

	// Wait for test result
	testTimeout := time.After(2 * time.Minute)
	select {
	case msg := <-a.inbox:
		if msg.Type == protocol.MsgResult || msg.Type == protocol.MsgTest {
			payloadJSON, _ := json.Marshal(msg.Payload)
			var testResult protocol.TestPayload
			if json.Unmarshal(payloadJSON, &testResult) == nil && testResult.Passed {
				pterm.Success.Println("🧪 测试通过!")
			} else {
				pterm.Warning.Println("🧪 测试存在问题，但继续完成")
			}
		}
	case <-testTimeout:
		pterm.Warning.Println("🧪 测试超时，跳过")
	case <-ctx.Done():
		return ctx.Err()
	}

	// Step 4.5: Dispatch code review request
	pterm.DefaultSection.Println("📝 Supervisor: 请求 Reviewer 审查代码质量")
	a.bus.Publish(ctx, protocol.Message{
		Sender:   protocol.RoleSupervisor,
		Receiver: protocol.RoleReviewer,
		Type:     protocol.MsgReview,
		Payload: protocol.TaskPayload{
			TaskID:      "review-all",
			Description: "Review all generated code files for quality and correctness",
		},
	})

	// Wait for review result (3-minute timeout)
	reviewTimeout := time.After(3 * time.Minute)
	select {
	case msg := <-a.inbox:
		if msg.Type == protocol.MsgReview {
			payloadJSON, _ := json.Marshal(msg.Payload)
			var review protocol.ReviewPayload
			if json.Unmarshal(payloadJSON, &review) == nil {
				if review.Passed {
					pterm.Success.Printf("📝 代码审查通过 (评分: %d/100)\n", review.Score)
				} else {
					pterm.Warning.Printf("📝 代码审查发现问题 (评分: %d/100): %s\n", review.Score, review.Feedback)
				}
			}
		}
	case <-reviewTimeout:
		pterm.Warning.Println("📝 代码审查超时，跳过")
	case <-ctx.Done():
		return ctx.Err()
	}

	// Step 5: Signal completion
	fmt.Println()
	var fileList []string
	for _, r := range allResults {
		fileList = append(fileList, r.FilesCreated...)
	}
	pterm.DefaultBox.WithTitle("✅ 多 Agent 协作完成").Println(
		fmt.Sprintf("子任务: %d/%d 完成\n生成文件: %s", completedTasks, totalTasks, strings.Join(fileList, ", ")),
	)

	a.bus.Publish(ctx, protocol.Message{
		Sender: protocol.RoleSupervisor,
		Type:   protocol.MsgComplete,
	})

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
