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

	var codeContent strings.Builder
	codeContent.WriteString("Project files:\n")

	// Dynamically discover all .go files by recursively scanning directories
	goFiles := a.discoverGoFiles(".", 0, 4) // max depth = 4
	logger.L().Infow("📝 Reviewer: discovered source files", "count", len(goFiles))

	if len(goFiles) == 0 {
		logger.L().Warnw("Reviewer: no .go files found")
		return
	}

	// Read each discovered file (cap at 20 files to keep within LLM context)
	maxFiles := 20
	if len(goFiles) < maxFiles {
		maxFiles = len(goFiles)
	}
	for _, path := range goFiles[:maxFiles] {
		fileResult := a.registry.Execute("read_file", map[string]string{"path": path})
		if fileResult.Success {
			codeContent.WriteString(fmt.Sprintf("\n--- %s ---\n%s\n", path, fileResult.Output))
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

	cleanJSON := protocol.ExtractJSON(resp.Content)
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

// discoverGoFiles recursively scans directories via MCP tools to find .go files.
func (a *Agent) discoverGoFiles(dir string, depth, maxDepth int) []string {
	if depth > maxDepth {
		return nil
	}

	result := a.registry.Execute("list_dir", map[string]string{"path": dir})
	if !result.Success {
		return nil
	}

	var goFiles []string
	for _, line := range strings.Split(result.Output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "[file] ") {
			name := strings.TrimPrefix(line, "[file] ")
			if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
				path := name
				if dir != "." {
					path = dir + "/" + name
				}
				goFiles = append(goFiles, path)
			}
		} else if strings.HasPrefix(line, "[dir] ") {
			subdir := strings.TrimPrefix(line, "[dir] ")
			// Skip hidden dirs, vendor, node_modules
			if strings.HasPrefix(subdir, ".") || subdir == "vendor" || subdir == "node_modules" {
				continue
			}
			subpath := subdir
			if dir != "." {
				subpath = dir + "/" + subdir
			}
			goFiles = append(goFiles, a.discoverGoFiles(subpath, depth+1, maxDepth)...)
		}
	}
	return goFiles
}
