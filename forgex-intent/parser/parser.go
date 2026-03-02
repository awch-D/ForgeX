// Package parser defines the structures for intent parsing.
package parser

import (
	"context"
	"encoding/json"
	"fmt"

	fxerr "github.com/awch-D/ForgeX/forgex-core/errors"
	"github.com/awch-D/ForgeX/forgex-core/types"
	"github.com/awch-D/ForgeX/forgex-llm/provider"
)

// TaskAnalysis represents the structured output from the LLM.
type TaskAnalysis struct {
	Status          string          `json:"status"` // "ready" or "need_info"
	CoreIntent      string          `json:"core_intent"`
	TechStack       []string        `json:"tech_stack"`
	EstimatedLevel  types.TaskLevel `json:"estimated_level"`
	MissingContext  []string        `json:"missing_context"` // Questions to ask user
	ExecutionPlan   []string        `json:"execution_plan"`
	FilesToModify   []string        `json:"files_to_modify"`
}

var systemPrompt = `You are the ForgeX Intent Clarifier v3.
Your job is to analyze a user's coding request and determine if you have enough information to generate code.
You MUST ALWAYS respond in valid JSON matching this schema:
{
  "status": "string (strictly 'ready' or 'need_info')",
  "core_intent": "string (brief summary of what needs to be built)",
  "tech_stack": ["golang", "react", "sqlite", ...],
  "estimated_level": 1|2|3|4 (1=Simple, 2=Medium, 3=Complex, 4=Cross-system),
  "missing_context": ["question1", "question2"] (only if status=need_info),
  "execution_plan": ["step1", "step2"],
  "files_to_modify": ["cmd/main.go", "internal/db/db.go"]
}

Guidelines:
- Don't ask more than 2-3 questions at a time.
- If the user provides a very general request like "build a web app", status=need_info, and ask about framework, DB, auth.
- If the request is specific enough to start coding (e.g. "Create a Go CLI tool to ping Google"), status=ready.
`

// Parse parses the current conversation history to produce a TaskAnalysis.
func Parse(ctx context.Context, llm provider.Provider, history []provider.Message) (*TaskAnalysis, error) {
	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: systemPrompt},
	}
	messages = append(messages, history...)

	opts := &provider.Options{
		Temperature: 0.1, // Low temp for structured JSON output
		JSONMode:    true,
	}

	resp, err := llm.Generate(ctx, messages, opts)
	if err != nil {
		return nil, fxerr.Wrap(fxerr.ErrLLMBadResponse, "intent parsing failed", err)
	}

	var analysis TaskAnalysis
	if err := json.Unmarshal([]byte(resp.Content), &analysis); err != nil {
		return nil, fxerr.Wrap(fxerr.ErrLLMBadResponse, fmt.Sprintf("failed to decode JSON from LLM: %s", resp.Content), err)
	}

	return &analysis, nil
}
