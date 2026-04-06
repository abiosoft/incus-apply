package incus

import (
	"fmt"
	"strings"

	"github.com/abiosoft/incus-apply/internal/terminal"
)

var (
	colorRed   = terminal.ColorRed
	colorGreen = terminal.ColorGreen
	colorReset = terminal.ColorReset
)

const (
	// Tree characters for visual hierarchy
	treeBranch = "├─"
	treeLast   = "└─"

	defaultMaxInlineDiffWidth = 100
)

// FormatDiffChanges renders a []DiffChange to a human-readable string with the given indent.
func FormatDiffChanges(changes []DiffChange, indent string) string {
	return FormatDiffChangesWithWidth(changes, indent, defaultMaxInlineDiffWidth)
}

// FormatDiffChangesWithWidth renders a []DiffChange to a human-readable string with the given indent and max inline width.
func FormatDiffChangesWithWidth(changes []DiffChange, indent string, maxWidth int) string {
	return FormatDiffChangesWithWidthAndTrailing(changes, indent, maxWidth, false)
}

// FormatDiffChangesWithWidthAndTrailing renders a []DiffChange to a human-readable string with the given indent and max inline width.
// When hasTrailingLine is true, the final diff entry uses a branch connector because another sibling line will follow.
func FormatDiffChangesWithWidthAndTrailing(changes []DiffChange, indent string, maxWidth int, hasTrailingLine bool) string {
	if maxWidth <= 0 {
		maxWidth = defaultMaxInlineDiffWidth
	}
	raw := make([]change, len(changes))
	for i, dc := range changes {
		raw[i] = change{
			path:     dc.Path,
			oldValue: dc.Old,
			newValue: dc.New,
			isAdd:    dc.Action == "add",
			isRemove: dc.Action == "remove",
		}
	}
	return formatChanges(raw, indent, maxWidth, hasTrailingLine)
}

// formatChanges formats the list of changes for display with tree structure.
// The indent prefix is prepended to each line.
func formatChanges(changes []change, indent string, maxWidth int, hasTrailingLine bool) string {
	var result strings.Builder

	for i, c := range changes {
		tree := treeBranch
		if i == len(changes)-1 && !hasTrailingLine {
			tree = treeLast
		}

		if summary, ok := formatLargeValueSummary(c, indent, tree, maxWidth); ok {
			result.WriteString(summary)
			continue
		}

		if c.isAdd {
			fmt.Fprintf(&result, "%s%s %s%s: %s%s\n", indent, tree, colorGreen, c.path, formatValue(c.newValue), colorReset)
		} else if c.isRemove {
			fmt.Fprintf(&result, "%s%s %s%s: %s%s\n", indent, tree, colorRed, c.path, formatValue(c.oldValue), colorReset)
		} else {
			fmt.Fprintf(&result, "%s%s %s: %s%s%s → %s%s%s\n",
				indent, tree, c.path,
				colorRed, formatValue(c.oldValue), colorReset,
				colorGreen, formatValue(c.newValue), colorReset)
		}
	}

	return result.String()
}

func formatLargeValueSummary(c change, indent, tree string, maxWidth int) (string, bool) {
	oldText, oldIsString := c.oldValue.(string)
	newText, newIsString := c.newValue.(string)

	switch {
	case c.isAdd && newIsString && shouldSummarizeStringChange(c.path, indent, tree, "", newText, true, false, maxWidth):
		return fmt.Sprintf("%s%s %s%s: %s%s\n",
			indent, tree, colorGreen, c.path,
			formatLargeValue(newText), colorReset), true
	case c.isRemove && oldIsString && shouldSummarizeStringChange(c.path, indent, tree, oldText, "", false, true, maxWidth):
		return fmt.Sprintf("%s%s %s%s: %s%s\n",
			indent, tree, colorRed, c.path,
			formatLargeValue(oldText), colorReset), true
	case !c.isAdd && !c.isRemove && oldIsString && newIsString && shouldSummarizeStringChange(c.path, indent, tree, oldText, newText, false, false, maxWidth):
		return fmt.Sprintf("%s%s %s: %s%s%s → %s%s%s\n",
			indent, tree, c.path,
			colorRed, formatLargeValue(oldText), colorReset,
			colorGreen, formatLargeValue(newText), colorReset), true
	default:
		return "", false
	}
}

func shouldSummarizeStringChange(path, indent, tree, oldValue, newValue string, isAdd, isRemove bool, maxWidth int) bool {
	if strings.Contains(oldValue, "\n") || strings.Contains(newValue, "\n") {
		return true
	}

	if isAdd {
		return estimateAddRemoveWidth(path, indent, tree, newValue) > maxWidth
	}
	if isRemove {
		return estimateAddRemoveWidth(path, indent, tree, oldValue) > maxWidth
	}
	return estimateModifyWidth(path, indent, tree, oldValue, newValue) > maxWidth
}

func estimateAddRemoveWidth(path, indent, tree, value string) int {
	return len(indent) + len(tree) + 1 + len(path) + 2 + len(formatValue(value))
}

func estimateModifyWidth(path, indent, tree, oldValue, newValue string) int {
	return len(indent) + len(tree) + 1 + len(path) + 2 + len(formatValue(oldValue)) + 3 + len(formatValue(newValue))
}

func formatLargeValue(value string) string {
	return fmt.Sprintf("<%d chars>", len(value))
}

// formatValue formats a value for display.
func formatValue(v any) string {
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%q", val)
	case map[string]any:
		return "{...}"
	case []any:
		items := make([]string, len(val))
		for i, item := range val {
			items[i] = formatValue(item)
		}
		return "[" + strings.Join(items, ", ") + "]"
	default:
		return fmt.Sprintf("%v", val)
	}
}
