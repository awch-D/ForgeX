// Package reviewer implements the Reviewer Agent.
// It reviews generated code and provides quality scores.
package reviewer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/awch-D/ForgeX/forgex-agent/protocol"
	"github.com/awch-D/ForgeX/forgex-core/logger"
	"github.com/awch-D/ForgeX/forgex-llm/provider"
	"github.com/awch-D/ForgeX/forgex-mcp/tools"
)

const reviewPrompt = `You are ForgeX Code Reviewer — a strict but fair senior engineer.
You review generated code for quality, security, and best practices.

Given the list of files, review each and provide an overall assessment.

Respond in JSON:
{
  "score": 85,
  "passed": true,
  "feedback": "Overall the code is well-structured. Minor suggestions: ..."
}

Scoring guide:
- 90-100: Excellent, production-ready
- 80-89: Good, minor improvements possible
- 70-79: Acceptable, some issues to address
- Below 70: Needs rework

Set passed=true if score >= 70.`

// Agent is the code reviewer.
type Agent struct {
	llm      provider.Provider
	bus      *protocol.EventBus
	inbox    <-chan protocol.Message
	registry *tools.Registry
}

// New creates a reviewer agent.
func New(llm provider.Provider, bus *protocol.EventBus, registry *tools.Registry) *Agent {
	inbox := bus.Subscribe(protocol.RoleReviewer, 20)
	return &Agent{llm: llm, bus: bus, inbox: inbox, registry: registry}
}

func (a *Agent) Role() protocol.AgentRole { return protocol.RoleReviewer }

// Run waits for review requests.
func (a *Agent) Run(ctx context.Context) error {
	for {
		select {
		case msg, ok := <-a.inbox:
			if !ok {
				return nil
			}
			if msg.Type == protocol.MsgReview {
				a.handleReview(ctx, msg)
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (a *Agent) handleReview(ctx context.Context, msg protocol.Message) {
	logger.L().Infow("📝 Reviewer: starting code review")

	// List all files
	listResult := a.registry.Execute("list_dir", map[string]string{"path": "."})
	if !listResult.Success {
		logger.L().Warnw("Reviewer: failed to list files", "error", listResult.Error)
		return
	}

	// Read key source files
	var codeContent strings.Builder
	codeContent.WriteString("Project files:\n")
	codeContent.WriteString(listResult.Output)
	codeContent.WriteString("\n")

	// Read .go files from common locations
	for _, dir := range []string{".", "cmd", "internal/handler", "internal/middleware", "internal/db", "internal/model"} {
		dirResult := a.registry.Execute("list_dir", map[string]string{"path": dir})
		if !dirResult.Success {
			continue
		}
		for _, line := range strings.Split(dirResult.Output, "\n") {
			if strings.Contains(line, ".go") {
				fname := strings.TrimPrefix(line, "[file] ")
				path := dir + "/" + fname
				if dir == "." {
					path = fname
				}
				fileResult := a.registry.Execute("read_file", map[string]string{"path": path})
				if fileResult.Success {
					codeContent.WriteString(fmt.Sprintf("\n--- %s ---\n%s\n", path, fileResult.Output))
				}
			}
		}
	}

	// Call LLM for review
	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: reviewPrompt},
		{Role: provider.RoleUser, Content: fmt.Sprintf("Review this code:\n\n%s", codeContent.String())},
	}

	resp, err := a.llm.Generate(ctx, messages, &provider.Options{
		Temperature: 0.3,
		JSONMode:    true,
		MaxTokens:   2048,
	})
	if err != nil {
		logger.L().Warnw("Reviewer: LLM call failed", "error", err)
		return
	}

	cleanJSON := extractJSON(resp.Content)
	var review protocol.ReviewPayload
	if err := json.Unmarshal([]byte(cleanJSON), &review); err != nil {
		logger.L().Warnw("Reviewer: failed to parse review", "error", err)
		return
	}

	a.bus.Publish(ctx, protocol.Message{
		Sender:   protocol.RoleReviewer,
		Receiver: protocol.RoleSupervisor,
		Type:     protocol.MsgReview,
		Payload:  review,
	})

	logger.L().Infow("📝 Reviewer: review complete", "score", review.Score, "passed", review.Passed)
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
