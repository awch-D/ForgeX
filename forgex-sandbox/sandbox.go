// Package sandbox provides an abstraction for executing commands safely.
// It defines the Executor interface that can be implemented by different
// backends (exec.Command, Wasmtime, Docker, etc.).
package sandbox

import (
	"context"
)

// ExecResult represents the outcome of a sandboxed command execution.
type ExecResult struct {
	Output   string
	ExitCode int
	TimedOut bool
	Error    string
}

// ExecOpts configures the sandbox execution environment.
type ExecOpts struct {
	WorkDir    string // Working directory for the command
	TimeoutSec int    // Max execution time in seconds (0 = default)
	MemoryMB   int    // Memory limit in MB (0 = no limit)
}

// Executor is the interface for sandboxed command execution.
// Implementations include:
//   - exec.LocalExecutor: Uses os/exec with ulimit (default, lightweight)
//   - (future) wasm.WasmExecutor: Uses Wasmtime for full isolation
type Executor interface {
	// Run executes a shell command within the sandbox.
	Run(ctx context.Context, cmd string, opts ExecOpts) (*ExecResult, error)

	// Name returns the name of this executor backend.
	Name() string
}
