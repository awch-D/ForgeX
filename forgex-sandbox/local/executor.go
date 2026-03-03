// Package local implements the sandbox.Executor interface using os/exec
// with Unix-level process group isolation and memory limits.
package local

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"syscall"
	"time"

	sandbox "github.com/awch-D/ForgeX/forgex-sandbox"

	"github.com/awch-D/ForgeX/forgex-core/logger"
)

// Executor runs commands via os/exec with timeout, memory limits, and process group isolation.
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

// Run executes a shell command with:
//   - Timeout enforcement via context
//   - Process group isolation (Setpgid) to prevent orphan processes
//   - Memory limits via ulimit injection
func (e *Executor) Run(ctx context.Context, cmdStr string, opts sandbox.ExecOpts) (*sandbox.ExecResult, error) {
	timeout := e.defaultTimeout
	if opts.TimeoutSec > 0 {
		timeout = time.Duration(opts.TimeoutSec) * time.Second
	}

	memMB := e.defaultMemMB
	if opts.MemoryMB > 0 {
		memMB = opts.MemoryMB
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Inject ulimit memory limit before the actual command
	// ulimit -v sets virtual memory limit in KB
	wrappedCmd := fmt.Sprintf("ulimit -v %d 2>/dev/null; %s", memMB*1024, cmdStr)

	cmd := exec.Command("sh", "-c", wrappedCmd)
	if opts.WorkDir != "" {
		cmd.Dir = opts.WorkDir
	}

	// Create a new process group so we can kill the entire tree
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &outBuf

	logger.L().Debugw("🔒 Sandbox: executing command",
		"backend", e.Name(),
		"command", cmdStr,
		"timeout", timeout.String(),
		"memory_limit_mb", memMB,
	)

	if err := cmd.Start(); err != nil {
		return &sandbox.ExecResult{
			ExitCode: -1,
			Error:    fmt.Sprintf("failed to start command: %v", err),
		}, nil
	}

	// Monitor context cancellation in a goroutine to kill the entire process group
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-execCtx.Done():
		// Context timed out or was cancelled — kill the entire process group
		if cmd.Process != nil {
			pgid := cmd.Process.Pid
			logger.L().Debugw("🔒 Sandbox: killing process group on timeout",
				"pgid", pgid,
			)
			_ = syscall.Kill(-pgid, syscall.SIGKILL)
		}
		// Wait for the process to actually exit after being killed
		<-done
		return &sandbox.ExecResult{
			Output:   outBuf.String(),
			ExitCode: -1,
			TimedOut: true,
			Error:    fmt.Sprintf("command timed out after %s (process group killed)", timeout),
		}, nil

	case err := <-done:
		// Command finished normally (or with error)
		result := &sandbox.ExecResult{
			Output: outBuf.String(),
		}
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				result.ExitCode = exitErr.ExitCode()
				// Check if killed by signal (e.g. OOM or ulimit)
				if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
					sig := ws.Signal()
					result.Error = fmt.Sprintf("process killed by signal: %s", sig)
					if sig == syscall.SIGKILL {
						result.Error = "process killed (likely OOM or memory limit exceeded)"
					}
					return result, nil
				}
			} else {
				result.ExitCode = -1
			}
			result.Error = err.Error()
		} else {
			result.ExitCode = 0
		}
		return result, nil
	}
}
