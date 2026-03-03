package graph

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStore_AddAndGetNode(t *testing.T) {
	s := NewStore()

	node := &Node{
		ID:     "file1",
		Labels: []string{"File"},
		Properties: map[string]string{
			"path": "/src/main.go",
		},
	}

	s.AddNode(node)

	retrieved, exists := s.GetNode("file1")
	if !exists {
		t.Fatalf("Expected node to exist")
	}

	if retrieved.ID != "file1" || retrieved.Properties["path"] != "/src/main.go" {
		t.Errorf("Retrieved node does not match original: %+v", retrieved)
	}
}

func TestStore_AddEdge(t *testing.T) {
	s := NewStore()

	s.AddNode(&Node{ID: "funcA", Labels: []string{"Function"}})
	s.AddNode(&Node{ID: "funcB", Labels: []string{"Function"}})

	// Target not exist
	err := s.AddEdge(&Edge{SrcID: "funcA", DstID: "funcX", Type: "Calls"})
	if err == nil {
		t.Fatalf("Expected error when adding edge to non-existent node")
	}

	// Valid edge
	err = s.AddEdge(&Edge{SrcID: "funcA", DstID: "funcB", Type: "Calls", Weight: 1.0})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	outEdges := s.GetOutEdges("funcA")
	if len(outEdges) != 1 || outEdges[0].DstID != "funcB" {
		t.Errorf("GetOutEdges failed: %+v", outEdges)
	}

	inEdges := s.GetInEdges("funcB")
	if len(inEdges) != 1 || inEdges[0].SrcID != "funcA" {
		t.Errorf("GetInEdges failed: %+v", inEdges)
	}
}

func TestStore_SaveAndLoad(t *testing.T) {
	s := NewStore()
	s.AddNode(&Node{ID: "1", Labels: []string{"A"}})
	s.AddNode(&Node{ID: "2", Labels: []string{"B"}})
	s.AddEdge(&Edge{SrcID: "1", DstID: "2", Type: "connects"})

	dir, err := os.MkdirTemp("", "graph-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "graph.json")
	if err := s.Save(path); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	s2 := NewStore()
	if err := s2.Load(path); err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	n1, exists := s2.GetNode("1")
	if !exists || n1.Labels[0] != "A" {
		t.Errorf("Failed to retrieve node 1 after load: %+v", n1)
	}

	outEdges := s2.GetOutEdges("1")
	if len(outEdges) != 1 || outEdges[0].DstID != "2" {
		t.Errorf("Failed to retrieve out edges after load")
	}
}

func TestStore_GetConnectedNodes(t *testing.T) {
	s := NewStore()
	s.AddNode(&Node{ID: "A"})
	s.AddNode(&Node{ID: "B"})
	s.AddNode(&Node{ID: "C"})
	s.AddNode(&Node{ID: "D"})

	s.AddEdge(&Edge{SrcID: "A", DstID: "B", Type: "conn"})
	s.AddEdge(&Edge{SrcID: "B", DstID: "C", Type: "conn"})
	s.AddEdge(&Edge{SrcID: "A", DstID: "D", Type: "conn"})

	// Depth 1 from A should find B, D (and A itself)
	nodes, edges := s.GetConnectedNodes("A", 1)
	if len(nodes) != 3 {
		t.Errorf("Expected 3 nodes at depth 1, got %d", len(nodes))
	}
	if len(edges) != 2 {
		t.Errorf("Expected 2 edges at depth 1, got %d", len(edges))
	}

	// Depth 2 from A should find all
	nodes2, _ := s.GetConnectedNodes("A", 2)
	if len(nodes2) != 4 {
		t.Errorf("Expected all 4 nodes at depth 2, got %d", len(nodes2))
	}

	// Make sure order/contents are matched (just basic check)
	foundC := false
	for _, n := range nodes2 {
		if n.ID == "C" {
			foundC = true
		}
	}
	if !foundC {
		t.Errorf("Expected to find node C at depth 2")
	}
}
