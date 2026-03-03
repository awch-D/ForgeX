package archaeology

import (
	"bufio"
	"os"
	"regexp"
	"strings"

	"github.com/awch-D/ForgeX/forgex-cognition/graph"
	"github.com/awch-D/ForgeX/forgex-core/logger"
)

var (
	// class Foo { ... }
	pyClassRegex = regexp.MustCompile(`^\s*class\s+([a-zA-Z0-9_]+)[\(:]`)
	// def func_name(args):
	pyFuncRegex = regexp.MustCompile(`^\s*def\s+([a-zA-Z0-9_]+)\s*\(`)
)

func (b *Builder) parsePythonFile(filePath, rootDir string) {
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
		logger.L().Debugw("Failed to open python file", "file", relPath, "error", err)
		return
	}
	defer file.Close()

	if info, err := file.Stat(); err == nil && info.Size() > 1*1024*1024 {
		// skip files > 1MB
		return
	}

	scanner := bufio.NewScanner(file)
	var currentClass string
	var currentClassIndent int = -1

	for scanner.Scan() {
		line := scanner.Text()

		// Determine indentation (spaces)
		indent := len(line) - len(strings.TrimLeft(line, " \t"))

		// If we un-indent past the current class, we are no longer in the class
		if currentClassIndent != -1 && indent <= currentClassIndent && strings.TrimSpace(line) != "" {
			// Actually only check this if it's a new declaration, but for simplicity:
			if !strings.HasPrefix(strings.TrimSpace(line), "#") {
				currentClass = ""
				currentClassIndent = -1
			}
		}

		if match := pyClassRegex.FindStringSubmatch(line); len(match) > 1 {
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
			currentClass = className
			currentClassIndent = indent
			continue
		}

		if match := pyFuncRegex.FindStringSubmatch(line); len(match) > 1 {
			funcName := match[1]

			// Is it a method?
			isMethod := currentClass != "" && indent > currentClassIndent

			fullName := funcName
			if isMethod {
				fullName = currentClass + "." + funcName
			}

			nodeID := "func:" + fullName
			b.store.AddNode(&graph.Node{
				ID:     nodeID,
				Labels: []string{"Function"},
				Properties: map[string]string{
					"name": fullName,
				},
			})

			_ = b.store.AddEdge(&graph.Edge{
				SrcID: nodeID,
				DstID: fileNode.ID,
				Type:  "DefinedIn",
			})

			if isMethod {
				_ = b.store.AddEdge(&graph.Edge{
					SrcID: nodeID,
					DstID: "type:" + currentClass,
					Type:  "MethodOf",
				})
			}
		}
	}
}
