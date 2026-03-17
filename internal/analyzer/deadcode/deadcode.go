package deadcode

import (
	"fmt"
	"strings"

	"github.com/gongoeloe/grapher/internal/analyzer"
	"github.com/gongoeloe/grapher/internal/graph"
)

// Analyzer detects unreachable symbols via BFS from entry points.
type Analyzer struct{}

func New() *Analyzer { return &Analyzer{} }

func (a *Analyzer) Name() string { return "deadcode" }
func (a *Analyzer) Flag() string { return "deadcode" }

func (a *Analyzer) Analyze(g *graph.DiGraph) ([]analyzer.Finding, error) {
	// Build adjacency: nodeID -> list of called nodeIDs
	adj := make(map[string][]string)
	for _, edge := range g.Edges {
		if edge.Kind == graph.EdgeKindCall || edge.Kind == graph.EdgeKindImport {
			adj[edge.From] = append(adj[edge.From], edge.To)
		}
	}

	// BFS from all entry points
	visited := make(map[string]bool)
	queue := []string{}
	for id, node := range g.Nodes {
		if node.IsEntryPoint {
			queue = append(queue, id)
			visited[id] = true
		}
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, next := range adj[cur] {
			if !visited[next] {
				visited[next] = true
				queue = append(queue, next)
			}
		}
	}

	// Dead = not visited
	var findings []analyzer.Finding
	for id, node := range g.Nodes {
		if visited[id] {
			continue
		}

		callerIDs := g.CallerIDs(id)
		callerNames := make([]string, 0, len(callerIDs))
		for _, cid := range callerIDs {
			if cn, ok := g.Nodes[cid]; ok {
				callerNames = append(callerNames, cn.Name)
			}
		}

		sev := severityFromCentrality(node.Centrality)

		why := buildWhy(node, callerNames)
		suggestion := buildSuggestion(node)
		fixPrompt := buildFixPrompt(node, why, callerNames)

		findings = append(findings, analyzer.Finding{
			AnalyzerName: a.Name(),
			Symbol:       *node,
			Severity:     sev,
			Centrality:   node.Centrality,
			Title:        fmt.Sprintf("Dead %s: %s", node.Kind, node.Name),
			Why:          why,
			Suggestion:   suggestion,
			FixPrompt:    fixPrompt,
			Callers:      callerNames,
		})
	}

	return findings, nil
}

func severityFromCentrality(c float64) analyzer.Severity {
	switch {
	case c >= 0.7:
		return analyzer.SeverityHigh
	case c >= 0.3:
		return analyzer.SeverityMedium
	default:
		return analyzer.SeverityLow
	}
}

func buildWhy(node *graph.Node, callers []string) string {
	if len(callers) == 0 {
		return fmt.Sprintf(
			"`%s` (%s at %s:%d) has no callers and is not an entry point.",
			node.Name, node.Kind, node.File, node.Line,
		)
	}
	return fmt.Sprintf(
		"`%s` (%s at %s:%d) is only called from other dead code: %s.",
		node.Name, node.Kind, node.File, node.Line, strings.Join(callers, ", "),
	)
}

func buildSuggestion(node *graph.Node) string {
	switch node.Kind {
	case graph.NodeKindClass:
		return "Remove the class if unused, or mark it with a TODO if you plan to use it later."
	case graph.NodeKindMethod:
		return "Remove the method, or convert it to a private helper if it may be needed."
	default:
		return "Remove the function if truly unused, or move it to a utils module if it may be reused."
	}
}

func buildFixPrompt(node *graph.Node, why string, callers []string) string {
	callerStr := "none"
	if len(callers) > 0 {
		callerStr = strings.Join(callers, ", ")
	}
	return fmt.Sprintf(
		"Dead code finding:\n"+
			"Symbol: %s (%s)\n"+
			"File: %s, line %d\n"+
			"Language: %s\n"+
			"Why it is dead: %s\n"+
			"Callers: %s\n"+
			"Centrality: %.2f\n\n"+
			"Please suggest the safest way to remove or refactor this dead code. "+
			"If the symbol is used via dynamic dispatch or reflection, note that and recommend adding a comment instead of deleting.",
		node.Name, node.Kind,
		node.File, node.Line,
		node.Language,
		why,
		callerStr,
		node.Centrality,
	)
}
