package deps

import (
	"testing"

	"github.com/gongoeloe/grapher/internal/graph"
)

func TestStubReturnsEmpty(t *testing.T) {
	a := New()
	findings, err := a.Analyze(graph.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(findings))
	}
}
