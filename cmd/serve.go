package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/gongoeloe/grapher/internal/analyzer"
	"github.com/gongoeloe/grapher/internal/graph"
	"github.com/gongoeloe/grapher/internal/parser"
	phpparser "github.com/gongoeloe/grapher/internal/parser/php"
	pyparser "github.com/gongoeloe/grapher/internal/parser/python"
	"github.com/gongoeloe/grapher/internal/serve"
)

var servePort int

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the grapher HTTP API server",
	Long: `Start an HTTP server that exposes grapher analysis as a REST API.

Runs all analyzers on the target repository and serves results as JSON.
Jobs are processed asynchronously — POST /api/v1/analyze to trigger a job,
then poll GET /api/v1/jobs/{id} for results.

Endpoints:
  POST /api/v1/analyze       Trigger analysis, returns job ID
  GET  /api/v1/jobs/{id}     Poll job status and results
  GET  /api/v1/findings      List findings (filter: ?analyzer=deadcode, ?sort=centrality)
  GET  /api/v1/graph         Code graph as JSON (nodes + edges) for visualization
  POST /api/v1/fix           Request a Claude fix proposal for a specific finding

Examples:
  grapher serve --repo ./my-repo --port 8080
  curl -X POST localhost:8080/api/v1/analyze -d '{"repo":".","analyzers":["deadcode"]}'`,
	RunE: runServe,
}

func init() {
	serveCmd.Flags().IntVar(&servePort, "port", 8080, "HTTP port to listen on")
}

func runServe(cmd *cobra.Command, args []string) error {
	store := serve.NewJobStore()
	pipeline := buildPipeline(repoPath)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	addr := fmt.Sprintf(":%d", servePort)
	return serve.Run(ctx, addr, store, pipeline)
}

// buildPipeline returns a Pipeline function that runs the requested analyzers.
func buildPipeline(defaultRepo string) serve.Pipeline {
	return func(repo string, analyzerNames []string) ([]analyzer.Finding, *graph.DiGraph, error) {
		if repo == "" {
			repo = defaultRepo
		}

		parsers := []parser.Parser{pyparser.New(), phpparser.New()}
		builder := graph.NewBuilder(parsers)
		g, _, err := builder.Build(repo)
		if err != nil {
			return nil, nil, fmt.Errorf("build graph: %w", err)
		}

		// Select analyzers
		var selected []analyzer.Analyzer
		if len(analyzerNames) == 0 {
			selected = Registry
		} else {
			nameSet := make(map[string]bool)
			for _, n := range analyzerNames {
				nameSet[n] = true
			}
			for _, a := range Registry {
				if nameSet[a.Flag()] {
					selected = append(selected, a)
				}
			}
		}

		var allFindings []analyzer.Finding
		for _, a := range selected {
			findings, err := a.Analyze(g)
			if err != nil {
				fmt.Fprintf(os.Stderr, "analyzer %s failed: %v\n", a.Name(), err)
				continue
			}
			allFindings = append(allFindings, findings...)
		}

		return allFindings, g, nil
	}
}
