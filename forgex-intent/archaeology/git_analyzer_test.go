package archaeology

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/awch-D/ForgeX/forgex-cognition/graph"
)

func TestGitAnalyzer_ValidRepo(t *testing.T) {
	// Setup a temporary git repo to test against real git binary
	tmpDir := t.TempDir()

	// Init git repo
	exec.Command("git", "-C", tmpDir, "init").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.name", "test").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run()

	// Commit 1: Modify file1.go and file2.go
	os.WriteFile(filepath.Join(tmpDir, "file1.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.go"), []byte("package main"), 0644)
	exec.Command("git", "-C", tmpDir, "add", ".").Run()
	exec.Command("git", "-C", tmpDir, "commit", "-m", "Commit 1").Run()

	// Commit 2: Modify file2.go and file3.go
	os.WriteFile(filepath.Join(tmpDir, "file2.go"), []byte("package main\n// update"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file3.go"), []byte("package main"), 0644)
	exec.Command("git", "-C", tmpDir, "add", ".").Run()
	exec.Command("git", "-C", tmpDir, "commit", "-m", "Commit 2").Run()

	// Commit 3: Modify file1.go and file2.go AGAIN (this crosses the threshold of >= 2 co-modifications)
	os.WriteFile(filepath.Join(tmpDir, "file1.go"), []byte("package main\n// update"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.go"), []byte("package main\n// update 2"), 0644)
	exec.Command("git", "-C", tmpDir, "add", ".").Run()
	exec.Command("git", "-C", tmpDir, "commit", "-m", "Commit 3").Run()

	// Run GitAnalyzer
	store := graph.NewStore()
	analyzer := NewGitAnalyzer(store)
	if err := analyzer.Analyze(context.Background(), tmpDir, 10); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Verification
	// file2.go should have commit_count = 3
	f2Node, ok := store.GetNode("file:file2.go")
	if !ok {
		t.Fatal("file2.go node not found")
	}
	if f2Node.Properties["commit_count"] != "3" {
		t.Errorf("expected file2.go commit_count to be 3, got %s", f2Node.Properties["commit_count"])
	}

	// Co-modification test: file1 and file2 should have edges (they were co-modified in Commit 1 and Commit 3)
	outEdges := store.GetOutEdges("file:file1.go")
	foundCoMod := false
	for _, e := range outEdges {
		if e.Type == "CoModifiedWith" && e.DstID == "file:file2.go" {
			foundCoMod = true
			if e.Weight != 2.0 {
				t.Errorf("expected CoModifiedWith weight to be 2.0, got %f", e.Weight)
			}
			break
		}
	}
	if !foundCoMod {
		t.Error("expected CoModifiedWith edge between file1.go and file2.go")
	}
}

func TestGitAnalyzer_NotARepo(t *testing.T) {
	tmpDir := t.TempDir() // empty dir, not a git repo
	store := graph.NewStore()
	analyzer := NewGitAnalyzer(store)

	// Should not error out, just skip gracefully
	if err := analyzer.Analyze(context.Background(), tmpDir, 10); err != nil {
		t.Errorf("expected no error for non-git repo, got %v", err)
	}

	// Store should be empty
	if len(store.SearchNodesByLabel("File")) > 0 {
		t.Error("expected store to be empty")
	}
}
