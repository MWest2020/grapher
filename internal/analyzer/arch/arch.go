package arch

import (
	"github.com/gongoeloe/grapher/internal/analyzer"
	"github.com/gongoeloe/grapher/internal/graph"
)

type Analyzer struct{}

func New() *Analyzer { return &Analyzer{} }

func (a *Analyzer) Name() string { return "arch" }
func (a *Analyzer) Flag() string { return "arch" }

func (a *Analyzer) Analyze(_ *graph.DiGraph) ([]analyzer.Finding, error) {
	// TODO: implement — layering violations, god modules, circular deps, orphaned modules
	return nil, nil
}
