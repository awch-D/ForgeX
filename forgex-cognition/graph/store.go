package graph

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Node represents a vertex in the knowledge graph.
// Examples: File, Function, Struct
type Node struct {
	ID         string            `json:"id"`
	Labels     []string          `json:"labels"`
	Properties map[string]string `json:"properties"`
}

// Edge represents a directed relationship between two nodes.
// Examples: Calls, Imports, Defines, Uses
type Edge struct {
	SrcID      string            `json:"src_id"`
	DstID      string            `json:"dst_id"`
	Type       string            `json:"type"`
	Weight     float64           `json:"weight"`
	Properties map[string]string `json:"properties,omitempty"`
}

// Store holds the nodes and edges of the knowledge graph in memory.
type Store struct {
	mu    sync.RWMutex
	nodes map[string]*Node

	// outEdges: srcID -> list of edges starting from srcID
	outEdges map[string][]*Edge

	// inEdges: dstID -> list of edges ending at dstID
	inEdges map[string][]*Edge
}

// NewStore creates a new graph store.
func NewStore() *Store {
	return &Store{
		nodes:    make(map[string]*Node),
		outEdges: make(map[string][]*Edge),
		inEdges:  make(map[string][]*Edge),
	}
}

// AddNode adds or updates a node in the graph.
func (s *Store) AddNode(node *Node) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if node.Properties == nil {
		node.Properties = make(map[string]string)
	}
	s.nodes[node.ID] = node
}

// GetNode retrieves a node by its ID.
func (s *Store) GetNode(id string) (*Node, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	node, exists := s.nodes[id]
	return node, exists
}

// AddEdge adds a directed edge from SrcID to DstID.
// Returns an error if either node doesn't exist.
func (s *Store) AddEdge(edge *Edge) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.nodes[edge.SrcID]; !ok {
		return fmt.Errorf("source node '%s' does not exist", edge.SrcID)
	}
	if _, ok := s.nodes[edge.DstID]; !ok {
		return fmt.Errorf("destination node '%s' does not exist", edge.DstID)
	}

	// De-duplicate: If exact same edge type exists between same nodes, we can update or ignore.
	// For simplicity, we just append here. If stricter logic is needed, we'd check existence.
	for i, existing := range s.outEdges[edge.SrcID] {
		if existing.DstID == edge.DstID && existing.Type == edge.Type {
			// Update properties/weight of existing edge
			s.outEdges[edge.SrcID][i] = edge

			// Update inEdges too
			for j, inE := range s.inEdges[edge.DstID] {
				if inE.SrcID == edge.SrcID && inE.Type == edge.Type {
					s.inEdges[edge.DstID][j] = edge
					return nil
				}
			}
			return nil
		}
	}

	s.outEdges[edge.SrcID] = append(s.outEdges[edge.SrcID], edge)
	s.inEdges[edge.DstID] = append(s.inEdges[edge.DstID], edge)
	return nil
}

// GetOutEdges returns all edges starting from the given node ID.
func (s *Store) GetOutEdges(srcID string) []*Edge {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy of the slice to prevent concurrent mutation issues
	edges := make([]*Edge, len(s.outEdges[srcID]))
	copy(edges, s.outEdges[srcID])
	return edges
}

// GetInEdges returns all edges ending at the given node ID.
func (s *Store) GetInEdges(dstID string) []*Edge {
	s.mu.RLock()
	defer s.mu.RUnlock()

	edges := make([]*Edge, len(s.inEdges[dstID]))
	copy(edges, s.inEdges[dstID])
	return edges
}

// SearchNodesByLabel returns all nodes having a specific label.
func (s *Store) SearchNodesByLabel(label string) []*Node {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Node
	for _, n := range s.nodes {
		for _, l := range n.Labels {
			if l == label {
				result = append(result, n)
				break
			}
		}
	}
	return result
}

// Snapshot represents a serializable version of the graph.
type Snapshot struct {
	Nodes []*Node `json:"nodes"`
	Edges []*Edge `json:"edges"`
}

// Save writes the entire graph to a JSON file.
func (s *Store) Save(filepath string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snap := Snapshot{
		Nodes: make([]*Node, 0, len(s.nodes)),
		Edges: make([]*Edge, 0),
	}

	for _, n := range s.nodes {
		snap.Nodes = append(snap.Nodes, n)
	}

	for _, edges := range s.outEdges {
		snap.Edges = append(snap.Edges, edges...)
	}

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath, data, 0644)
}

// Load reads the graph from a JSON file, replacing the current state.
func (s *Store) Load(filepath string) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // OK if file doesn't exist yet
		}
		return err
	}

	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.nodes = make(map[string]*Node)
	s.outEdges = make(map[string][]*Edge)
	s.inEdges = make(map[string][]*Edge)

	// Re-add nodes
	for _, n := range snap.Nodes {
		if n.Properties == nil {
			n.Properties = make(map[string]string)
		}
		s.nodes[n.ID] = n
	}

	// Re-add edges (bypassing validation for speed, assuming valid JSON)
	for _, e := range snap.Edges {
		s.outEdges[e.SrcID] = append(s.outEdges[e.SrcID], e)
		s.inEdges[e.DstID] = append(s.inEdges[e.DstID], e)
	}

	return nil
}

// GetConnectedNodes returns an adjacency map of a node up to a specified depth.
func (s *Store) GetConnectedNodes(startID string, maxDepth int) ([]*Node, []*Edge) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.nodes[startID]; !ok {
		return nil, nil
	}

	visitedNodes := make(map[string]bool)
	visitedEdges := make(map[string]bool) // naive sig: src_dst_type
	var nodes []*Node
	var edges []*Edge

	type queueItem struct {
		id    string
		depth int
	}

	queue := []queueItem{{id: startID, depth: 0}}
	visitedNodes[startID] = true
	nodes = append(nodes, s.nodes[startID])

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if curr.depth >= maxDepth {
			continue
		}

		// Explore out edges
		for _, edge := range s.outEdges[curr.id] {
			edgeSig := edge.SrcID + "_" + edge.DstID + "_" + edge.Type
			if !visitedEdges[edgeSig] {
				visitedEdges[edgeSig] = true
				edges = append(edges, edge)
			}

			if !visitedNodes[edge.DstID] {
				visitedNodes[edge.DstID] = true
				nodes = append(nodes, s.nodes[edge.DstID])
				queue = append(queue, queueItem{id: edge.DstID, depth: curr.depth + 1})
			}
		}

		// Optionally explore in edges (if graph is meant to be navigated bidirectionally)
		for _, edge := range s.inEdges[curr.id] {
			edgeSig := edge.SrcID + "_" + edge.DstID + "_" + edge.Type
			if !visitedEdges[edgeSig] {
				visitedEdges[edgeSig] = true
				edges = append(edges, edge)
			}

			if !visitedNodes[edge.SrcID] {
				visitedNodes[edge.SrcID] = true
				nodes = append(nodes, s.nodes[edge.SrcID])
				queue = append(queue, queueItem{id: edge.SrcID, depth: curr.depth + 1})
			}
		}
	}

	return nodes, edges
}

// BuildSummary generates a high-level markdown context of the project architecture
// for LLM consumption.
func (s *Store) BuildSummary() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Node
	for _, n := range s.nodes {
		for _, l := range n.Labels {
			if l == "File" {
				result = append(result, n)
				break
			}
		}
	}
	if len(result) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n--- Codebase Architecture Context ---\n")
	sb.WriteString("The following files and their primary definitions are available in the project graph:\n")

	for _, f := range result {
		path := f.Properties["path"]
		sb.WriteString(fmt.Sprintf("\nFile: %s\n", path))

		var definedNodes []string
		for _, e := range s.inEdges[f.ID] {
			if e.Type == "DefinedIn" {
				if node, ok := s.nodes[e.SrcID]; ok {
					name := node.Properties["name"]
					if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' {
						definedNodes = append(definedNodes, "- "+node.ID)
					}
				}
			}
		}

		if len(definedNodes) > 0 {
			for _, dn := range definedNodes {
				sb.WriteString(dn + "\n")
			}
		} else {
			sb.WriteString("- (no exported struct/func detected)\n")
		}

		// Inject Git Archaeology metadata
		if count, ok := f.Properties["commit_count"]; ok {
			sb.WriteString(fmt.Sprintf("  [🚧 Git 热点] 近期被频繁修改 %s 次\n", count))
		}

		var related []string
		for _, e := range s.outEdges[f.ID] {
			if e.Type == "CoModifiedWith" {
				if tgt, ok := s.nodes[e.DstID]; ok {
					// Use path relative or just base name
					related = append(related, fmt.Sprintf("%s (共现 %v 次)", tgt.Properties["path"], e.Weight))
				}
			}
		}
		if len(related) > 0 {
			sb.WriteString(fmt.Sprintf("  [🔗 隐性耦合] 经常同此文件一起修改的有: %s\n", strings.Join(related, ", ")))
		}
	}
	sb.WriteString("-------------------------------------\n")
	return sb.String()
}
