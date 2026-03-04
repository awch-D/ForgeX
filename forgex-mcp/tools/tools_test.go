package tools

import (
	"testing"
)

func TestParseToolCalls_SingleObject(t *testing.T) {
	raw := `{"name":"write_file","args":{"path":"main.go","content":"package main"}}`
	calls, err := ParseToolCalls(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "write_file" {
		t.Errorf("expected name 'write_file', got '%s'", calls[0].Name)
	}
	if calls[0].Args["path"] != "main.go" {
		t.Errorf("expected arg path=main.go, got '%s'", calls[0].Args["path"])
	}
}

func TestParseToolCalls_Array(t *testing.T) {
	raw := `[{"name":"read_file","args":{"path":"a.go"}},{"name":"list_dir","args":{"path":"."}}]`
	calls, err := ParseToolCalls(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}
	if calls[0].Name != "read_file" || calls[1].Name != "list_dir" {
		t.Errorf("unexpected call names: %s, %s", calls[0].Name, calls[1].Name)
	}
}

func TestParseToolCalls_InvalidJSON(t *testing.T) {
	raw := `not json at all`
	_, err := ParseToolCalls(raw)
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestParseToolCalls_WhitespaceWrapped(t *testing.T) {
	raw := `  {"name":"run_command","args":{"command":"echo hi"}}  `
	calls, err := ParseToolCalls(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 1 || calls[0].Name != "run_command" {
		t.Errorf("unexpected result: %+v", calls)
	}
}

func TestRegistry_Execute_UnknownTool(t *testing.T) {
	r := NewRegistry(t.TempDir())
	result := r.Execute("nonexistent_tool", map[string]string{})
	if result.Success {
		t.Error("expected failure for unknown tool")
	}
	if result.Error == "" {
		t.Error("expected error message for unknown tool")
	}
}

func TestRegistry_ListTools(t *testing.T) {
	r := NewRegistry(t.TempDir())
	tools := r.ListTools()
	if len(tools) < 3 {
		t.Errorf("expected at least 3 builtin tools, got %d", len(tools))
	}

	// Verify built-in names
	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name] = true
	}
	for _, expected := range []string{"write_file", "read_file", "list_dir", "run_command"} {
		if !names[expected] {
			t.Errorf("expected builtin tool '%s' to be registered", expected)
		}
	}
}

func TestRegistry_ToolsForLLM(t *testing.T) {
	r := NewRegistry(t.TempDir())
	prompt := r.ToolsForLLM()
	if prompt == "" {
		t.Error("expected non-empty LLM tool prompt")
	}
	if len(prompt) < 100 {
		t.Errorf("expected substantial tool prompt, got %d chars", len(prompt))
	}
}

func TestRegistry_WriteAndReadFile(t *testing.T) {
	dir := t.TempDir()
	r := NewRegistry(dir)

	// Write
	result := r.Execute("write_file", map[string]string{
		"path":    "test.txt",
		"content": "hello world",
	})
	if !result.Success {
		t.Fatalf("write_file failed: %s", result.Error)
	}

	// Read
	result = r.Execute("read_file", map[string]string{
		"path": "test.txt",
	})
	if !result.Success {
		t.Fatalf("read_file failed: %s", result.Error)
	}
	if result.Output != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", result.Output)
	}
}

func TestRegistry_ListDir(t *testing.T) {
	dir := t.TempDir()
	r := NewRegistry(dir)

	// Create a file first
	r.Execute("write_file", map[string]string{
		"path":    "listme.go",
		"content": "package main",
	})

	result := r.Execute("list_dir", map[string]string{"path": "."})
	if !result.Success {
		t.Fatalf("list_dir failed: %s", result.Error)
	}
	if result.Output == "" {
		t.Error("expected non-empty list_dir output")
	}
}
