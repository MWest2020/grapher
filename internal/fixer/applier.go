package fixer

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Applier presents Claude's proposals to the user and optionally writes changes.
type Applier struct {
	in  io.Reader
	out io.Writer
}

func NewApplier() *Applier {
	return &Applier{in: os.Stdin, out: os.Stdout}
}

// ConfirmAndApply shows the proposal for a file and asks for confirmation.
// If confirmed, it writes newContent to filePath.
func (a *Applier) ConfirmAndApply(filePath, proposal, newContent string) (bool, error) {
	fmt.Fprintf(a.out, "\n--- Fix proposal for %s ---\n", filePath)
	fmt.Fprintln(a.out, proposal)
	fmt.Fprintf(a.out, "\nApply this change to %s? [y/N] ", filePath)

	scanner := bufio.NewScanner(a.in)
	if !scanner.Scan() {
		return false, nil
	}
	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	if answer != "y" && answer != "yes" {
		fmt.Fprintln(a.out, "Skipped.")
		return false, nil
	}

	if err := os.WriteFile(filePath, []byte(newContent), 0o644); err != nil {
		return false, fmt.Errorf("write %s: %w", filePath, err)
	}
	fmt.Fprintf(a.out, "Applied to %s.\n", filePath)
	return true, nil
}
