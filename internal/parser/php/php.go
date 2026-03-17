package php

import (
	"context"
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	tsphp "github.com/smacker/go-tree-sitter/php"

	"github.com/gongoeloe/grapher/internal/parser"
)

// Parser implements parser.Parser for PHP source files.
type Parser struct{}

func New() *Parser { return &Parser{} }

func (p *Parser) Language() string     { return "php" }
func (p *Parser) Extensions() []string { return []string{".php"} }

func (p *Parser) Parse(path string, src []byte) (*parser.ParseResult, error) {
	lang := tsphp.GetLanguage()
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
	v.walk(root, "", false)
	v.walkForImports(root)
	return result, nil
}

type visitor struct {
	src        []byte
	path       string
	result     *parser.ParseResult
	classStack []string
}

func (v *visitor) currentClass() string {
	if len(v.classStack) == 0 {
		return ""
	}
	return v.classStack[len(v.classStack)-1]
}

func (v *visitor) walk(n *sitter.Node, callerName string, topLevel bool) {
	if n == nil {
		return
	}

	switch n.Type() {
	case "program":
		for i := 0; i < int(n.ChildCount()); i++ {
			v.walk(n.Child(i), "", true)
		}
		return

	case "class_declaration":
		name := ""
		if nameNode := n.ChildByFieldName("name"); nameNode != nil {
			name = nameNode.Content(v.src)
		}
		v.result.Symbols = append(v.result.Symbols, parser.Symbol{
			Name:     name,
			Kind:     parser.SymbolKindClass,
			File:     v.path,
			Line:     int(n.StartPoint().Row) + 1,
			Language: "php",
		})
		v.classStack = append(v.classStack, name)
		body := n.ChildByFieldName("body")
		if body != nil {
			v.walk(body, callerName, false)
		}
		v.classStack = v.classStack[:len(v.classStack)-1]
		return

	case "method_declaration", "function_definition":
		name := ""
		if nameNode := n.ChildByFieldName("name"); nameNode != nil {
			name = nameNode.Content(v.src)
		}
		kind := parser.SymbolKindFunction
		parentClass := v.currentClass()
		if parentClass != "" || n.Type() == "method_declaration" {
			kind = parser.SymbolKindMethod
		}

		// Entry point only for public methods; top-level calls are handled by markTopLevelCalls
		isEntry := v.isPublicMethod(n)
		sym := parser.Symbol{
			Name:         name,
			Kind:         kind,
			File:         v.path,
			Line:         int(n.StartPoint().Row) + 1,
			Language:     "php",
			IsEntryPoint: isEntry,
			ParentName:   parentClass,
		}
		v.result.Symbols = append(v.result.Symbols, sym)

		body := n.ChildByFieldName("body")
		if body != nil {
			v.walkForCalls(body, name)
		}
		return

	case "expression_statement":
		if topLevel {
			v.markTopLevelCalls(n)
		}
	}

	for i := 0; i < int(n.ChildCount()); i++ {
		v.walk(n.Child(i), callerName, topLevel)
	}
}

func (v *visitor) isPublicMethod(n *sitter.Node) bool {
	for i := 0; i < int(n.ChildCount()); i++ {
		child := n.Child(i)
		if child.Type() == "visibility_modifier" && child.Content(v.src) == "public" {
			return true
		}
	}
	return false
}

func (v *visitor) markTopLevelCalls(n *sitter.Node) {
	if n == nil {
		return
	}
	if n.Type() == "function_call_expression" {
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
		v.markTopLevelCalls(n.Child(i))
	}
}

func (v *visitor) walkForCalls(n *sitter.Node, callerName string) {
	if n == nil {
		return
	}

	switch n.Type() {
	case "function_call_expression":
		fn := n.ChildByFieldName("function")
		if fn != nil {
			callee := fn.Content(v.src)
			if callee == "call_user_func" || callee == "call_user_func_array" {
				v.result.Caveats = append(v.result.Caveats, parser.Caveat{
					File:    v.path,
					Line:    int(n.StartPoint().Row) + 1,
					Message: "possible dynamic dispatch via " + callee + " — call graph may be incomplete",
				})
			} else if strings.HasPrefix(callee, "$") {
				v.result.Caveats = append(v.result.Caveats, parser.Caveat{
					File:    v.path,
					Line:    int(n.StartPoint().Row) + 1,
					Message: "possible dynamic dispatch via variable function call — call graph may be incomplete",
				})
			} else if callee != callerName {
				v.result.Calls = append(v.result.Calls, parser.CallEdge{
					CallerName: callerName,
					CalleeName: callee,
					Line:       int(n.StartPoint().Row) + 1,
				})
			}
		}

	case "method_call_expression", "static_method_call_expression":
		nameNode := n.ChildByFieldName("name")
		if nameNode != nil {
			callee := nameNode.Content(v.src)
			if callee == "__call" || callee == "__get" {
				v.result.Caveats = append(v.result.Caveats, parser.Caveat{
					File:    v.path,
					Line:    int(n.StartPoint().Row) + 1,
					Message: "possible dynamic dispatch via magic method — call graph may be incomplete",
				})
			} else if callee != callerName {
				v.result.Calls = append(v.result.Calls, parser.CallEdge{
					CallerName: callerName,
					CalleeName: callee,
					Line:       int(n.StartPoint().Row) + 1,
				})
			}
		}
	}

	for i := 0; i < int(n.ChildCount()); i++ {
		v.walkForCalls(n.Child(i), callerName)
	}
}

func (v *visitor) walkForImports(root *sitter.Node) {
	v.findImports(root)
}

func (v *visitor) findImports(n *sitter.Node) {
	if n == nil {
		return
	}
	switch n.Type() {
	case "require_expression", "require_once_expression",
		"include_expression", "include_once_expression":
		for i := 0; i < int(n.ChildCount()); i++ {
			child := n.Child(i)
			if child.Type() == "string" || child.Type() == "encapsed_string" {
				v.result.Imports = append(v.result.Imports, parser.ImportEdge{
					SourceFile:   v.path,
					ImportedName: child.Content(v.src),
					Line:         int(n.StartPoint().Row) + 1,
				})
			}
		}
	}
	for i := 0; i < int(n.ChildCount()); i++ {
		v.findImports(n.Child(i))
	}
}
