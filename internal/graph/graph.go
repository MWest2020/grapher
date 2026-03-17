package graph

import "fmt"

// NodeKind identifies what kind of symbol a node represents.
type NodeKind string

const (
	NodeKindFunction NodeKind = "function"
	NodeKindClass    NodeKind = "class"
	NodeKindMethod   NodeKind = "method"
	NodeKindModule   NodeKind = "module"
)

// EdgeKind identifies the type of relationship between two nodes.
type EdgeKind string

const (
	EdgeKindCall       EdgeKind = "call"
	EdgeKindImport     EdgeKind = "import"
	EdgeKindInherits   EdgeKind = "inherits"
	EdgeKindImplements EdgeKind = "implements"
)

// Node represents a symbol in the code graph.
type Node struct {
	ID          string
	Name        string
	Kind        NodeKind
	File        string
	Line        int
	Language    string
	IsEntryPoint bool
	Centrality  float64
}

// Edge represents a directed relationship between two nodes.
type Edge struct {
	From string // Node ID
	To   string // Node ID
	Kind EdgeKind
}

// DiGraph is a directed graph of code symbols.
type DiGraph struct {
	Nodes map[string]*Node
	Edges []Edge

	// outDegree[id] = number of unique targets this node calls
	outDegree map[string]int
	// inEdges[id] = list of source node IDs pointing to this node
	inEdges map[string][]string
}

// New creates an empty DiGraph.
func New() *DiGraph {
	return &DiGraph{
		Nodes:     make(map[string]*Node),
		outDegree: make(map[string]int),
		inEdges:   make(map[string][]string),
	}
}

// AddNode adds a node to the graph. If a node with the same ID exists, it is overwritten.
func (g *DiGraph) AddNode(n *Node) {
	g.Nodes[n.ID] = n
}

// AddEdge adds a directed edge. Both node IDs must already exist; unknown IDs are silently ignored
// to handle forward-references across files.
func (g *DiGraph) AddEdge(e Edge) {
	g.Edges = append(g.Edges, e)
	if _, ok := g.Nodes[e.From]; ok {
		g.outDegree[e.From]++
	}
	if _, ok := g.Nodes[e.To]; ok {
		g.inEdges[e.To] = append(g.inEdges[e.To], e.From)
	}
}

// CallerIDs returns all node IDs that have an edge pointing to nodeID.
func (g *DiGraph) CallerIDs(nodeID string) []string {
	return g.inEdges[nodeID]
}

// InDegree returns the number of incoming edges for nodeID.
func (g *DiGraph) InDegree(nodeID string) int {
	return len(g.inEdges[nodeID])
}

// ComputeCentrality sets centrality on every node based on normalized out-degree.
// centrality = outDegree(node) / max(outDegree across all nodes), or 0 if max is 0.
func (g *DiGraph) ComputeCentrality() {
	max := 0
	for _, v := range g.outDegree {
		if v > max {
			max = v
		}
	}
	for id, node := range g.Nodes {
		if max == 0 {
			node.Centrality = 0
		} else {
			node.Centrality = float64(g.outDegree[id]) / float64(max)
		}
	}
}

// NodeID produces a stable, unique ID for a symbol given its file, name, and kind.
func NodeID(file, name string, kind NodeKind) string {
	return fmt.Sprintf("%s::%s::%s", file, kind, name)
}
