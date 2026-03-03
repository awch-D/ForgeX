package archaeology

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/awch-D/ForgeX/forgex-cognition/graph"
	"github.com/awch-D/ForgeX/forgex-core/logger"
)

// Builder traverses a Go codebase and generates a knowledge graph.
type Builder struct {
	store *graph.Store
	fset  *token.FileSet
}

// NewBuilder creates a new graph builder.
func NewBuilder(store *graph.Store) *Builder {
	return &Builder{
		store: store,
		fset:  token.NewFileSet(),
	}
}

// Build scans the given directory and builds the graph.
func (b *Builder) Build(dir string) error {
	logger.L().Infof("🔍 正在扫描架构图谱: %s", dir)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor and hidden dirs
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") || info.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		b.parseFile(path, dir)
		return nil
	})

	return err
}

func (b *Builder) parseFile(filePath, rootDir string) {
	relPath, err := filepath.Rel(rootDir, filePath)
	if err != nil {
		relPath = filePath
	}

	// 1. Create File Node
	fileNode := &graph.Node{
		ID:     "file:" + relPath,
		Labels: []string{"File"},
		Properties: map[string]string{
			"path": relPath,
		},
	}
	b.store.AddNode(fileNode)

	// Parse AST
	f, err := parser.ParseFile(b.fset, filePath, nil, parser.ParseComments)
	if err != nil {
		logger.L().Debugw("Failed to parse file for graph", "file", relPath, "error", err)
		return
	}

	// First pass: identify all declarations in this file
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					nodeID := "type:" + ts.Name.Name
					label := "Type"

					if _, isStruct := ts.Type.(*ast.StructType); isStruct {
						label = "Struct"
					} else if _, isInterface := ts.Type.(*ast.InterfaceType); isInterface {
						label = "Interface"
					}

					node := &graph.Node{
						ID:     nodeID,
						Labels: []string{label},
						Properties: map[string]string{
							"name": ts.Name.Name,
							"pkg":  f.Name.Name,
						},
					}
					b.store.AddNode(node)

					// Edge: Type DefinedIn File
					_ = b.store.AddEdge(&graph.Edge{
						SrcID: nodeID,
						DstID: fileNode.ID,
						Type:  "DefinedIn",
					})
				}
			}

		case *ast.FuncDecl:
			funcName := d.Name.Name
			var recvName string

			// Handle methods attached to structs
			if d.Recv != nil && len(d.Recv.List) > 0 {
				switch r := d.Recv.List[0].Type.(type) {
				case *ast.Ident:
					recvName = r.Name
				case *ast.StarExpr:
					if ident, ok := r.X.(*ast.Ident); ok {
						recvName = ident.Name
					}
				}
				funcName = recvName + "." + funcName
			}

			nodeID := "func:" + funcName
			node := &graph.Node{
				ID:     nodeID,
				Labels: []string{"Function"},
				Properties: map[string]string{
					"name": funcName,
					"pkg":  f.Name.Name,
				},
			}
			b.store.AddNode(node)

			// Edge: Function DefinedIn File
			_ = b.store.AddEdge(&graph.Edge{
				SrcID: nodeID,
				DstID: fileNode.ID,
				Type:  "DefinedIn",
			})

			if recvName != "" {
				// Edge: Function MethodOf Struct
				_ = b.store.AddEdge(&graph.Edge{
					SrcID: nodeID,
					DstID: "type:" + recvName,
					Type:  "MethodOf",
				})
			}

			// Second pass: Find calls within function
			b.extractCalls(d.Body, nodeID)
		}
	}
}

func (b *Builder) extractCalls(body *ast.BlockStmt, callerID string) {
	if body == nil {
		return
	}

	ast.Inspect(body, func(n ast.Node) bool {
		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		var targetName string

		switch fun := callExpr.Fun.(type) {
		case *ast.Ident:
			// Local function call: foo()
			targetName = fun.Name
		case *ast.SelectorExpr:
			// Package or method call: pkg.Foo() or obj.Foo()
			if xIdent, ok := fun.X.(*ast.Ident); ok {
				targetName = xIdent.Name + "." + fun.Sel.Name
			} else {
				// Complex selector, just record the method name
				targetName = fun.Sel.Name
			}
		}

		if targetName != "" {
			targetID := "func:" + targetName

			// We boldly add the edge even if target node might not exist yet (or is external)
			// A lightweight graph database allows dangling edges or we can lazily create the node.
			// To be safe with our strict Store, let's artificially ensure the target node exists as an external ghost if missing.

			if _, exists := b.store.GetNode(targetID); !exists {
				b.store.AddNode(&graph.Node{
					ID:     targetID,
					Labels: []string{"Function", "External"},
					Properties: map[string]string{
						"name": targetName,
					},
				})
			}

			_ = b.store.AddEdge(&graph.Edge{
				SrcID: callerID,
				DstID: targetID,
				Type:  "Calls",
			})
		}
		return true
	})
}
