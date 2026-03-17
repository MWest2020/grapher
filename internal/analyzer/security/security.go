package security

import (
	"github.com/gongoeloe/grapher/internal/analyzer"
	"github.com/gongoeloe/grapher/internal/graph"
)

type Analyzer struct{}

func New() *Analyzer { return &Analyzer{} }

func (a *Analyzer) Name() string { return "security" }
func (a *Analyzer) Flag() string { return "security" }

func (a *Analyzer) Analyze(_ *graph.DiGraph) ([]analyzer.Finding, error) {
	// TODO: implement — hardcoded secrets, dangerous calls, SQL injection, CVE cross-reference
	return nil, nil
}
