// Package parser defines the structures for intent parsing.
package parser

import (
	"context"
	"encoding/json"
	"strings"

	fxerr "github.com/awch-D/ForgeX/forgex-core/errors"
	"github.com/awch-D/ForgeX/forgex-core/types"
	"github.com/awch-D/ForgeX/forgex-llm/provider"
)

// TaskAnalysis represents the structured output from the LLM.
type TaskAnalysis struct {
	Status         string          `json:"status"` // "ready" or "need_info"
	CoreIntent     string          `json:"core_intent"`
	TechStack      []string        `json:"tech_stack"`
	EstimatedLevel types.TaskLevel `json:"estimated_level"`
	MissingContext []string        `json:"missing_context"` // Questions to ask user
	ExecutionPlan  []string        `json:"execution_plan"`
	FilesToModify  []string        `json:"files_to_modify"`
}

var systemPrompt = `You are the ForgeX Intent Clarifier v3.
Your job is to analyze a user's coding request and determine if you have enough information to generate code.
You MUST ALWAYS respond in valid JSON only. Do NOT wrap your response in markdown code blocks.
Do NOT use triple backticks. Output raw JSON directly.
The JSON must match this schema:
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

// extractJSON strips markdown code fences and extracts pure JSON from LLM output.
func extractJSON(raw string) string {
	s := strings.TrimSpace(raw)

	// Strip ```json ... ``` or ``` ... ```
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

	// Try to find JSON object boundaries if raw text surrounds it
	if start := strings.Index(s, "{"); start > 0 {
		s = s[start:]
	}
	if end := strings.LastIndex(s, "}"); end >= 0 && end < len(s)-1 {
		s = s[:end+1]
	}

	return s
}

// Parse parses the current conversation history to produce a TaskAnalysis.
// It includes retry and graceful degradation:
//   - First attempt: parse LLM response as JSON
//   - Retry: re-prompt LLM with explicit JSON instruction
//   - Fallback: build a minimal TaskAnalysis from the raw prompt
func Parse(ctx context.Context, llm provider.Provider, history []provider.Message) (*TaskAnalysis, error) {
	messages := []provider.Message{
		{Role: provider.RoleSystem, Content: systemPrompt},
	}
	messages = append(messages, history...)

	opts := &provider.Options{
		Temperature: 0.1, // Low temp for structured JSON output
		JSONMode:    true,
	}

	// Attempt 1: normal parse
	resp, err := llm.Generate(ctx, messages, opts)
	if err != nil {
		return nil, fxerr.Wrap(fxerr.ErrLLMBadResponse, "intent parsing failed", err)
	}

	cleanJSON := extractJSON(resp.Content)

	var analysis TaskAnalysis
	if err := json.Unmarshal([]byte(cleanJSON), &analysis); err == nil {
		return &analysis, nil
	}

	// Attempt 2: retry with explicit instruction
	retryMsgs := append(messages, provider.Message{
		Role:    provider.RoleUser,
		Content: "Your previous response was not valid JSON. Please respond with ONLY a raw JSON object matching the schema. Do not use markdown code blocks.",
	})
	resp2, err2 := llm.Generate(ctx, retryMsgs, opts)
	if err2 == nil {
		cleanJSON2 := extractJSON(resp2.Content)
		if err := json.Unmarshal([]byte(cleanJSON2), &analysis); err == nil {
			return &analysis, nil
		}
	}

	// Fallback: build minimal analysis from the raw user prompt
	var userPrompt string
	for _, m := range history {
		if m.Role == provider.RoleUser {
			userPrompt = m.Content
		}
	}
	fallback := &TaskAnalysis{
		Status:         "ready",
		CoreIntent:     userPrompt,
		TechStack:      []string{},
		EstimatedLevel: 2,
		ExecutionPlan:  []string{"Implement the requested feature"},
	}
	return fallback, nil
}
