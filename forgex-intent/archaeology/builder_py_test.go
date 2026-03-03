package archaeology

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/awch-D/ForgeX/forgex-cognition/graph"
)

func TestBuilder_ParsePythonFile(t *testing.T) {
	tmpDir := t.TempDir()
	pyFile := filepath.Join(tmpDir, "main.py")

	pyContent := `
import os

def query_db():
    pass

class DataManager(BaseManager):
    def fetch_data(self):
        query_db()
        pass

    def save_data(self, obj):
        pass

def global_cleanup():
    pass
`
	os.WriteFile(pyFile, []byte(pyContent), 0644)

	store := graph.NewStore()
	builder := NewBuilder(store)
	builder.parsePythonFile(pyFile, tmpDir)

	// Verify global functions
	if _, ok := store.GetNode("func:query_db"); !ok {
		t.Error("Expected func:query_db")
	}
	if _, ok := store.GetNode("func:global_cleanup"); !ok {
		t.Error("Expected func:global_cleanup")
	}

	// Verify Class
	if _, ok := store.GetNode("type:DataManager"); !ok {
		t.Error("Expected type:DataManager")
	}

	// Verify Class Methods
	f1, ok := store.GetNode("func:DataManager.fetch_data")
	if !ok {
		t.Fatal("Expected func:DataManager.fetch_data")
	}
	if f1.Labels[0] != "Function" {
		t.Errorf("Expected label Function, got %v", f1.Labels)
	}

	if _, ok := store.GetNode("func:DataManager.save_data"); !ok {
		t.Error("Expected func:DataManager.save_data")
	}

	// Verify MethodOf edge
	edges := store.GetOutEdges("func:DataManager.fetch_data")
	foundMethodEdge := false
	for _, e := range edges {
		if e.Type == "MethodOf" && e.DstID == "type:DataManager" {
			foundMethodEdge = true
			break
		}
	}
	if !foundMethodEdge {
		t.Error("Expected MethodOf edge from fetch_data to DataManager")
	}
}
