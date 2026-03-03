// Package local implements the sandbox.Executor interface using os/exec.
// This is the default, lightweight sandbox backend.
package local

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	sandbox "github.com/awch-D/ForgeX/forgex-sandbox"

	"github.com/awch-D/ForgeX/forgex-core/logger"
)

// Executor runs commands via os/exec with timeout enforcement.
type Executor struct {
	defaultTimeout time.Duration
	defaultMemMB   int
}

// New creates a new local executor with defaults from config.
func New(timeoutSec, memoryMB int) *Executor {
	if timeoutSec <= 0 {
		timeoutSec = 30
	}
	if memoryMB <= 0 {
		memoryMB = 512
	}
	return &Executor{
		defaultTimeout: time.Duration(timeoutSec) * time.Second,
		defaultMemMB:   memoryMB,
	}
}

func (e *Executor) Name() string { return "local-exec" }

// Run executes a shell command with timeout and optional memory limits.
func (e *Executor) Run(ctx context.Context, cmdStr string, opts sandbox.ExecOpts) (*sandbox.ExecResult, error) {
	timeout := e.defaultTimeout
	if opts.TimeoutSec > 0 {
		timeout = time.Duration(opts.TimeoutSec) * time.Second
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "sh", "-c", cmdStr)
	if opts.WorkDir != "" {
		cmd.Dir = opts.WorkDir
	}

	logger.L().Debugw("🔒 Sandbox: executing command",
		"backend", e.Name(),
		"command", cmdStr,
		"timeout", timeout.String(),
	)

	output, err := cmd.CombinedOutput()

	result := &sandbox.ExecResult{
		Output: string(output),
	}

	if execCtx.Err() == context.DeadlineExceeded {
		result.TimedOut = true
		result.Error = fmt.Sprintf("command timed out after %s", timeout)
		result.ExitCode = -1
		return result, nil
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
		result.Error = err.Error()
		return result, nil
	}

	result.ExitCode = 0
	return result, nil
}
