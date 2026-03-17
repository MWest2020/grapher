# Getting Started

## What is grapher?

grapher is a single-binary CLI tool that analyzes codebases by building a **code graph** — a directed graph where every symbol (function, class, method) is a node and every relationship (calls, imports, inheritance) is an edge.

Analyzers run as graph queries over this structure. Every finding is ranked by **centrality** — how many other symbols depend on the affected node. A dead function that calls 40 other things has a higher blast radius than a leaf nobody uses.

## Installation

```bash
git clone https://github.com/gongoeloe/grapher
cd grapher
go build -o grapher .
# optionally move to PATH
mv grapher /usr/local/bin/grapher
```

Requires Go 1.23+. No runtime dependencies — single static binary.

## Quick start

```bash
# Find dead code in a Python/PHP repo
grapher --repo ./my-repo --deadcode

# Run all analyzers
grapher --repo ./my-repo --all

# Output as JSON
grapher --repo ./my-repo --deadcode --json

# Get Claude fix proposals (requires ANTHROPIC_API_KEY)
ANTHROPIC_API_KEY=sk-... grapher --repo ./my-repo --deadcode --fix

# Interactively apply fixes
ANTHROPIC_API_KEY=sk-... grapher --repo ./my-repo --deadcode --fix --apply
```

Reports are written to `./grapher-reports/<analyzer>_<timestamp>.md`.

## Supported languages

| Language | Status |
|----------|--------|
| Python   | ✅ Phase 1 |
| PHP      | ✅ Phase 1 |
| JavaScript / TypeScript | 🔜 Phase 2 |
| Go       | 🔜 Phase 2 |

## Supported analyzers

| Flag | Status | Description |
|------|--------|-------------|
| `--deadcode` | ✅ | Unreachable symbols via BFS from entry points |
| `--security` | 🔜 stub | Hardcoded secrets, dangerous calls, SQL injection |
| `--tests`    | 🔜 stub | Untested functions prioritized by centrality |
| `--deps`     | 🔜 stub | Outdated/vulnerable dependencies |
| `--arch`     | 🔜 stub | Layering violations, god modules, circular deps |
