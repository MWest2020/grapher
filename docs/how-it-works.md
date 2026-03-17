# How It Works

## The code graph

grapher parses your repository with [tree-sitter](https://tree-sitter.github.io/) and builds a directed graph:

- **Nodes** — every symbol: functions, classes, methods, modules
- **Edges** — every relationship: calls, imports, inheritance, interface implementation
- **Entry points** — nodes reachable by definition: `main` functions, exported symbols, route handlers, test functions, event listeners

Each node gets a **centrality score** (0.0–1.0) based on normalized out-degree: how many unique symbols it calls. Higher = larger blast radius if removed.

## Dead code detection (`--deadcode`)

1. Build the full call graph
2. BFS/DFS from all entry points
3. Any node not visited = dead

Each finding includes:
- Symbol name, kind, file, line
- Why it's dead (zero callers, or only called from other dead code)
- Callers list (even if dead)
- Centrality score
- Severity: High (≥0.7), Medium (≥0.3), Low (<0.3)
- A `FixPrompt` (always populated, even without `--fix`)

**Caveats flagged automatically:**
- Python: `getattr`, callable arguments
- PHP: `call_user_func`, variable functions (`$func()`), magic methods

## Entry point detection

| Language | Entry point rules |
|----------|-------------------|
| Python | `if __name__ == "__main__"` callees, `test_*` functions, `@app.route`, `@pytest.*`, `@click.command` |
| PHP | `public` methods, functions called at top level of file |

## Fix mode (`--fix`)

When `--fix` is passed, grapher calls the Claude API (model: `claude-sonnet-4-20250514`) for each finding. The `FixPrompt` includes: symbol name/kind, file:line, why it's dead, callers, and centrality.

`--fix` alone is **always read-only**. `--apply` enables writing changes after per-file confirmation.

Requires `ANTHROPIC_API_KEY` environment variable.

## Centrality formula

```
centrality(node) = outDegree(node) / max(outDegree across all nodes)
```

This is a fast O(E) approximation of blast radius. Full betweenness centrality may be added in a future version.

## Package structure

```
grapher/
├── main.go                    # analyzer registry + cobra wiring
├── openapi.yaml               # OpenAPI 3.1 spec for serve mode
├── cmd/
│   ├── root.go               # root command, --repo flag
│   ├── analyze.go            # analyzer flags, pipeline, progress output
│   └── serve.go              # HTTP serve subcommand
└── internal/
    ├── graph/                 # DiGraph, Node, Edge, centrality, repo walker
    ├── parser/python/         # tree-sitter Python parser
    ├── parser/php/            # tree-sitter PHP parser
    ├── analyzer/deadcode/     # BFS dead code analyzer
    ├── analyzer/security/     # stub
    ├── analyzer/tests/        # stub
    ├── analyzer/deps/         # stub
    ├── analyzer/arch/         # stub
    ├── fixer/                 # Claude API + file applier
    ├── report/                # Markdown + JSON reporters
    └── serve/                 # HTTP server, job store, handlers
```

## Adding a new analyzer

1. Create `internal/analyzer/<name>/<name>.go` implementing the `Analyzer` interface:
   ```go
   type Analyzer interface {
       Name() string
       Flag() string
       Analyze(g *graph.DiGraph) ([]Finding, error)
   }
   ```
2. Register it in `main.go`:
   ```go
   cmd.Registry = []analyzer.Analyzer{
       deadcode.New(),
       yourpkg.New(), // ← add here
   }
   ```
3. That's it — the CLI flag (`--<Flag()>`) is registered automatically.
