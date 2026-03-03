package archaeology

import (
	"bufio"
	"os"
	"regexp"

	"github.com/awch-D/ForgeX/forgex-cognition/graph"
	"github.com/awch-D/ForgeX/forgex-core/logger"
)

var (
	tsClassRegex     = regexp.MustCompile(`(?:export\s+)?(?:abstract\s+)?class\s+([a-zA-Z0-9_]+)`)
	tsInterfaceRegex = regexp.MustCompile(`(?:export\s+)?(?:interface|type)\s+([a-zA-Z0-9_]+)`)
	tsFuncRegex      = regexp.MustCompile(`(?:export\s+)?(?:async\s+)?function\s+([a-zA-Z0-9_]+)`)
	tsArrowRegex     = regexp.MustCompile(`(?:export\s+)?(?:const|let|var)\s+([a-zA-Z0-9_]+)\s*=\s*(?:async\s*)?(?:\([^)]*\)|[a-zA-Z0-9_]+)\s*=>`)
)

func (b *Builder) parseTSFile(filePath, rootDir string) {
	relPath := b.getRelPath(filePath, rootDir)

	fileNode := &graph.Node{
		ID:     "file:" + relPath,
		Labels: []string{"File"},
		Properties: map[string]string{
			"path": relPath,
		},
	}
	b.store.AddNode(fileNode)

	file, err := os.Open(filePath)
	if err != nil {
		logger.L().Debugw("Failed to open TS/JS file", "file", relPath, "error", err)
		return
	}
	defer file.Close()

	if info, err := file.Stat(); err == nil && info.Size() > 1*1024*1024 {
		// skip files > 1MB
		return
	}

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		if match := tsClassRegex.FindStringSubmatch(line); len(match) > 1 {
			className := match[1]
			nodeID := "type:" + className
			b.store.AddNode(&graph.Node{
				ID:     nodeID,
				Labels: []string{"Class"},
				Properties: map[string]string{
					"name": className,
				},
			})
			_ = b.store.AddEdge(&graph.Edge{
				SrcID: nodeID,
				DstID: fileNode.ID,
				Type:  "DefinedIn",
			})
			continue
		}

		if match := tsInterfaceRegex.FindStringSubmatch(line); len(match) > 1 {
			intfName := match[1]
			nodeID := "type:" + intfName
			b.store.AddNode(&graph.Node{
				ID:     nodeID,
				Labels: []string{"Interface"},
				Properties: map[string]string{
					"name": intfName,
				},
			})
			_ = b.store.AddEdge(&graph.Edge{
				SrcID: nodeID,
				DstID: fileNode.ID,
				Type:  "DefinedIn",
			})
			continue
		}

		funcName := ""
		if match := tsFuncRegex.FindStringSubmatch(line); len(match) > 1 {
			funcName = match[1]
		} else if match := tsArrowRegex.FindStringSubmatch(line); len(match) > 1 {
			funcName = match[1]
		}

		if funcName != "" {
			nodeID := "func:" + funcName
			b.store.AddNode(&graph.Node{
				ID:     nodeID,
				Labels: []string{"Function"},
				Properties: map[string]string{
					"name": funcName,
				},
			})
			_ = b.store.AddEdge(&graph.Edge{
				SrcID: nodeID,
				DstID: fileNode.ID,
				Type:  "DefinedIn",
			})
		}
	}
}
