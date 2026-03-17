package parser

// Parser extracts symbols and relationships from source files of a given language.
type Parser interface {
	Language() string
	Extensions() []string
	Parse(path string, src []byte) (*ParseResult, error)
}

// ParseResult is the raw output of parsing one file.
type ParseResult struct {
	Symbols  []Symbol
	Calls    []CallEdge
	Imports  []ImportEdge
	Caveats  []Caveat
}

// Symbol represents an extracted code symbol.
type Symbol struct {
	Name         string
	Kind         SymbolKind
	File         string
	Line         int
	Language     string
	IsEntryPoint bool
	ParentName   string // for methods: the class name
}

// SymbolKind identifies the kind of symbol.
type SymbolKind string

const (
	SymbolKindFunction SymbolKind = "function"
	SymbolKindClass    SymbolKind = "class"
	SymbolKindMethod   SymbolKind = "method"
	SymbolKindModule   SymbolKind = "module"
)

// CallEdge represents a call from one symbol to another within a file.
type CallEdge struct {
	CallerName string
	CalleeName string
	Line       int
}

// ImportEdge represents a file-level import of a name.
type ImportEdge struct {
	SourceFile string
	ImportedName string
	Line       int
}

// Caveat records a warning about analysis limitations (e.g. dynamic dispatch).
type Caveat struct {
	File    string
	Line    int
	Message string
}
