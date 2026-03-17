package analyzer

import "github.com/gongoeloe/grapher/internal/graph"

// Severity ranks how critical a finding is.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// Finding is a single reported issue from an analyzer.
type Finding struct {
	AnalyzerName string
	Symbol       graph.Node
	Severity     Severity
	Centrality   float64
	Title        string
	Why          string
	Suggestion   string
	FixPrompt    string
	Callers      []string // node names of callers
}

// Analyzer runs a specific analysis over the full code graph.
type Analyzer interface {
	Name() string
	Flag() string
	Analyze(g *graph.DiGraph) ([]Finding, error)
}
