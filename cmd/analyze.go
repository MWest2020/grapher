package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/gongoeloe/grapher/internal/analyzer"
	"github.com/gongoeloe/grapher/internal/fixer"
	"github.com/gongoeloe/grapher/internal/graph"
	"github.com/gongoeloe/grapher/internal/parser"
	phpparser "github.com/gongoeloe/grapher/internal/parser/php"
	pyparser "github.com/gongoeloe/grapher/internal/parser/python"
	"github.com/gongoeloe/grapher/internal/report"
)

// Registry of all available analyzers — populated by main.go.
var Registry []analyzer.Analyzer

var (
	flagDeadcode bool
	flagSecurity bool
	flagTests    bool
	flagDeps     bool
	flagArch     bool
	flagAll      bool
	flagFix      bool
	flagApply    bool
	flagJSON     bool
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze a repository (default subcommand)",
	Long: `Build a code graph from the target repository and run one or more analyzers.

Findings are written to ./grapher-reports/<analyzer>_<timestamp>.md by default.
Use --json to stream findings as JSON to stdout instead.

Fix mode (requires ANTHROPIC_API_KEY):
  --fix         Calls Claude API for each finding and prints a fix proposal
  --apply       After --fix, prompts per-file confirmation before writing changes

Examples:
  grapher --repo ./my-repo --deadcode
  grapher --repo ./my-repo --all --json
  grapher --repo ./my-repo --deadcode --fix
  grapher --repo ./my-repo --deadcode --fix --apply`,
	RunE: runAnalyze,
}

func init() {
	// Also wire flags on Root so `grapher --deadcode` works without the subcommand
	for _, cmd := range []*cobra.Command{analyzeCmd, Root} {
		cmd.Flags().BoolVar(&flagDeadcode, "deadcode", false, "Find unreachable symbols via graph reachability (BFS from entry points)")
		cmd.Flags().BoolVar(&flagSecurity, "security", false, "Find hardcoded secrets, dangerous calls, SQL injection (stub — coming soon)")
		cmd.Flags().BoolVar(&flagTests, "tests", false, "Find untested functions prioritized by centrality (stub — coming soon)")
		cmd.Flags().BoolVar(&flagDeps, "deps", false, "Find outdated or vulnerable dependencies (stub — coming soon)")
		cmd.Flags().BoolVar(&flagArch, "arch", false, "Find layering violations, god modules, circular deps (stub — coming soon)")
		cmd.Flags().BoolVar(&flagAll, "all", false, "Run all registered analyzers")
		cmd.Flags().BoolVar(&flagFix, "fix", false, "Call Claude API (ANTHROPIC_API_KEY) for a fix proposal per finding (read-only)")
		cmd.Flags().BoolVar(&flagApply, "apply", false, "Write fix proposals to disk with per-file confirmation (requires --fix)")
		cmd.Flags().BoolVar(&flagJSON, "json", false, "Output findings as JSON array to stdout instead of writing a Markdown report")
	}
	Root.RunE = runAnalyze
}

// styles
var (
	styleHeader  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	styleSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styleWarn    = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styleDanger  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	styleDim     = lipgloss.NewStyle().Faint(true)
)

func runAnalyze(cmd *cobra.Command, args []string) error {
	if flagApply && !flagFix {
		return fmt.Errorf("--apply requires --fix")
	}

	selected := selectedAnalyzers()
	if len(selected) == 0 {
		return fmt.Errorf("no analyzer selected — use --deadcode, --security, --tests, --deps, --arch, or --all")
	}

	// Build the graph
	fmt.Fprintf(os.Stderr, "%s Building code graph for %s...\n", styleHeader.Render("→"), repoPath)

	parsers := []parser.Parser{pyparser.New(), phpparser.New()}
	builder := graph.NewBuilder(parsers)

	symbolCount := 0
	builder.OnProgress = func(file string, count int) {
		symbolCount = count
		fmt.Fprintf(os.Stderr, "\r%s %d symbols found...", styleDim.Render("  parsing"), count)
	}

	g, caveats, err := builder.Build(repoPath)
	if err != nil {
		return fmt.Errorf("build graph: %w", err)
	}
	fmt.Fprintf(os.Stderr, "\r%s\n", styleSuccess.Render(fmt.Sprintf("  ✓ Graph built: %d symbols, %d edges, %d caveats",
		symbolCount, len(g.Edges), len(caveats))))

	if len(caveats) > 0 {
		fmt.Fprintf(os.Stderr, "%s\n", styleWarn.Render(fmt.Sprintf("  ⚠ %d dynamic dispatch caveats (see report for details)", len(caveats))))
	}

	// Run selected analyzers
	var allFindings []analyzer.Finding
	for _, a := range selected {
		fmt.Fprintf(os.Stderr, "%s Running %s analyzer...\n", styleHeader.Render("→"), a.Name())
		findings, err := a.Analyze(g)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", styleDanger.Render(fmt.Sprintf("  ✗ %s failed: %v", a.Name(), err)))
			continue
		}
		allFindings = append(allFindings, findings...)
		printSummaryTable(a.Name(), findings)
	}

	// --fix: get Claude proposals
	fixes := map[string]string{}
	if flagFix {
		f, err := fixer.New()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", styleDanger.Render("  ✗ "+err.Error()))
		} else {
			applier := fixer.NewApplier()
			for _, finding := range allFindings {
				key := fmt.Sprintf("%s:%d", finding.Symbol.File, finding.Symbol.Line)
				fmt.Fprintf(os.Stderr, "%s Getting fix for %s...\n", styleHeader.Render("→"), finding.Symbol.Name)
				proposal, err := f.Propose(context.Background(), finding)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%s\n", styleDanger.Render(fmt.Sprintf("  ✗ Claude error: %v", err)))
					continue
				}
				fixes[key] = proposal
				fmt.Printf("\n%s %s (%s:%d)\n%s\n",
					styleHeader.Render("Fix proposal for"), finding.Symbol.Name,
					finding.Symbol.File, finding.Symbol.Line,
					proposal)

				if flagApply {
					// For now, proposals describe changes — we show them and ask for confirmation
					// A full applier would parse the diff; here we show the proposal and skip writing
					// since Claude returns prose, not a diff. The user can apply manually.
					applier.ConfirmAndApply(finding.Symbol.File, proposal, "")
				}
			}
		}
	}

	// Output
	if flagJSON {
		return report.JSONReport(os.Stdout, allFindings)
	}

	path, err := report.MarkdownReport(selectedName(selected), allFindings, repoPath, fixes)
	if err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	fmt.Fprintf(os.Stderr, "\n%s\n", styleSuccess.Render(fmt.Sprintf("✓ Report written to %s", path)))
	return nil
}

func selectedAnalyzers() []analyzer.Analyzer {
	if flagAll {
		return Registry
	}
	var result []analyzer.Analyzer
	flagMap := map[string]bool{
		"deadcode": flagDeadcode,
		"security": flagSecurity,
		"tests":    flagTests,
		"deps":     flagDeps,
		"arch":     flagArch,
	}
	for _, a := range Registry {
		if flagMap[a.Flag()] {
			result = append(result, a)
		}
	}
	return result
}

func selectedName(analyzers []analyzer.Analyzer) string {
	if len(analyzers) == 1 {
		return analyzers[0].Name()
	}
	names := make([]string, len(analyzers))
	for i, a := range analyzers {
		names[i] = a.Name()
	}
	sort.Strings(names)
	return strings.Join(names, "_")
}

func printSummaryTable(name string, findings []analyzer.Finding) {
	counts := map[analyzer.Severity]int{}
	for _, f := range findings {
		counts[f.Severity]++
	}
	fmt.Fprintf(os.Stderr, "\n%s\n", styleHeader.Render(fmt.Sprintf("  %s results", name)))
	fmt.Fprintf(os.Stderr, "  %-10s %s\n", "Severity", "Count")
	fmt.Fprintf(os.Stderr, "  %-10s %s\n", "--------", "-----")
	for _, sev := range []analyzer.Severity{
		analyzer.SeverityCritical, analyzer.SeverityHigh,
		analyzer.SeverityMedium, analyzer.SeverityLow, analyzer.SeverityInfo,
	} {
		if c, ok := counts[sev]; ok {
			style := styleDim
			switch sev {
			case analyzer.SeverityCritical, analyzer.SeverityHigh:
				style = styleDanger
			case analyzer.SeverityMedium:
				style = styleWarn
			}
			fmt.Fprintf(os.Stderr, "  %-10s %s\n", strings.Title(string(sev)), style.Render(fmt.Sprintf("%d", c)))
		}
	}
	fmt.Fprintf(os.Stderr, "  %-10s %d\n\n", "Total", len(findings))
}
