package archaeology

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/awch-D/ForgeX/forgex-cognition/graph"
)

func TestBuilder_ParseTSFile(t *testing.T) {
	tmpDir := t.TempDir()
	tsFile := filepath.Join(tmpDir, "app.ts")

	tsContent := `
export class UserService {
	constructor() {}
	// a method
	login() {}
}

export interface UserContext {
	id: string;
}

export async function fetchUser() {
	return {};
}

const checkAuth = async () => {
	// ...
}
`
	os.WriteFile(tsFile, []byte(tsContent), 0644)

	store := graph.NewStore()
	builder := NewBuilder(store)
	builder.parseTSFile(tsFile, tmpDir)

	// Verify Class
	cls, ok := store.GetNode("type:UserService")
	if !ok {
		t.Fatal("Expected node type:UserService")
	}
	if cls.Labels[0] != "Class" {
		t.Errorf("Expected label Class, got %v", cls.Labels)
	}

	// Verify Interface
	intf, ok := store.GetNode("type:UserContext")
	if !ok {
		t.Fatal("Expected node type:UserContext")
	}
	if intf.Labels[0] != "Interface" {
		t.Errorf("Expected label Interface, got %v", intf.Labels)
	}

	// Verify Functions
	if _, ok := store.GetNode("func:fetchUser"); !ok {
		t.Error("Expected func:fetchUser")
	}
	if _, ok := store.GetNode("func:checkAuth"); !ok {
		t.Error("Expected func:checkAuth")
	}
}
