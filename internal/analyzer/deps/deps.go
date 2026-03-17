package deps

import (
	"github.com/gongoeloe/grapher/internal/analyzer"
	"github.com/gongoeloe/grapher/internal/graph"
)

type Analyzer struct{}

func New() *Analyzer { return &Analyzer{} }

func (a *Analyzer) Name() string { return "deps" }
func (a *Analyzer) Flag() string { return "deps" }

func (a *Analyzer) Analyze(_ *graph.DiGraph) ([]analyzer.Finding, error) {
	// TODO: implement — outdated packages, CVEs, transitive sprawl
	return nil, nil
}
