package report

import (
	"os"
	"strings"
	"testing"

	"github.com/gongoeloe/grapher/internal/analyzer"
	"github.com/gongoeloe/grapher/internal/graph"
)

func TestMarkdownReportCreatesFile(t *testing.T) {
	// Use a temp dir so we don't pollute the working directory
	orig, _ := os.Getwd()
	tmp := t.TempDir()
	os.Chdir(tmp)
	defer os.Chdir(orig)

	findings := []analyzer.Finding{
		{
			AnalyzerName: "deadcode",
			Symbol:       graph.Node{Name: "unused_func", Kind: graph.NodeKindFunction, File: "main.py", Line: 42},
			Severity:     analyzer.SeverityMedium,
			Centrality:   0.5,
			Why:          "no callers",
			Suggestion:   "remove it",
			FixPrompt:    "please fix",
			Callers:      nil,
		},
	}

	path, err := MarkdownReport("deadcode", findings, "/my/repo", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("report file not found: %v", err)
	}
	content, _ := os.ReadFile(path)
	if !strings.Contains(string(content), "unused_func") {
		t.Error("expected finding symbol name in report")
	}
	if !strings.Contains(string(content), "Summary") {
		t.Error("expected Summary section")
	}
}

func TestMarkdownReportEmpty(t *testing.T) {
	orig, _ := os.Getwd()
	tmp := t.TempDir()
	os.Chdir(tmp)
	defer os.Chdir(orig)

	path, err := MarkdownReport("deadcode", nil, "/my/repo", nil)
	if err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(path)
	if !strings.Contains(string(content), "No findings") {
		t.Error("expected 'No findings' in empty report")
	}
}
