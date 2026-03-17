package tests

import (
	"github.com/gongoeloe/grapher/internal/analyzer"
	"github.com/gongoeloe/grapher/internal/graph"
)

type Analyzer struct{}

func New() *Analyzer { return &Analyzer{} }

func (a *Analyzer) Name() string { return "tests" }
func (a *Analyzer) Flag() string { return "tests" }

func (a *Analyzer) Analyze(_ *graph.DiGraph) ([]analyzer.Finding, error) {
	// TODO: implement — untested functions/methods, prioritized by centrality
	return nil, nil
}
