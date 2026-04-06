package incus

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// immutableFields are top-level fields that cannot be changed after creation.
// These are read-only properties defined by Incus.
var immutableFields = map[string]bool{
	"name":         true, // Can only be changed via `incus rename`
	"project":      true,
	"type":         true, // Instance type (container or vm)
	"created_at":   true,
	"last_used_at": true,
	"stateful":     true,
	"status":       true,
	"status_code":  true,
	"architecture": true, // Set from image at creation
	"location":     true, // Cluster location
	"used_by":      true, // Incus-managed: list of resources referencing this resource
}

// Diff generates a human-readable diff between current and desired YAML configurations.
// Returns an empty string if configurations are identical.
// Only shows the top-level keys that have changes.
// Excludes immutable fields and incus-managed config keys (volatile.*).
func Diff(current, desired string) (string, error) {
	return DiffWithIndent(current, desired, "  ")
}

// DiffChanges returns structured diff changes between current and desired YAML.
// Returns nil if configurations are identical.
func DiffChanges(current, desired string) ([]DiffChange, error) {
	currentMap, err := parseYAMLToMap(current, "current config")
	if err != nil {
		return nil, err
	}
	desiredMap, err := parseYAMLToMap(desired, "desired config")
	if err != nil {
		return nil, err
	}
	raw := findChanges(currentMap, desiredMap, "")
	if len(raw) == 0 {
		return nil, nil
	}
	result := make([]DiffChange, len(raw))
	for i, c := range raw {
		dc := DiffChange{Path: c.path}
		switch {
		case c.isAdd:
			dc.Action = "add"
			dc.New = c.newValue
		case c.isRemove:
			dc.Action = "remove"
			dc.Old = c.oldValue
		default:
			dc.Action = "modify"
			dc.Old = c.oldValue
			dc.New = c.newValue
		}
		result[i] = dc
	}
	return result, nil
}

// DiffWithIndent computes a diff with a custom indentation prefix.
// The indent is prepended to each line of the output.
func DiffWithIndent(current, desired, indent string) (string, error) {
	currentMap, err := parseYAMLToMap(current, "current config")
	if err != nil {
		return "", err
	}

	desiredMap, err := parseYAMLToMap(desired, "desired config")
	if err != nil {
		return "", err
	}

	changes := findChanges(currentMap, desiredMap, "")
	if len(changes) == 0 {
		return "", nil
	}

	return formatChanges(changes, indent, defaultMaxInlineDiffWidth, false), nil
}

// HasChanges checks if there are any differences between current and desired config.
func HasChanges(current, desired string) (bool, error) {
	diff, err := Diff(current, desired)
	if err != nil {
		return false, err
	}
	return diff != "", nil
}

// parseYAMLToMap unmarshals YAML into a map for comparison.
func parseYAMLToMap(data, label string) (map[string]any, error) {
	var m map[string]any
	if err := yaml.Unmarshal([]byte(data), &m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", label, err)
	}
	return m, nil
}

// change represents a single field change (internal).
type change struct {
	path     string
	oldValue any
	newValue any
	isAdd    bool
	isRemove bool
}

// DiffChange is an exported representation of a single field change.
type DiffChange struct {
	Path   string `json:"path"`
	Old    any    `json:"old,omitempty"`
	New    any    `json:"new,omitempty"`
	Action string `json:"action"` // "add", "remove", "modify"
}

// shouldSkipKey returns true if the key should be excluded from diff comparison.
func shouldSkipKey(prefix, key string) bool {
	// Skip immutable fields at top level
	if prefix == "" && immutableFields[key] {
		return true
	}
	// Skip Incus-managed and incus-apply tracking keys in config.
	if prefix == "config" && (strings.HasPrefix(key, "volatile.") || strings.HasPrefix(key, "image.") || strings.HasPrefix(key, "user.incus-apply.")) {
		return true
	}
	return false
}

// findChanges recursively finds differences between two maps.
// It skips immutable fields at the top level and volatile.* config keys.
func findChanges(current, desired map[string]any, prefix string) []change {
	var changes []change

	// Check all keys in desired (additions and modifications)
	desiredKeys := slices.Collect(maps.Keys(desired))
	slices.Sort(desiredKeys)
	for _, key := range desiredKeys {
		if shouldSkipKey(prefix, key) {
			continue
		}

		path := joinPath(prefix, key)
		desiredVal := desired[key]
		currentVal, exists := current[key]

		if !exists {
			changes = append(changes, change{path: path, newValue: desiredVal, isAdd: true})
			continue
		}

		// Both exist, compare values
		changes = append(changes, compareValues(path, currentVal, desiredVal)...)
	}

	// Check for removals (keys in current but not in desired)
	currentKeys := slices.Collect(maps.Keys(current))
	slices.Sort(currentKeys)
	for _, key := range currentKeys {
		if shouldSkipKey(prefix, key) {
			continue
		}

		if _, exists := desired[key]; !exists {
			path := joinPath(prefix, key)
			changes = append(changes, change{path: path, oldValue: current[key], isRemove: true})
		}
	}

	return changes
}

// compareValues compares two values and returns changes.
func compareValues(path string, current, desired any) []change {
	// If both are maps, recurse
	currentMap, currentIsMap := current.(map[string]any)
	desiredMap, desiredIsMap := desired.(map[string]any)
	if currentIsMap && desiredIsMap {
		return findChanges(currentMap, desiredMap, path)
	}

	currentSlice, currentIsSlice := current.([]any)
	desiredSlice, desiredIsSlice := desired.([]any)
	if currentIsSlice && desiredIsSlice {
		return compareSlices(path, currentSlice, desiredSlice)
	}

	// If values are equal, no change
	if fmt.Sprintf("%v", current) == fmt.Sprintf("%v", desired) {
		return nil
	}

	// Values differ
	return []change{{path: path, oldValue: current, newValue: desired}}
}

func compareSlices(path string, current, desired []any) []change {
	var changes []change

	maxLen := len(current)
	if len(desired) > maxLen {
		maxLen = len(desired)
	}

	for i := 0; i < maxLen; i++ {
		indexPath := joinIndexPath(path, i)

		switch {
		case i >= len(current):
			changes = append(changes, change{path: indexPath, newValue: desired[i], isAdd: true})
		case i >= len(desired):
			changes = append(changes, change{path: indexPath, oldValue: current[i], isRemove: true})
		default:
			changes = append(changes, compareValues(indexPath, current[i], desired[i])...)
		}
	}

	return changes
}

// joinPath joins path segments with a dot.
func joinPath(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

func joinIndexPath(prefix string, index int) string {
	return fmt.Sprintf("%s[%d]", prefix, index)
}
