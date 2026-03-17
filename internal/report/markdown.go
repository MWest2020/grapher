package report

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gongoeloe/grapher/internal/analyzer"
)

// MarkdownReport writes a Markdown report to ./grapher-reports/<analyzer>_<timestamp>.md
// and returns the path written.
func MarkdownReport(analyzerName string, findings []analyzer.Finding, repoPath string, fixes map[string]string) (string, error) {
	dir := "grapher-reports"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create report dir: %w", err)
	}

	ts := time.Now().UTC().Format("20060102_150405")
	filename := filepath.Join(dir, fmt.Sprintf("%s_%s.md", analyzerName, ts))

	var sb strings.Builder
	writeMarkdown(&sb, analyzerName, findings, repoPath, fixes)

	if err := os.WriteFile(filename, []byte(sb.String()), 0o644); err != nil {
		return "", fmt.Errorf("write report: %w", err)
	}
	return filename, nil
}

func writeMarkdown(sb *strings.Builder, analyzerName string, findings []analyzer.Finding, repoPath string, fixes map[string]string) {
	sb.WriteString(fmt.Sprintf("# Grapher — %s Report\n\n", strings.Title(analyzerName)))
	sb.WriteString(fmt.Sprintf("**Repo:** %s  \n", repoPath))
	sb.WriteString(fmt.Sprintf("**Generated:** %s UTC  \n\n", time.Now().UTC().Format("2006-01-02 15:04:05")))

	// Summary table
	counts := map[analyzer.Severity]int{}
	for _, f := range findings {
		counts[f.Severity]++
	}
	sb.WriteString("## Summary\n\n")
	sb.WriteString("| Severity | Count |\n|----------|-------|\n")
	for _, sev := range []analyzer.Severity{
		analyzer.SeverityCritical, analyzer.SeverityHigh,
		analyzer.SeverityMedium, analyzer.SeverityLow, analyzer.SeverityInfo,
	} {
		if c, ok := counts[sev]; ok {
			sb.WriteString(fmt.Sprintf("| %s | %d |\n", strings.Title(string(sev)), c))
		}
	}
	sb.WriteString(fmt.Sprintf("| **Total** | **%d** |\n\n", len(findings)))

	if len(findings) == 0 {
		sb.WriteString("No findings. Clean!\n\n")
		writeWhatNext(sb, 0)
		return
	}

	// Group findings by file
	byFile := map[string][]analyzer.Finding{}
	for _, f := range findings {
		byFile[f.Symbol.File] = append(byFile[f.Symbol.File], f)
	}
	files := make([]string, 0, len(byFile))
	for f := range byFile {
		files = append(files, f)
	}
	sort.Strings(files)

	sb.WriteString("## Findings\n\n")
	for _, file := range files {
		sb.WriteString(fmt.Sprintf("### `%s`\n\n", file))
		fileFindings := byFile[file]
		sort.Slice(fileFindings, func(i, j int) bool {
			return fileFindings[i].Symbol.Line < fileFindings[j].Symbol.Line
		})
		for _, f := range fileFindings {
			writeFinding(sb, f)
		}
	}

	// Fix proposals section
	if len(fixes) > 0 {
		sb.WriteString("## Claude Fix Proposals\n\n")
		for key, proposal := range fixes {
			sb.WriteString(fmt.Sprintf("### `%s`\n\n", key))
			sb.WriteString(proposal)
			sb.WriteString("\n\n---\n\n")
		}
	}

	writeWhatNext(sb, len(findings))
}

func writeFinding(sb *strings.Builder, f analyzer.Finding) {
	sb.WriteString(fmt.Sprintf("#### `%s` — %s (line %d)\n\n", f.Symbol.Name, f.Symbol.Kind, f.Symbol.Line))
	sb.WriteString(fmt.Sprintf("**Severity:** %s  \n", strings.Title(string(f.Severity))))
	sb.WriteString(fmt.Sprintf("**Centrality:** %.2f  \n", f.Centrality))
	sb.WriteString(fmt.Sprintf("**Why:** %s  \n", f.Why))
	sb.WriteString(fmt.Sprintf("**Suggestion:** %s  \n", f.Suggestion))
	if len(f.Callers) > 0 {
		sb.WriteString(fmt.Sprintf("**Callers:** %s  \n", strings.Join(f.Callers, ", ")))
	} else {
		sb.WriteString("**Callers:** none  \n")
	}
	sb.WriteString("\n---\n\n")
}

func writeWhatNext(sb *strings.Builder, total int) {
	sb.WriteString("## What to do next\n\n")
	if total == 0 {
		sb.WriteString("- No action required.\n")
		return
	}
	sb.WriteString("- Review High severity findings first — they have the highest blast radius.\n")
	sb.WriteString("- Run with `--fix` to get AI-powered fix proposals for each finding.\n")
	sb.WriteString("- Run with `--fix --apply` to interactively write fixes to disk.\n")
}
