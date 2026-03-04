// Package tools provides built-in MCP-style tools for ForgeX agents.
// These are implemented as simple Go functions rather than external MCP servers,
// keeping the system local-first and dependency-free.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/awch-D/ForgeX/forgex-core/logger"
	"github.com/awch-D/ForgeX/forgex-governance/audit"
	"github.com/awch-D/ForgeX/forgex-governance/safety"
	sandbox "github.com/awch-D/ForgeX/forgex-sandbox"
)

// Tool represents a callable tool available to agents.
type Tool struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Parameters  map[string]ParamSpec `json:"parameters"`
}

// ParamSpec describes a tool parameter.
type ParamSpec struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// ToolResult is the outcome of a tool invocation.
type ToolResult struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

// Registry holds all available tools.
type Registry struct {
	tools            map[string]Tool
	execFn           map[string]func(args map[string]string) *ToolResult
	workDir          string
	autoApproveLevel safety.Level
	auditLogger      *audit.Logger
	executor         sandbox.Executor // sandboxed command execution backend
}

// NewRegistry creates a new tool registry rooted at the given working directory.
func NewRegistry(workDir string) *Registry {
	r := &Registry{
		tools:            make(map[string]Tool),
		execFn:           make(map[string]func(args map[string]string) *ToolResult),
		workDir:          workDir,
		autoApproveLevel: safety.Yellow, // default: auto-approve green + yellow
	}
	r.registerBuiltins()
	return r
}

// SetAutoApproveLevel configures the safety auto-approve threshold.
func (r *Registry) SetAutoApproveLevel(level safety.Level) {
	r.autoApproveLevel = level
}

// SetAuditLogger configures audit logging for tool invocations.
func (r *Registry) SetAuditLogger(l *audit.Logger) {
	r.auditLogger = l
}

// SetExecutor configures the sandboxed command executor.
// If not set, run_command falls back to plain exec.Command.
func (r *Registry) SetExecutor(e sandbox.Executor) {
	r.executor = e
	logger.L().Infow("🔒 Sandbox executor configured", "backend", e.Name())
}

// ListTools returns the schema of all registered tools (for LLM context).
func (r *Registry) ListTools() []Tool {
	var result []Tool
	for _, t := range r.tools {
		result = append(result, t)
	}
	return result
}

// ToolsForLLM generates the tool description text for the LLM system prompt.
func (r *Registry) ToolsForLLM() string {
	var sb strings.Builder
	sb.WriteString("Available tools:\n")
	for _, t := range r.tools {
		sb.WriteString(fmt.Sprintf("\n### %s\n%s\nParameters:\n", t.Name, t.Description))
		for name, spec := range t.Parameters {
			req := ""
			if spec.Required {
				req = " (required)"
			}
			sb.WriteString(fmt.Sprintf("  - %s (%s)%s: %s\n", name, spec.Type, req, spec.Description))
		}
	}
	return sb.String()
}

// Execute invokes a tool by name with safety classification and audit logging.
func (r *Registry) Execute(name string, args map[string]string) *ToolResult {
	fn, ok := r.execFn[name]
	if !ok {
		return &ToolResult{Success: false, Error: fmt.Sprintf("unknown tool: %s", name)}
	}

	// Safety classification
	level := safety.Classify(name, args)

	if level.IsBlocked() {
		logger.L().Warnw(safety.FormatDecision(name, level, false), "args", args)
		r.recordAudit(name, args, level, false, false, "operation blocked by safety policy")
		return &ToolResult{Success: false, Error: fmt.Sprintf("⚫ BLOCKED: %s is classified as %s and cannot be executed", name, level.String())}
	}

	logger.L().Infow("🔧 Tool invoked",
		"tool", name, "safety", level.String(), "args", args)

	result := fn(args)

	if result.Success {
		logger.L().Infow("✅ Tool succeeded", "tool", name, "output_len", len(result.Output))
	} else {
		logger.L().Warnw("❌ Tool failed", "tool", name, "error", result.Error)
	}

	// Audit logging
	r.recordAudit(name, args, level, true, result.Success, result.Error)

	return result
}

func (r *Registry) recordAudit(name string, args map[string]string, level safety.Level, approved, success bool, errStr string) {
	if r.auditLogger == nil {
		return
	}
	argsJSON, _ := json.Marshal(args)
	_ = r.auditLogger.Record(audit.Entry{
		ToolName:    name,
		Args:        string(argsJSON),
		SafetyLevel: level,
		Approved:    approved,
		Success:     success,
		Error:       errStr,
	})
}

func (r *Registry) registerBuiltins() {
	// ===== write_file =====
	r.register(Tool{
		Name:        "write_file",
		Description: "Create or overwrite a file with the given content. Parent directories will be created automatically.",
		Parameters: map[string]ParamSpec{
			"path":    {Type: "string", Description: "Relative file path from project root", Required: true},
			"content": {Type: "string", Description: "Full file content to write", Required: true},
		},
	}, func(args map[string]string) *ToolResult {
		path := filepath.Join(r.workDir, args["path"])
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return &ToolResult{Success: false, Error: err.Error()}
		}
		if err := os.WriteFile(path, []byte(args["content"]), 0644); err != nil {
			return &ToolResult{Success: false, Error: err.Error()}
		}
		return &ToolResult{Success: true, Output: fmt.Sprintf("File written: %s (%d bytes)", args["path"], len(args["content"]))}
	})

	// ===== read_file =====
	r.register(Tool{
		Name:        "read_file",
		Description: "Read the full content of a file.",
		Parameters: map[string]ParamSpec{
			"path": {Type: "string", Description: "Relative file path from project root", Required: true},
		},
	}, func(args map[string]string) *ToolResult {
		path := filepath.Join(r.workDir, args["path"])
		data, err := os.ReadFile(path)
		if err != nil {
			return &ToolResult{Success: false, Error: err.Error()}
		}
		return &ToolResult{Success: true, Output: string(data)}
	})

	// ===== list_dir =====
	r.register(Tool{
		Name:        "list_dir",
		Description: "List files and directories at the given path.",
		Parameters: map[string]ParamSpec{
			"path": {Type: "string", Description: "Relative directory path from project root (use '.' for root)", Required: true},
		},
	}, func(args map[string]string) *ToolResult {
		path := filepath.Join(r.workDir, args["path"])
		entries, err := os.ReadDir(path)
		if err != nil {
			return &ToolResult{Success: false, Error: err.Error()}
		}
		var sb strings.Builder
		for _, e := range entries {
			kind := "file"
			if e.IsDir() {
				kind = "dir"
			}
			sb.WriteString(fmt.Sprintf("[%s] %s\n", kind, e.Name()))
		}
		return &ToolResult{Success: true, Output: sb.String()}
	})

	// ===== run_command =====
	r.register(Tool{
		Name:        "run_command",
		Description: "Execute a shell command in the project directory. Use for running tests, installing deps, etc. Timeout: 30s.",
		Parameters: map[string]ParamSpec{
			"command": {Type: "string", Description: "Shell command to execute", Required: true},
		},
	}, func(args map[string]string) *ToolResult {
		// Prefer sandbox executor if configured
		if r.executor != nil {
			result, err := r.executor.Run(context.Background(), args["command"], sandbox.ExecOpts{
				WorkDir: r.workDir,
			})
			if err != nil {
				return &ToolResult{Success: false, Error: err.Error()}
			}
			if result.TimedOut {
				return &ToolResult{Success: false, Output: result.Output, Error: result.Error}
			}
			if result.ExitCode != 0 {
				return &ToolResult{Success: false, Output: result.Output, Error: result.Error}
			}
			return &ToolResult{Success: true, Output: result.Output}
		}

		// Fallback: plain exec.Command (no sandbox)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "sh", "-c", args["command"])
		cmd.Dir = r.workDir

		output, err := cmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			return &ToolResult{Success: false, Output: string(output), Error: "command timed out (30s)"}
		}
		if err != nil {
			return &ToolResult{Success: false, Output: string(output), Error: err.Error()}
		}
		return &ToolResult{Success: true, Output: string(output)}
	})
}

func (r *Registry) register(t Tool, fn func(args map[string]string) *ToolResult) {
	r.tools[t.Name] = t
	r.execFn[t.Name] = fn
}

// ParseToolCall extracts tool name and arguments from a JSON tool call block.
type ToolCall struct {
	Name string            `json:"name"`
	Args map[string]string `json:"args"`
}

// ParseToolCalls parses a JSON array of tool calls from LLM output.
func ParseToolCalls(raw string) ([]ToolCall, error) {
	clean := strings.TrimSpace(raw)
	// Try array first
	var calls []ToolCall
	if err := json.Unmarshal([]byte(clean), &calls); err == nil {
		return calls, nil
	}
	// Try single object
	var single ToolCall
	if err := json.Unmarshal([]byte(clean), &single); err == nil {
		return []ToolCall{single}, nil
	}
	return nil, fmt.Errorf("failed to parse tool calls from: %s", clean[:min(len(clean), 200)])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
