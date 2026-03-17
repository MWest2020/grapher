package python

import (
	"context"
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	tspython "github.com/smacker/go-tree-sitter/python"

	"github.com/gongoeloe/grapher/internal/parser"
)

// Parser implements parser.Parser for Python source files.
type Parser struct{}

func New() *Parser { return &Parser{} }

func (p *Parser) Language() string       { return "python" }
func (p *Parser) Extensions() []string   { return []string{".py"} }

func (p *Parser) Parse(path string, src []byte) (*parser.ParseResult, error) {
	lang := tspython.GetLanguage()
	root, err := sitter.ParseCtx(context.Background(), src, lang)
	if err != nil {
		return nil, fmt.Errorf("tree-sitter parse %s: %w", path, err)
	}

	result := &parser.ParseResult{}
	v := &visitor{
		src:    src,
		path:   path,
		result: result,
	}
	v.walk(root, "")
	v.detectMainEntryPoints(root)

	return result, nil
}

type visitor struct {
	src    []byte
	path   string
	result *parser.ParseResult
	// track current class scope for method attribution
	classStack []string
}

func (v *visitor) currentClass() string {
	if len(v.classStack) == 0 {
		return ""
	}
	return v.classStack[len(v.classStack)-1]
}

func (v *visitor) walk(n *sitter.Node, callerName string) {
	if n == nil {
		return
	}

	t := n.Type()

	switch t {
	case "class_definition":
		name := ""
		if nameNode := n.ChildByFieldName("name"); nameNode != nil {
			name = nameNode.Content(v.src)
		}
		sym := parser.Symbol{
			Name:     name,
			Kind:     parser.SymbolKindClass,
			File:     v.path,
			Line:     int(n.StartPoint().Row) + 1,
			Language: "python",
		}
		v.result.Symbols = append(v.result.Symbols, sym)
		v.classStack = append(v.classStack, name)
		body := n.ChildByFieldName("body")
		if body != nil {
			v.walk(body, callerName)
		}
		v.classStack = v.classStack[:len(v.classStack)-1]
		return

	case "function_definition":
		name := ""
		if nameNode := n.ChildByFieldName("name"); nameNode != nil {
			name = nameNode.Content(v.src)
		}
		kind := parser.SymbolKindFunction
		parentClass := v.currentClass()
		if parentClass != "" {
			kind = parser.SymbolKindMethod
		}
		isEntry := v.isFunctionEntryPoint(n, name)
		sym := parser.Symbol{
			Name:         name,
			Kind:         kind,
			File:         v.path,
			Line:         int(n.StartPoint().Row) + 1,
			Language:     "python",
			IsEntryPoint: isEntry,
			ParentName:   parentClass,
		}
		v.result.Symbols = append(v.result.Symbols, sym)

		// Walk body for calls, passing this function as the caller
		body := n.ChildByFieldName("body")
		if body != nil {
			v.walkForCalls(body, name)
		}
		return

	case "decorated_definition":
		// Check decorators for entry point markers, then walk the inner definition
		v.walkDecoratedDef(n, callerName)
		return
	}

	// Default: recurse
	for i := 0; i < int(n.ChildCount()); i++ {
		v.walk(n.Child(i), callerName)
	}
}

func (v *visitor) walkDecoratedDef(n *sitter.Node, callerName string) {
	var decorators []string
	var defNode *sitter.Node

	for i := 0; i < int(n.ChildCount()); i++ {
		child := n.Child(i)
		switch child.Type() {
		case "decorator":
			decorators = append(decorators, child.Content(v.src))
		case "function_definition", "class_definition":
			defNode = child
		}
	}

	if defNode == nil {
		return
	}

	// Walk the definition node normally; entry point detection uses decorator list
	if defNode.Type() == "function_definition" {
		name := ""
		if nameNode := defNode.ChildByFieldName("name"); nameNode != nil {
			name = nameNode.Content(v.src)
		}
		kind := parser.SymbolKindFunction
		parentClass := v.currentClass()
		if parentClass != "" {
			kind = parser.SymbolKindMethod
		}
		isEntry := v.isFunctionEntryPoint(defNode, name) || v.isDecoratorEntryPoint(decorators)
		sym := parser.Symbol{
			Name:         name,
			Kind:         kind,
			File:         v.path,
			Line:         int(defNode.StartPoint().Row) + 1,
			Language:     "python",
			IsEntryPoint: isEntry,
			ParentName:   parentClass,
		}
		v.result.Symbols = append(v.result.Symbols, sym)
		body := defNode.ChildByFieldName("body")
		if body != nil {
			v.walkForCalls(body, name)
		}
	} else {
		v.walk(defNode, callerName)
	}
}

func (v *visitor) isFunctionEntryPoint(n *sitter.Node, name string) bool {
	// test_ prefix
	if strings.HasPrefix(name, "test_") || name == "setUp" || name == "tearDown" {
		return true
	}
	return false
}

func (v *visitor) isDecoratorEntryPoint(decorators []string) bool {
	entryPatterns := []string{
		"@app.route", "@route", "@get", "@post", "@put", "@delete", "@patch",
		"@pytest.fixture", "@pytest.mark", "@unittest",
		"@celery", "@task", "@shared_task",
		"@click.command", "@cli.command",
	}
	for _, dec := range decorators {
		for _, pat := range entryPatterns {
			if strings.HasPrefix(dec, pat) {
				return true
			}
		}
	}
	return false
}

// detectMainEntryPoints scans for `if __name__ == "__main__"` blocks and marks
// any direct calls within them as entry points.
func (v *visitor) detectMainEntryPoints(root *sitter.Node) {
	v.findMainBlock(root)
}

func (v *visitor) findMainBlock(n *sitter.Node) {
	if n == nil {
		return
	}
	if n.Type() == "if_statement" {
		cond := n.ChildByFieldName("condition")
		if cond != nil && strings.Contains(cond.Content(v.src), "__name__") &&
			strings.Contains(cond.Content(v.src), "__main__") {
			// Mark any direct call in the block as an entry point
			body := n.ChildByFieldName("consequence")
			if body != nil {
				v.markCallsAsEntryPoints(body)
			}
			return
		}
	}
	for i := 0; i < int(n.ChildCount()); i++ {
		v.findMainBlock(n.Child(i))
	}
}

func (v *visitor) markCallsAsEntryPoints(n *sitter.Node) {
	if n == nil {
		return
	}
	if n.Type() == "call" {
		fn := n.ChildByFieldName("function")
		if fn != nil {
			name := fn.Content(v.src)
			for i := range v.result.Symbols {
				if v.result.Symbols[i].Name == name {
					v.result.Symbols[i].IsEntryPoint = true
				}
			}
		}
	}
	for i := 0; i < int(n.ChildCount()); i++ {
		v.markCallsAsEntryPoints(n.Child(i))
	}
}

// walkForCalls traverses a subtree looking for call expressions and getattr.
func (v *visitor) walkForCalls(n *sitter.Node, callerName string) {
	if n == nil {
		return
	}

	switch n.Type() {
	case "call":
		fn := n.ChildByFieldName("function")
		if fn != nil {
			callee := extractCalleeName(fn, v.src)
			if callee != "" && callee != callerName {
				v.result.Calls = append(v.result.Calls, parser.CallEdge{
					CallerName: callerName,
					CalleeName: callee,
					Line:       int(n.StartPoint().Row) + 1,
				})
			}
			if fn.Type() == "identifier" && fn.Content(v.src) == "getattr" {
				v.result.Caveats = append(v.result.Caveats, parser.Caveat{
					File:    v.path,
					Line:    int(n.StartPoint().Row) + 1,
					Message: "possible dynamic dispatch via getattr — call graph may be incomplete",
				})
			}
		}
	}

	for i := 0; i < int(n.ChildCount()); i++ {
		v.walkForCalls(n.Child(i), callerName)
	}
}

// extractCalleeName extracts the simple name from a call's function node.
func extractCalleeName(fn *sitter.Node, src []byte) string {
	switch fn.Type() {
	case "identifier":
		return fn.Content(src)
	case "attribute":
		attr := fn.ChildByFieldName("attribute")
		if attr != nil {
			return attr.Content(src)
		}
	}
	return ""
}

// walkForImports extracts import statements from the module level.
func (p *Parser) walkForImports(root *sitter.Node, src []byte, path string, result *parser.ParseResult) {
	for i := 0; i < int(root.ChildCount()); i++ {
		child := root.Child(i)
		switch child.Type() {
		case "import_statement":
			for j := 0; j < int(child.ChildCount()); j++ {
				c := child.Child(j)
				if c.Type() == "dotted_name" || c.Type() == "identifier" {
					result.Imports = append(result.Imports, parser.ImportEdge{
						SourceFile:   path,
						ImportedName: c.Content(src),
						Line:         int(child.StartPoint().Row) + 1,
					})
				}
			}
		case "import_from_statement":
			for j := 0; j < int(child.ChildCount()); j++ {
				c := child.Child(j)
				if c.Type() == "dotted_name" || c.Type() == "identifier" {
					result.Imports = append(result.Imports, parser.ImportEdge{
						SourceFile:   path,
						ImportedName: c.Content(src),
						Line:         int(child.StartPoint().Row) + 1,
					})
				}
			}
		}
	}
}
