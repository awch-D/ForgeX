package audit_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/awch-D/ForgeX/forgex-governance/audit"
	"github.com/awch-D/ForgeX/forgex-governance/safety"
)

func setupTestLogger(t *testing.T) (*audit.Logger, func()) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test_audit.db")
	l, err := audit.NewLogger(dbPath)
	if err != nil {
		t.Fatalf("failed to create audit logger: %v", err)
	}
	return l, func() {
		l.Close()
		os.Remove(dbPath)
	}
}

func TestAuditLogger_Record(t *testing.T) {
	logger, cleanup := setupTestLogger(t)
	defer cleanup()

	err := logger.Record(audit.Entry{
		ToolName:    "write_file",
		Args:        `{"path":"main.go"}`,
		SafetyLevel: safety.Yellow,
		Approved:    true,
		Success:     true,
	})
	if err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	count, err := logger.Count()
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 entry, got %d", count)
	}
}

func TestAuditLogger_Recent(t *testing.T) {
	logger, cleanup := setupTestLogger(t)
	defer cleanup()

	// Record 3 entries
	for i := 0; i < 3; i++ {
		logger.Record(audit.Entry{
			ToolName:    "read_file",
			Args:        `{"path":"test.go"}`,
			SafetyLevel: safety.Green,
			Approved:    true,
			Success:     true,
		})
	}

	entries, err := logger.Recent(2)
	if err != nil {
		t.Fatalf("Recent failed: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 recent entries, got %d", len(entries))
	}
}

func TestAuditLogger_Stats(t *testing.T) {
	logger, cleanup := setupTestLogger(t)
	defer cleanup()

	// Record mixed entries
	logger.Record(audit.Entry{
		ToolName: "read_file", SafetyLevel: safety.Green, Approved: true, Success: true,
	})
	logger.Record(audit.Entry{
		ToolName: "write_file", SafetyLevel: safety.Yellow, Approved: true, Success: true,
	})
	logger.Record(audit.Entry{
		ToolName: "run_command", SafetyLevel: safety.Black, Approved: false, Success: false,
		Error: "blocked",
	})

	stats, err := logger.Stats()
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}
	if stats.Total != 3 {
		t.Errorf("expected 3 total, got %d", stats.Total)
	}
	if stats.Approved != 2 {
		t.Errorf("expected 2 approved, got %d", stats.Approved)
	}
	if stats.Blocked != 1 {
		t.Errorf("expected 1 blocked, got %d", stats.Blocked)
	}
	if stats.ByLevel[safety.Green] != 1 {
		t.Errorf("expected 1 green, got %d", stats.ByLevel[safety.Green])
	}
}
