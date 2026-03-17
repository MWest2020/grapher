package deadcode

import (
	"testing"

	"github.com/gongoeloe/grapher/internal/graph"
)

func buildGraph() *graph.DiGraph {
	g := graph.New()
	entry := &graph.Node{ID: "entry", Name: "main", Kind: graph.NodeKindFunction, File: "main.py", Line: 1, Language: "python", IsEntryPoint: true}
	used := &graph.Node{ID: "used", Name: "used_func", Kind: graph.NodeKindFunction, File: "main.py", Line: 5, Language: "python"}
	dead := &graph.Node{ID: "dead", Name: "dead_func", Kind: graph.NodeKindFunction, File: "main.py", Line: 10, Language: "python"}
	g.AddNode(entry)
	g.AddNode(used)
	g.AddNode(dead)
	g.AddEdge(graph.Edge{From: "entry", To: "used", Kind: graph.EdgeKindCall})
	g.ComputeCentrality()
	return g
}

func TestDeadCodeDetected(t *testing.T) {
	g := buildGraph()
	a := New()
	findings, err := a.Analyze(g)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 dead finding, got %d", len(findings))
	}
	if findings[0].Symbol.Name != "dead_func" {
		t.Errorf("expected dead_func, got %s", findings[0].Symbol.Name)
	}
}

func TestEntryPointNotDead(t *testing.T) {
	g := buildGraph()
	a := New()
	findings, err := a.Analyze(g)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		if f.Symbol.Name == "main" {
			t.Error("entry point should not be reported as dead")
		}
	}
}

func TestFixPromptAlwaysPopulated(t *testing.T) {
	g := buildGraph()
	a := New()
	findings, err := a.Analyze(g)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		if f.FixPrompt == "" {
			t.Errorf("finding %s has empty FixPrompt", f.Symbol.Name)
		}
	}
}

func TestSeverityFromCentrality(t *testing.T) {
	cases := []struct {
		c    float64
		want string
	}{
		{0.8, "high"},
		{0.5, "medium"},
		{0.1, "low"},
	}
	for _, tc := range cases {
		got := string(severityFromCentrality(tc.c))
		if got != tc.want {
			t.Errorf("centrality %.1f: expected %s, got %s", tc.c, tc.want, got)
		}
	}
}
