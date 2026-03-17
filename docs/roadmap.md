# Roadmap

## Phase 2 — Language support

- [ ] JavaScript / TypeScript parser (tree-sitter)
- [ ] Go parser (tree-sitter, self-hosting)

## Phase 2 — Analyzer implementations

- [ ] `--security`: hardcoded secrets in string literals, dangerous calls (`eval`, `exec`, `shell_exec`, `system`), SQL injection patterns, dependency CVE cross-reference
- [ ] `--tests`: functions/methods with no corresponding test, prioritized by centrality
- [ ] `--deps`: outdated packages, known CVEs (OSV/Snyk), transitive dependency sprawl
- [ ] `--arch`: layering violations, god modules (unusually high degree), circular dependencies, orphaned modules

## Phase 2 — Remote repos

- [ ] `--from-github <owner/repo>` flag: clone or fetch via GitHub API before analysis
  - Option A: shell out to `git clone` into a temp dir
  - Option B: use `go-git` for pure-Go clone
  - Option C: GitHub Contents API for small repos (no clone needed)

## Phase 3 — Quality of life

- [ ] `--since <commit>` — only analyze symbols changed since a commit (incremental mode)
- [ ] CI mode: non-zero exit code when findings exceed a threshold (`--max-findings`, `--fail-on high`)
- [ ] SARIF output format for GitHub Code Scanning integration
- [ ] Config file (`.grapher.yaml`) for per-repo settings (entry point overrides, ignore patterns)
- [ ] Persistent job store for serve mode (SQLite)
- [ ] Full betweenness centrality (upgrade from out-degree approximation)

## Known limitations (v1)

- Only local repos — no GitHub/GitLab fetch support yet
- PHP tree-sitter grammar: some edge cases in method call node types may produce missed edges
- `--apply` shows Claude's prose proposals but does not parse/apply diffs automatically
- In-memory job store in serve mode — jobs lost on server restart
