package config

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// BoolVal is a string type that can be unmarshaled from bool or template variable strings.
// It accepts any string value during parsing to support template variables like "${VAR}".
// Validation happens after interpolation via ValidateBoolVal().
type BoolVal string

// UnmarshalYAML allows BoolVal to accept bool values or any string.
// String validation (for template variables) is deferred to after interpolation.
func (b *BoolVal) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		switch node.Tag {
		case "!!bool":
			// Boolean literal in YAML - convert to string
			*b = BoolVal(node.Value)
			return nil
		case "!!str", "":
			// String value - accept as-is (could be template variable or bool string)
			// Validation deferred to after interpolation
			*b = BoolVal(node.Value)
			return nil
		default:
			return fmt.Errorf("cannot unmarshal %s into bool", node.Tag)
		}
	default:
		return fmt.Errorf("cannot unmarshal %v into bool", node.Kind)
	}
}

// MarshalYAML converts BoolVal back to YAML as a boolean if it's a valid bool string.
func (b BoolVal) MarshalYAML() (any, error) {
	lower := strings.ToLower(string(b))
	if lower == "true" {
		return true, nil
	}
	if lower == "false" {
		return false, nil
	}
	// Return as is in case it is a template variable
	return string(b), nil
}

// Bool returns the boolean value.
func (b BoolVal) Bool() bool {
	return strings.ToLower(string(b)) == "true"
}

// String returns the string representation.
func (b BoolVal) String() string { return string(b) }

// ValidateBoolVal validates a BoolVal after interpolation.
// Returns an error if the value is not a valid boolean string.
func (b BoolVal) Validate() error {
	if b == "" {
		return nil // empty is acceptable (omitted field)
	}
	_, err := strconv.ParseBool(string(b))
	return err
}
