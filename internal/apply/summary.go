package apply

import (
	"fmt"
	"os"
	"strings"

	"github.com/abiosoft/incus-apply/internal/terminal"
)

// ANSI color aliases for concise usage within this package.
var (
	colorRed    = terminal.ColorRed
	colorGreen  = terminal.ColorGreen
	colorYellow = terminal.ColorYellow
	colorDim    = terminal.ColorDim
	colorReset  = terminal.ColorReset
)

// result tracks planning and execution statistics for summary reporting.
type result struct {
	created   int
	updated   int
	replaced  int
	unchanged int
	deleted   int
	skipped   int
	errors    []error
}

// recordError adds an error to the result and returns it if FailFast is enabled.
func (s *result) recordError(failFast bool, resourceID, action string, err error) error {
	errMsg := fmt.Errorf("%s: %s: %w", resourceID, action, err)
	printError("%s: %v", resourceID, err)
	s.errors = append(s.errors, errMsg)
	if failFast {
		return errMsg
	}
	return nil
}

// errorResult returns an error summarising all recorded errors, or nil.
func (s result) errorResult() error {
	if len(s.errors) == 1 {
		return s.errors[0]
	}
	if len(s.errors) > 0 {
		return fmt.Errorf("%d error(s) occurred", len(s.errors))
	}
	return nil
}

func (s result) hasErrors() bool {
	return len(s.errors) > 0
}

func (s result) upsertSummary() string {
	var parts []string
	if s.created > 0 {
		parts = append(parts, fmt.Sprintf("%d to create", s.created))
	}
	if s.updated > 0 {
		parts = append(parts, fmt.Sprintf("%d to update", s.updated))
	}
	if s.replaced > 0 {
		parts = append(parts, fmt.Sprintf("%d to replace", s.replaced))
	}
	if len(s.errors) > 0 {
		parts = append(parts, fmt.Sprintf("%d errors", len(s.errors)))
	}
	if len(parts) == 0 {
		return ""
	}
	return "Summary: " + strings.Join(parts, ", ") + "."
}

func (s result) deleteSummary() string {
	var parts []string
	if s.deleted > 0 {
		parts = append(parts, fmt.Sprintf("%d to delete", s.deleted))
	}
	if len(s.errors) > 0 {
		parts = append(parts, fmt.Sprintf("%d errors", len(s.errors)))
	}
	if len(parts) == 0 {
		return ""
	}
	return "Summary: " + strings.Join(parts, ", ") + "."
}

func resetSummary(del, create *result) string {
	var parts []string
	if del.deleted > 0 {
		parts = append(parts, fmt.Sprintf("%d to delete", del.deleted))
	}
	if create.created > 0 {
		parts = append(parts, fmt.Sprintf("%d to create", create.created))
	}
	errs := len(del.errors) + len(create.errors)
	if errs > 0 {
		parts = append(parts, fmt.Sprintf("%d errors", errs))
	}
	if len(parts) == 0 {
		return ""
	}
	return "Summary: " + strings.Join(parts, ", ") + "."
}

// --- Summary Printing ---

// printer prints a summary of operation statistics.
type printer interface {
	Print(quiet bool, stats result)
}

// upsertPrinter prints upsert operation summaries.
type upsertPrinter struct{}

func (upsertPrinter) Print(quiet bool, stats result) {
	if quiet {
		return
	}
	var parts []string
	if stats.created > 0 {
		parts = append(parts, fmt.Sprintf("%d created", stats.created))
	}
	if stats.updated > 0 {
		parts = append(parts, fmt.Sprintf("%d updated", stats.updated))
	}
	if stats.replaced > 0 {
		parts = append(parts, fmt.Sprintf("%d replaced", stats.replaced))
	}
	if stats.unchanged > 0 {
		parts = append(parts, fmt.Sprintf("%d unchanged", stats.unchanged))
	}
	if len(stats.errors) > 0 {
		parts = append(parts, fmt.Sprintf("%d errors", len(stats.errors)))
	}
	if len(parts) > 0 {
		fmt.Println()
		fmt.Println("Summary: " + strings.Join(parts, ", ") + ".")
	}
}

// deletePrinter prints delete operation summaries.
type deletePrinter struct{}

func (deletePrinter) Print(quiet bool, stats result) {
	if quiet {
		return
	}
	fmt.Println()
	fmt.Printf("Summary: %d deleted, %d skipped, %d errors.\n",
		stats.deleted, stats.skipped, len(stats.errors))
}

// --- Output Helpers ---

func printColored(quiet bool, color, format string, args ...any) {
	if !quiet {
		fmt.Printf(color+format+colorReset+"\n", args...)
	}
}

func printError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}

func printInfo(quiet bool, format string, args ...any) {
	if !quiet {
		fmt.Printf(format+"\n", args...)
	}
}

func printWarning(quiet bool, format string, args ...any) {
	printColored(quiet, colorYellow, format, args...)
}
