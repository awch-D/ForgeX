package archaeology

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/awch-D/ForgeX/forgex-cognition/graph"
)

func TestBuilder_Build(t *testing.T) {
	// Create a temporary directory with some Go code to test the AST builder.
	dir, err := os.MkdirTemp("", "archaeology-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Create test package a
	pkgADir := filepath.Join(dir, "pkgA")
	os.MkdirAll(pkgADir, 0755)

	fileACode := `package pkgA
	
	type MyStruct struct {}
	
	func (s *MyStruct) DoSomething() {
		// internal activity
	}

	func RegularFunc() {
		s := &MyStruct{}
		s.DoSomething()
	}
	`
	os.WriteFile(filepath.Join(pkgADir, "a.go"), []byte(fileACode), 0644)

	// Create test package b that calls a
	pkgBDir := filepath.Join(dir, "pkgB")
	os.MkdirAll(pkgBDir, 0755)

	fileBCode := `package pkgB
	import "pkgA"

	func MainFunc() {
		pkgA.RegularFunc()
	}
	`
	os.WriteFile(filepath.Join(pkgBDir, "b.go"), []byte(fileBCode), 0644)

	store := graph.NewStore()
	builder := NewBuilder(store)

	if err := builder.Build(dir); err != nil {
		t.Fatalf("Failed to build graph: %v", err)
	}

	// Verify the graph structure
	// Nodes:
	// - file: pkgA/a.go, pkgB/b.go
	// - type: MyStruct
	// - func: MyStruct.DoSomething, RegularFunc, MainFunc, pkgA.RegularFunc (external ghost)

	n, ok := store.GetNode("type:MyStruct")
	if !ok {
		t.Errorf("Expected node type:MyStruct to exist")
	} else if n.Labels[0] != "Struct" {
		t.Errorf("Expected type:MyStruct label to be Struct, got %v", n.Labels)
	}

	n, ok = store.GetNode("func:MyStruct.DoSomething")
	if !ok {
		t.Errorf("Expected func:MyStruct.DoSomething to exist")
	}

	// Verify edges
	// RegularFunc should Call MyStruct.DoSomething
	outEdges := store.GetOutEdges("func:RegularFunc")
	foundCall := false
	for _, e := range outEdges {
		if e.Type == "Calls" && e.DstID == "func:s.DoSomething" {
			// Note: Our simple AST parser currently captures "s.DoSomething()" as target "s.DoSomething"
			// The receiver type resolution would require a full type-checker.
			// For lightweight AST, "s.DoSomething" is the naive target, which may resolve to "func:s.DoSomething" ghost.
			foundCall = true
		}
	}
	if !foundCall {
		t.Errorf("Expected RegularFunc -> Calls -> DoSomething edge. Edges: %+v", outEdges)
	}

	// MainFunc should call pkgA.RegularFunc
	outEdgesB := store.GetOutEdges("func:MainFunc")
	foundCallB := false
	for _, e := range outEdgesB {
		if e.Type == "Calls" && e.DstID == "func:pkgA.RegularFunc" {
			foundCallB = true
		}
	}
	if !foundCallB {
		t.Errorf("Expected MainFunc -> Calls -> pkgA.RegularFunc edge. Edges: %+v", outEdgesB)
	}
}
