package archaeology

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/awch-D/ForgeX/forgex-cognition/graph"
	"github.com/awch-D/ForgeX/forgex-core/logger"
)

// GitAnalyzer extracts implicit relationships and hot files from Git history.
type GitAnalyzer struct {
	store *graph.Store
}

// NewGitAnalyzer creates a new GitAnalyzer.
func NewGitAnalyzer(store *graph.Store) *GitAnalyzer {
	return &GitAnalyzer{store: store}
}

// Analyze reads the git log and injects CommitCount properties and CoModifiedWith edges into the graph.
func (a *GitAnalyzer) Analyze(ctx context.Context, dir string, maxCommits int) error {
	logger.L().Infof("🔍 正在扫描 Git 历史考古 (Top %d commits): %s", maxCommits, dir)

	// 1. Check if it's a git repo
	if err := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "--is-inside-work-tree").Run(); err != nil {
		logger.L().Debugw("Directory is not a git repository, skipping git analysis", "dir", dir)
		return nil
	}

	// 2. Fetch git log: list of modified files grouped by commit
	// "commit" followed by raw file paths mutated in that commit
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "log", "--name-only", "--pretty=format:commit", "--diff-filter=MAD", "-n", fmt.Sprint(maxCommits))
	out, err := cmd.Output()
	if err != nil {
		logger.L().Warnw("Failed to read git log", "error", err)
		return err
	}

	raw := string(out)
	commits := strings.Split(raw, "commit\n")

	fileCounts := make(map[string]int)
	coMatrix := make(map[string]map[string]int)

	for _, c := range commits {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}

		var files []string
		for _, line := range strings.Split(c, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				files = append(files, line)
			}
		}

		// Count individual file modification frequencies (hot spots)
		for _, f := range files {
			fileCounts[f]++
		}

		// Calculate co-occurrence (implicit coupling)
		for i := 0; i < len(files); i++ {
			for j := i + 1; j < len(files); j++ {
				f1, f2 := files[i], files[j]
				if f1 > f2 {
					f1, f2 = f2, f1 // Normalize pair order (A, B) where A < B
				}
				if coMatrix[f1] == nil {
					coMatrix[f1] = make(map[string]int)
				}
				coMatrix[f1][f2]++
			}
		}
	}

	// 3. Inject findings into the graph store

	// Add commit counts to existing or new File nodes
	for f, count := range fileCounts {
		// Only consider actual source files to avoid noisy config/markdown files in graph,
		// but since it's global analysis, we'll keep it broad or just limit to .go/.js/etc later.
		// For now, track all files modified in the repo.
		nodeID := "file:" + f
		node, exists := a.store.GetNode(nodeID)
		if !exists {
			node = &graph.Node{
				ID:     nodeID,
				Labels: []string{"File"},
				Properties: map[string]string{
					"path": f,
				},
			}
		}
		node.Properties["commit_count"] = fmt.Sprint(count)
		a.store.AddNode(node)
	}

	// Add 'CoModifiedWith' edges for files that change together frequently
	for f1, targets := range coMatrix {
		for f2, coCount := range targets {
			// Threshold: files must have been modified together at least twice to be considered related
			if coCount >= 2 {
				weight := float64(coCount)
				_ = a.store.AddEdge(&graph.Edge{
					SrcID:  "file:" + f1,
					DstID:  "file:" + f2,
					Type:   "CoModifiedWith",
					Weight: weight,
				})
				// Undirected conceptual relationship, but graph is directed, so add reverse edge
				_ = a.store.AddEdge(&graph.Edge{
					SrcID:  "file:" + f2,
					DstID:  "file:" + f1,
					Type:   "CoModifiedWith",
					Weight: weight,
				})
			}
		}
	}

	logger.L().Infow("Git archaeology complete",
		"hot_files", len(fileCounts),
		"coupled_pairs", len(coMatrix))

	return nil
}
