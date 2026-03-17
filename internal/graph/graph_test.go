package graph

import (
	"testing"
)

func TestAddNodeAndEdge(t *testing.T) {
	g := New()
	a := &Node{ID: "a", Name: "funcA", Kind: NodeKindFunction, File: "a.py", Line: 1, Language: "python"}
	b := &Node{ID: "b", Name: "funcB", Kind: NodeKindFunction, File: "a.py", Line: 10, Language: "python"}
	g.AddNode(a)
	g.AddNode(b)
	g.AddEdge(Edge{From: "a", To: "b", Kind: EdgeKindCall})

	if len(g.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(g.Nodes))
	}
	if len(g.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(g.Edges))
	}
	if g.InDegree("b") != 1 {
		t.Errorf("expected in-degree 1 for b, got %d", g.InDegree("b"))
	}
	callers := g.CallerIDs("b")
	if len(callers) != 1 || callers[0] != "a" {
		t.Errorf("expected caller [a], got %v", callers)
	}
}

func TestCentrality(t *testing.T) {
	g := New()
	hub := &Node{ID: "hub", Name: "hub", Kind: NodeKindFunction, File: "f.py", Line: 1, Language: "python"}
	leaf := &Node{ID: "leaf", Name: "leaf", Kind: NodeKindFunction, File: "f.py", Line: 10, Language: "python"}
	g.AddNode(hub)
	g.AddNode(leaf)
	g.AddEdge(Edge{From: "hub", To: "leaf", Kind: EdgeKindCall})

	g.ComputeCentrality()

	if hub.Centrality != 1.0 {
		t.Errorf("expected hub centrality 1.0, got %f", hub.Centrality)
	}
	if leaf.Centrality != 0.0 {
		t.Errorf("expected leaf centrality 0.0, got %f", leaf.Centrality)
	}
}

func TestCentralityNoEdges(t *testing.T) {
	g := New()
	g.AddNode(&Node{ID: "a", Name: "a", Kind: NodeKindFunction, File: "f.py", Line: 1, Language: "python"})
	g.ComputeCentrality()
	if g.Nodes["a"].Centrality != 0.0 {
		t.Errorf("expected 0 centrality with no edges, got %f", g.Nodes["a"].Centrality)
	}
}

func TestNodeID(t *testing.T) {
	id := NodeID("main.py", "foo", NodeKindFunction)
	if id != "main.py::function::foo" {
		t.Errorf("unexpected node ID: %s", id)
	}
}
