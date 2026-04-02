package renderer

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/abiosoft/incus-apply/internal/apply"
	"github.com/abiosoft/incus-apply/internal/incus"
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

// actionColors maps actions to their display colors.
var actionColors = map[apply.Action]string{
	apply.ActionCreate:   colorGreen,
	apply.ActionUpdate:   colorYellow,
	apply.ActionReplace:  colorYellow,
	apply.ActionUnchange: colorDim,
	apply.ActionDelete:   colorRed,
	apply.ActionNotFound: colorDim,
}

// actionPrefixes maps actions to their line prefix symbol.
var actionPrefixes = map[apply.Action]string{
	apply.ActionCreate:   "+",
	apply.ActionUpdate:   "~",
	apply.ActionReplace:  "!",
	apply.ActionUnchange: "=",
	apply.ActionDelete:   "-",
	apply.ActionNotFound: "=",
}

// TextRenderer renders output to the terminal with ANSI colours.
type TextRenderer struct {
	Writer io.Writer
	Quiet  bool
}

// NewTextRenderer creates a TextRenderer that writes to stdout.
func NewTextRenderer(quiet bool) *TextRenderer {
	return &TextRenderer{Writer: os.Stdout, Quiet: quiet}
}

// Render outputs the preview results to the terminal.
func (r TextRenderer) Render(output apply.Output) error {
	if r.Quiet {
		return nil
	}

	r.println()
	r.printf("Found %d %s in %d %s.\n",
		output.ResourceCount, plural("resource", output.ResourceCount),
		output.FileCount, plural("file", output.FileCount))

	if len(output.Groups) > 0 {
		r.println()
		hasActions := false
		for _, g := range output.Groups {
			if g.Action != apply.ActionUnchange {
				hasActions = true
				break
			}
		}
		if hasActions {
			r.println("The following actions would be performed:")
		} else {
			r.println("No actions would be performed.")
		}

		for _, group := range output.Groups {
			r.println()
			r.printf("  %s (%d):\n", group.Action, len(group.Items))
			color := actionColors[group.Action]
			prefix := actionPrefixes[group.Action]
			maxWidth := terminal.Width(r.Writer, 100)
			for _, item := range group.Items {
				r.printf("%s    %s %s%s\n", color, prefix, item.ResourceID, colorReset)
				if len(item.Changes) > 0 {
					r.print(incus.FormatDiffChangesWithWidth(item.Changes, "      ", maxWidth))
				}
				if item.Note != "" {
					r.printf("      └─ %s\n", item.Note)
				}
			}
		}
	}

	if output.Summary != "" {
		r.println()
		r.println(output.Summary)
	}

	return nil
}

func (r *TextRenderer) print(text string) {
	_, _ = fmt.Fprint(r.Writer, text)
}

func (r *TextRenderer) printf(format string, args ...any) {
	_, _ = fmt.Fprintf(r.Writer, format, args...)
}

func (r *TextRenderer) println(args ...any) {
	_, _ = fmt.Fprintln(r.Writer, args...)
}

// JSONRenderer renders output as JSON.
type JSONRenderer struct {
	Writer io.Writer
}

// NewJSONRenderer creates a JSONRenderer that writes to stdout.
func NewJSONRenderer() *JSONRenderer {
	return &JSONRenderer{Writer: os.Stdout}
}

// Render outputs the preview results as JSON.
func (r JSONRenderer) Render(output apply.Output) error {
	encoder := json.NewEncoder(r.Writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// plural returns the singular or plural form of a word based on count.
func plural(word string, count int) string {
	if count == 1 {
		return word
	}
	return word + "s"
}
