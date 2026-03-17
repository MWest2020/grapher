# Grapher — Claude Code Prompt

Paste this entire prompt into Claude Code at the start of a new session.

---

## Prompt

You are a senior Go engineer and software architect. We are building **`grapher`** — an extensible CLI tool that analyzes codebases via code graphs and surfaces actionable insights. Think of it as a static analysis platform where each concern (dead code, security, test coverage, dependencies, architecture violations) is a pluggable module.

Your first job is to **spec this out properly before writing any code**. Produce:

1. An **OpenAPI 3.1 spec** (`openapi.yaml`) for the grapher HTTP report server (optional mode: `grapher serve`)
2. A **Go architecture document** (`ARCHITECTURE.md`) covering package structure, interfaces, and extension points
3. Then implement the full tool in Go, starting with the dead code module

---

## Vision

```
grapher --repo ./my-repo --deadcode
grapher --repo ./my-repo --deadcode --fix
grapher --repo ./my-repo --security
grapher --repo ./my-repo --tests
grapher --repo ./my-repo --deps
grapher --repo ./my-repo --arch
grapher --repo ./my-repo --all
grapher --repo ./my-repo --all --fix --apply
grapher serve --repo ./my-repo --port 8080
```

Single binary. No runtime dependencies. Point it at a repo, get a rich Markdown report plus optional Claude-powered fix proposals.

---

## Core concepts

**Code graph:** Every symbol (function, class, method, module) is a node. Every relationship (calls, imports, inherits, implements) is a directed edge. All analysis runs as graph queries over this structure.

**Entry points:** Nodes that are reachable by definition — main functions, route handlers, test functions, exported public interfaces, event listeners, decorators that register handlers. Everything not reachable from an entry point is a candidate for flagging.

**Modules:** Each analyzer (`--deadcode`, `--security`, etc.) is a self-contained module implementing a shared `Analyzer` interface. The CLI discovers and runs them independently or together.

**Severity:** Every finding is ranked not just by type but by **graph centrality** — how many other nodes depend on the affected node. A security issue in a highly central function is critical. The same issue in a leaf node nobody calls is low priority.

**Fix mode:** `--fix` calls the Claude API (claude-sonnet-4-20250514) with full context about each finding — the symbol, its source, why it's flagged, and what depends on it. Claude proposes a fix in the terminal. `--apply` asks for per-file confirmation then writes changes.

---

## Language support

Build the parser layer with **tree-sitter** (Go bindings: `github.com/smacker/go-tree-sitter`).

Phase 1 (implement now):
- Python
- PHP

Phase 2 (stub interfaces, implement later):
- JavaScript / TypeScript
- Go (self-hosting)

Each language needs:
- Symbol extraction (functions, classes, methods)
- Call/import edge extraction
- Entry point detection rules (language-specific)

---

## Analyzer modules

### `--deadcode`
**What:** Find symbols with no callers that are not entry points.

**How:**
1. Build directed call graph
2. BFS/DFS from all entry points
3. Unreachable nodes = dead

**Report per finding:**
- Symbol name, kind (function/class/method), file, line
- Why it's dead (in-degree 0, or only called from other dead code)
- List of callers if any (even dead ones)
- Centrality score (how many things *it* calls, giving a sense of blast radius if removed)
- Suggestion: remove / mark as TODO / move to utils

**Caveats to flag:**
- Dynamic dispatch (`getattr`, `call_user_func`, `$func()`)
- Magic methods and dunder methods
- Reflection-based usage

---

### `--security` (stub now, implement later)
- Hardcoded secrets in string literals connected to auth flows
- Dangerous function calls: `eval`, `exec`, `shell_exec`, `system`
- SQL injection patterns
- Dependency CVE cross-reference

---

### `--tests` (stub now, implement later)
- Functions/methods with no corresponding test
- Prioritized by centrality (untested central code = high risk)
- Suggest test file and test function names

---

### `--deps` (stub now, implement later)
- Outdated packages
- Known vulnerabilities (cross-ref OSV / Snyk)
- Overly heavy dependencies for simple tasks
- Transitive dependency sprawl

---

### `--arch` (stub now, implement later)
- Layering violations (e.g. DB layer calling UI layer)
- God modules (nodes with unusually high degree)
- Circular dependencies
- Orphaned modules

---

## Go package structure

Design this properly. Suggested layout — adapt if you have a better idea, but justify it:

```
grapher/
├── main.go
├── cmd/
│   ├── root.go          # cobra root command
│   ├── analyze.go       # --repo + analyzer flags
│   └── serve.go         # grapher serve
├── internal/
│   ├── graph/
│   │   ├── graph.go     # DiGraph type, node/edge structs
│   │   └── builder.go   # walks repo, delegates to parsers
│   ├── parser/
│   │   ├── parser.go    # Parser interface
│   │   ├── python/      # tree-sitter Python
│   │   └── php/         # tree-sitter PHP
│   ├── analyzer/
│   │   ├── analyzer.go  # Analyzer interface + Finding struct
│   │   ├── deadcode/
│   │   ├── security/    # stub
│   │   ├── tests/       # stub
│   │   ├── deps/        # stub
│   │   └── arch/        # stub
│   ├── fixer/
│   │   ├── fixer.go     # Claude API integration
│   │   └── applier.go   # writes changes to files
│   └── report/
│       ├── markdown.go
│       └── json.go
└── openapi.yaml
```

---

## Key interfaces to define

```go
// Parser extracts symbols and edges from source files of a given language.
type Parser interface {
    Language() string
    Extensions() []string
    Parse(path string, src []byte) (*ParseResult, error)
}

// ParseResult is the raw output of parsing one file.
type ParseResult struct {
    Symbols    []Symbol
    Calls      []Edge
    Imports    []Edge
}

// Analyzer runs a specific analysis over the full graph.
type Analyzer interface {
    Name() string
    Flag() string   // CLI flag name, e.g. "deadcode"
    Analyze(g *graph.Graph) ([]Finding, error)
}

// Finding is a single reported issue.
type Finding struct {
    AnalyzerName string
    Symbol       graph.Symbol
    Severity     Severity   // Critical / High / Medium / Low / Info
    Centrality   float64    // 0.0 - 1.0, derived from graph degree
    Title        string
    Why          string
    Suggestion   string
    FixPrompt    string     // sent to Claude if --fix
}
```

---

## Report format

Default output: Markdown file written to `./grapher-reports/<analyzer>_<timestamp>.md`

Structure:
```markdown
# Grapher — Dead Code Report

**Repo:** ...
**Generated:** ...

## Summary
| Metric | Value |
...

## Findings

### `path/to/file.py`

#### `function_name` — function (line 42)
**Severity:** Medium
**Centrality:** 0.12 (calls 4 other symbols)
**Why it's dead:** Nothing calls this function and it is not an entry point.
**Suggestion:** ...

---

## Claude Fix Proposals (if --fix)
...

## What to do next
...
```

---

## CLI UX

Use `cobra` for commands and `github.com/charmbracelet/lipgloss` or similar for terminal output. The terminal output while running should show:

- A progress indicator while parsing
- Live count of symbols found
- Summary table after each analyzer
- Dead symbols listed with file:line, kind, and one-line reason
- Fix proposals printed inline before asking for confirmation

---

## `grapher serve`

Optional HTTP server mode. Runs all analyzers and serves results as JSON via REST API. This is what the OpenAPI spec covers.

Endpoints:
- `GET /api/v1/analyze` — trigger analysis, returns job ID
- `GET /api/v1/jobs/{id}` — poll job status and results
- `GET /api/v1/findings` — list all findings with filter/sort
- `GET /api/v1/graph` — return graph as JSON (nodes + edges) for visualization
- `POST /api/v1/fix` — request Claude fix proposal for a specific finding

---

## What to build, in order

1. `openapi.yaml` — full spec for the serve API
2. `ARCHITECTURE.md` — package design, interfaces, decisions
3. `internal/graph/` — core graph types
4. `internal/parser/python/` — Python parser via tree-sitter
5. `internal/analyzer/deadcode/` — dead code analyzer
6. `internal/report/markdown.go` — Markdown reporter
7. `cmd/` — CLI wiring with cobra
8. `internal/fixer/` — Claude API integration
9. `internal/parser/php/` — PHP parser
10. Stub remaining analyzers with proper interfaces

---

## Constraints and quality bar

- All interfaces must be defined before implementations
- Each package must have a `_test.go` with at least one meaningful test
- No globals — pass dependencies explicitly
- The `Analyzer` interface must be the only thing `cmd/analyze.go` depends on — new analyzers should require zero changes to CLI code
- Claude API key read from `ANTHROPIC_API_KEY` env var, never hardcoded
- `--fix` without `--apply` is always read-only (no file changes)
- Findings must always include a `FixPrompt` field even if `--fix` is not used — this is the context that would be sent to Claude

Start with the OpenAPI spec and architecture doc. Ask me any clarifying questions before writing code if something is ambiguous.
