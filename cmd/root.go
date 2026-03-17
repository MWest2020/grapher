package cmd

import (
	"github.com/spf13/cobra"
)

var repoPath string

// Root is the root cobra command.
var Root = &cobra.Command{
	Use:   "grapher",
	Short: "Analyze codebases via code graphs and surface actionable insights",
	Long: `grapher is an extensible static analysis tool that builds a directed code graph
from your repository and surfaces dead code, security issues, test coverage gaps,
dependency problems, and architectural violations.

Every symbol (function, class, method) becomes a graph node. Every relationship
(calls, imports, inherits) becomes a directed edge. Analyzers run as graph queries
over this structure, ranking findings by centrality — how central the affected
symbol is to the rest of the codebase.

Supported languages (Phase 1): Python, PHP
Supported analyzers: deadcode (full), security/tests/deps/arch (stubs)

Examples:
  grapher --repo ./my-repo --deadcode
  grapher --repo ./my-repo --all
  grapher --repo ./my-repo --deadcode --fix
  grapher --repo ./my-repo --deadcode --fix --apply
  grapher --repo ./my-repo --deadcode --json
  grapher serve --repo ./my-repo --port 8080`,
}

func init() {
	Root.PersistentFlags().StringVar(&repoPath, "repo", ".", "Path to the local repository to analyze (Python and PHP files only in Phase 1)")
	Root.AddCommand(analyzeCmd)
	Root.AddCommand(serveCmd)
}
