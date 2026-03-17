# Grapher Build Log

## 2026-03-16 — Initial build (v0.1)

### What was built
Full implementation of the `grapher` CLI tool from scratch.

### Sequence
1. **openspec artifacts** — proposal, specs (9 capability specs), design, tasks
2. **openapi.yaml** — full OpenAPI 3.1 spec for serve mode (5 endpoints)
3. **internal/graph** — DiGraph, Node, Edge types + repo walker/builder
4. **internal/parser** — Parser interface + Python and PHP parsers via tree-sitter
5. **internal/analyzer** — Analyzer interface + deadcode (BFS reachability) + 4 stubs (security, tests, deps, arch)
6. **internal/report** — Markdown reporter + JSON reporter
7. **internal/fixer** — Claude API integration (anthropic-sdk-go v1.26) + file applier
8. **internal/serve** — async job store + HTTP handlers + server
9. **cmd** — root, analyze, serve cobra commands
10. **main.go** — analyzer registry wired in

### Compile fixes applied
- `n.Child(uint32(i))` → `n.Child(i)` across python.go and php.go (sitter.Node.Child takes int)
- Anthropic SDK v1.26 API: replaced `anthropic.F()` / `anthropic.UserMessageParam` with
  `anthropic.NewUserMessage()` / `anthropic.NewTextBlock()` and direct struct field assignment

### Test results
```
ok  internal/analyzer/arch
ok  internal/analyzer/deadcode
ok  internal/analyzer/deps
ok  internal/analyzer/security
ok  internal/analyzer/tests
ok  internal/graph
ok  internal/parser/php
ok  internal/parser/python
ok  internal/report
```

### Smoke test
```
grapher --repo /tmp/test-repo --deadcode
```
Input: 5 symbols (main, used, dead_function, DeadClass, dead_method)
Result: 3 dead findings (dead_function, DeadClass, dead_method) ✓
main and used correctly NOT reported (main=entry point, used=reachable from main) ✓

### Post-build fix
- PHP parser: `isEntry := topLevel` was marking ALL top-level function definitions as entry points,
  suppressing dead code detection. Fixed to only use `isPublicMethod()` for entry detection;
  top-level *calls* still handled by `markTopLevelCalls()`.

### Docs and help improvements (2026-03-17)
- Improved `--help` output: Long descriptions on root, analyze, serve commands
- All flags now have descriptive usage strings (stubs marked "coming soon")
- Created `docs/` directory:
  - `docs/getting-started.md` — install, quick start, language/analyzer status table
  - `docs/how-it-works.md` — graph model, dead code algorithm, centrality, package structure, how to add an analyzer
  - `docs/serve-api.md` — HTTP API reference with curl examples
  - `docs/roadmap.md` — Phase 2 todos incl. GitHub remote repo support

### Known limitations / TODOs
- 4 analyzer stubs return empty findings (security, tests, deps, arch)
- `--apply` shows proposals but doesn't write diffs (Claude returns prose, not diffs)
- In-memory job store in serve mode — jobs lost on restart
- Centrality = normalized out-degree (not full betweenness centrality)
- PHP: `member_call_expression` node type may differ across PHP grammar versions
- No JS/TS or Go parsers yet (Phase 2)
