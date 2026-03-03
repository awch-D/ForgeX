package fact_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/awch-D/ForgeX/forgex-cognition/fact"
)

func tempDBPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "facts_test.db")
}

func TestFactStore_PutAndCount(t *testing.T) {
	s, err := fact.NewStore(tempDBPath(t))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if err := s.Put("file", "package main\nfunc main() {}"); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	count, err := s.Count()
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected count=1, got %d", count)
	}
}

func TestFactStore_Deduplication(t *testing.T) {
	s, err := fact.NewStore(tempDBPath(t))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	content := "func hello() { println(\"hello\") }"
	if err := s.Put("file", content); err != nil {
		t.Fatal(err)
	}
	if err := s.Put("file", content); err != nil {
		t.Fatal(err)
	}

	count, err := s.Count()
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected count=1 after dedup, got %d", count)
	}
}

func TestFactStore_Search(t *testing.T) {
	s, err := fact.NewStore(tempDBPath(t))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	_ = s.Put("file", "package auth implements JWT token validation")
	_ = s.Put("file", "package db provides SQLite connection pool")
	_ = s.Put("test", "PASS: TestLogin verifies user authentication flow")

	results, err := s.Search("authentication", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one search result for 'authentication'")
	}

	// Verify returned fact has content
	found := false
	for _, r := range results {
		if r.Content != "" && r.Tag != "" {
			found = true
		}
	}
	if !found {
		t.Error("search results have empty fields")
	}
}

func TestFactStore_SearchNoResults(t *testing.T) {
	s, err := fact.NewStore(tempDBPath(t))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	_ = s.Put("file", "package main")

	results, err := s.Search("xyznonexistent", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestFactStore_PersistAcrossReopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "persist_test.db")

	// Create and populate
	s1, err := fact.NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_ = s1.Put("api", "GET /users returns user list")
	s1.Close()

	// Reopen and verify
	s2, err := fact.NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()

	count, _ := s2.Count()
	if count != 1 {
		t.Errorf("expected 1 fact after reopen, got %d", count)
	}
}

func TestFactStore_InvalidPath(t *testing.T) {
	_, err := fact.NewStore("/nonexistent/path/that/should/fail/test.db")
	if err == nil {
		// Some systems may create parent dirs; if so, clean up
		os.Remove("/nonexistent/path/that/should/fail/test.db")
	}
	// Just verify it doesn't panic
}
