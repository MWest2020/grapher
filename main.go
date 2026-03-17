package main

import (
	"fmt"
	"os"

	"github.com/gongoeloe/grapher/cmd"
	"github.com/gongoeloe/grapher/internal/analyzer"
	"github.com/gongoeloe/grapher/internal/analyzer/arch"
	"github.com/gongoeloe/grapher/internal/analyzer/deadcode"
	"github.com/gongoeloe/grapher/internal/analyzer/deps"
	"github.com/gongoeloe/grapher/internal/analyzer/security"
	"github.com/gongoeloe/grapher/internal/analyzer/tests"
)

func main() {
	// Register all analyzers here — cmd/analyze.go depends only on the Analyzer interface.
	cmd.Registry = []analyzer.Analyzer{
		deadcode.New(),
		security.New(),
		tests.New(),
		deps.New(),
		arch.New(),
	}

	if err := cmd.Root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
