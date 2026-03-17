package report

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/gongoeloe/grapher/internal/analyzer"
)

// JSONReport writes findings as a JSON array to w.
func JSONReport(w io.Writer, findings []analyzer.Finding) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(findings); err != nil {
		return fmt.Errorf("json encode findings: %w", err)
	}
	return nil
}
