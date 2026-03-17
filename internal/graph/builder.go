package graph

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/gongoeloe/grapher/internal/parser"
)

// Builder walks a repository and builds a DiGraph from all parseable files.
type Builder struct {
	parsers    map[string]parser.Parser // extension -> parser
	OnProgress func(file string, symbolCount int)
}

// NewBuilder creates a Builder with the given parsers.
func NewBuilder(parsers []parser.Parser) *Builder {
	b := &Builder{
		parsers: make(map[string]parser.Parser),
	}
	for _, p := range parsers {
		for _, ext := range p.Extensions() {
			b.parsers[ext] = p
		}
	}
	return b
}

// Build walks the repo directory and returns a fully assembled DiGraph.
func (b *Builder) Build(repoPath string) (*DiGraph, []parser.Caveat, error) {
	g := New()
	var allCaveats []parser.Caveat

	// First pass: collect all symbols as nodes
	allResults := make(map[string]*parser.ParseResult) // path -> result

	err := filepath.WalkDir(repoPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" ||
				name == "__pycache__" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		p, ok := b.parsers[ext]
		if !ok {
			return nil
		}

		src, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		result, err := p.Parse(path, src)
		if err != nil {
			// Non-fatal: log as caveat and continue
			allCaveats = append(allCaveats, parser.Caveat{
				File:    path,
				Line:    0,
				Message: fmt.Sprintf("parse error: %v", err),
			})
			return nil
		}

		allResults[path] = result
		allCaveats = append(allCaveats, result.Caveats...)

		for _, sym := range result.Symbols {
			nodeKind := symbolKindToNodeKind(sym.Kind)
			id := NodeID(path, sym.Name, nodeKind)
			node := &Node{
				ID:           id,
				Name:         sym.Name,
				Kind:         nodeKind,
				File:         sym.File,
				Line:         sym.Line,
				Language:     sym.Language,
				IsEntryPoint: sym.IsEntryPoint,
			}
			g.AddNode(node)
		}

		if b.OnProgress != nil {
			b.OnProgress(path, len(g.Nodes))
		}

		return nil
	})

	if err != nil {
		return nil, allCaveats, fmt.Errorf("walk %s: %w", repoPath, err)
	}

	// Second pass: add call edges (resolve callee names to node IDs)
	for path, result := range allResults {
		for _, call := range result.Calls {
			fromID := resolveSymbolID(g, path, call.CallerName)
			if fromID == "" {
				continue
			}
			toID := resolveSymbolIDByName(g, call.CalleeName)
			if toID == "" {
				continue
			}
			g.AddEdge(Edge{From: fromID, To: toID, Kind: EdgeKindCall})
		}
		for _, imp := range result.Imports {
			fromID := moduleNodeID(path)
			// Try to find a module node or any node matching the import name
			toID := resolveSymbolIDByName(g, imp.ImportedName)
			if toID == "" {
				continue
			}
			_ = fromID
			// We attach import edges from the first symbol in the file if no module node
			firstID := firstSymbolInFile(g, path)
			if firstID == "" {
				continue
			}
			g.AddEdge(Edge{From: firstID, To: toID, Kind: EdgeKindImport})
		}
	}

	g.ComputeCentrality()
	return g, allCaveats, nil
}

func symbolKindToNodeKind(k parser.SymbolKind) NodeKind {
	switch k {
	case parser.SymbolKindClass:
		return NodeKindClass
	case parser.SymbolKindMethod:
		return NodeKindMethod
	case parser.SymbolKindModule:
		return NodeKindModule
	default:
		return NodeKindFunction
	}
}

// resolveSymbolID finds a node ID for a symbol in the same file.
func resolveSymbolID(g *DiGraph, file, name string) string {
	for _, kind := range []NodeKind{NodeKindFunction, NodeKindMethod, NodeKindClass, NodeKindModule} {
		id := NodeID(file, name, kind)
		if _, ok := g.Nodes[id]; ok {
			return id
		}
	}
	return ""
}

// resolveSymbolIDByName finds any node matching name across all files.
func resolveSymbolIDByName(g *DiGraph, name string) string {
	for id, node := range g.Nodes {
		if node.Name == name {
			return id
		}
	}
	return ""
}

func moduleNodeID(file string) string {
	return NodeID(file, filepath.Base(file), NodeKindModule)
}

func firstSymbolInFile(g *DiGraph, file string) string {
	for id, node := range g.Nodes {
		if node.File == file {
			return id
		}
	}
	return ""
}
