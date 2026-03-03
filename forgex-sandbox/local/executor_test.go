package local

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	sandbox "github.com/awch-D/ForgeX/forgex-sandbox"
)

func TestExecutor_BasicCommand(t *testing.T) {
	e := New(10, 512)
	result, err := e.Run(context.Background(), "echo hello", sandbox.ExecOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "hello") {
		t.Errorf("expected output to contain 'hello', got: %s", result.Output)
	}
}

func TestExecutor_Timeout(t *testing.T) {
	e := New(1, 512)
	result, err := e.Run(context.Background(), "sleep 30", sandbox.ExecOpts{
		TimeoutSec: 1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.TimedOut {
		t.Error("expected command to be timed out")
	}
	if result.ExitCode != -1 {
		t.Errorf("expected exit code -1 for timeout, got %d", result.ExitCode)
	}
}

func TestExecutor_ProcessGroupKill(t *testing.T) {
	// This test verifies that spawned child processes are killed
	// when the parent is killed (process group isolation).
	e := New(10, 512)

	// Start a command that spawns a background child process
	// The marker string helps us find the process later
	marker := "forgex_sandbox_test_marker_12345"
	cmdStr := "sh -c 'sleep 300 & echo " + marker + "'"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := e.Run(ctx, cmdStr, sandbox.ExecOpts{
		TimeoutSec: 2,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The command should have timed out
	if !result.TimedOut {
		t.Log("Command completed before timeout (ok for this test)")
	}

	// Give OS a moment to clean up
	time.Sleep(500 * time.Millisecond)

	// Verify no orphan process remains with our marker
	out, _ := exec.Command("sh", "-c", "ps aux | grep "+marker+" | grep -v grep").CombinedOutput()
	if strings.Contains(string(out), marker) {
		t.Errorf("orphan process still running after process group kill: %s", string(out))
	}
}

func TestExecutor_NonZeroExitCode(t *testing.T) {
	e := New(10, 512)
	result, err := e.Run(context.Background(), "exit 42", sandbox.ExecOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 42 {
		t.Errorf("expected exit code 42, got %d", result.ExitCode)
	}
}

func TestExecutor_MemoryLimitInjection(t *testing.T) {
	// Test that ulimit is injected (we can't easily test it triggers,
	// but we can verify the command executes correctly with ulimit prefix)
	e := New(10, 256) // 256MB limit
	result, err := e.Run(context.Background(), "echo mem_test_ok", sandbox.ExecOpts{
		MemoryMB: 256,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d (error: %s)", result.ExitCode, result.Error)
	}
	if !strings.Contains(result.Output, "mem_test_ok") {
		t.Errorf("expected output 'mem_test_ok', got: %s", result.Output)
	}
}
