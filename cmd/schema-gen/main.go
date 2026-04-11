// Command schema-gen generates JSON Schema for incus-apply configuration files.
//
// Usage:
//
//	go run ./cmd/schema-gen > schema/incus-apply.schema.json
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/abiosoft/incus-apply/internal/config"
	"github.com/abiosoft/incus-apply/internal/resource"
)

// Schema represents a JSON Schema document.
type Schema struct {
	Schema            string             `json:"$schema,omitempty"`
	ID                string             `json:"$id,omitempty"`
	Title             string             `json:"title,omitempty"`
	Description       string             `json:"description,omitempty"`
	Type              string             `json:"type,omitempty"`
	If                *Schema            `json:"if,omitempty"`
	Then              *Schema            `json:"then,omitempty"`
	OneOf             []Schema           `json:"oneOf,omitempty"`
	Properties        map[string]*Schema `json:"properties,omitempty"`
	Required          []string           `json:"required,omitempty"`
	Items             *Schema            `json:"items,omitempty"`
	Enum              []string           `json:"enum,omitempty"`
	PatternProperties map[string]*Schema `json:"patternProperties,omitempty"`

	AdditionalProperties *bool `json:"additionalProperties,omitempty"`
}

func main() {
	schema := generateRootSchema()

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(schema); err != nil {
		fmt.Fprintf(os.Stderr, "error encoding schema: %v\n", err)
		os.Exit(1)
	}
}

func generateRootSchema() Schema {
	resourceTypes := collectResourceTypes()
	allTypes := append(resourceTypes, "vars")

	// The schema only enforces constraints when `type` is present and matches a
	// known incus-apply value. This allows the schema to be applied broadly
	// (e.g. to all *.yaml files) without flagging unrelated YAML documents.
	return Schema{
		Schema:      "https://json-schema.org/draft/2020-12/schema",
		ID:          "https://raw.githubusercontent.com/abiosoft/incus-apply/refs/heads/main/schema/incus-apply.schema.json",
		Title:       "incus-apply configuration",
		Description: "Schema for incus-apply configuration files. Each YAML document in the file is either a resource definition (identified by a supported `type`) or a vars declaration. Documents without a recognized `type` value are unconstrained.",
		If: &Schema{
			Properties: map[string]*Schema{
				"type": {
					Enum: allTypes,
				},
			},
			Required: []string{"type"},
		},
		Then: &Schema{
			OneOf: []Schema{
				generateResourceSchema(resourceTypes),
				generateVarsSchema(),
			},
		},
	}
}

func collectResourceTypes() []string {
	var types []string
	for _, t := range []resource.Type{
		resource.TypeInstance,
		resource.TypeProfile,
		resource.TypeNetwork,
		resource.TypeNetworkForward,
		resource.TypeNetworkACL,
		resource.TypeNetworkZone,
		resource.TypeStoragePool,
		resource.TypeStorageVolume,
		resource.TypeStorageBucket,
		resource.TypeProject,
		resource.TypeClusterGroup,
	} {
		types = append(types, string(t))
	}
	return types
}

func generateResourceSchema(resourceTypes []string) Schema {
	properties := structProperties(reflect.TypeOf(config.Resource{}))

	var variants []Schema
	for _, resourceType := range resourceTypes {
		required := []string{"type", "name"}
		if resourceType == string(resource.TypeNetworkForward) {
			required = []string{"type", "listen_address", "network"}
		}

		variantProperties := cloneProperties(properties)
		variantProperties["type"] = &Schema{
			Type:        "string",
			Description: "Resource type",
			Enum:        []string{resourceType},
		}

		variants = append(variants, Schema{
			Type:        "object",
			Description: "An Incus resource definition.",
			Properties:  variantProperties,
			Required:    required,
		})
	}

	return Schema{
		OneOf: variants,
	}
}

func cloneProperties(properties map[string]*Schema) map[string]*Schema {
	cloned := make(map[string]*Schema, len(properties))
	for key, value := range properties {
		copy := *value
		cloned[key] = &copy
	}
	return cloned
}

func generateVarsSchema() Schema {
	falseVal := false
	return Schema{
		Type:        "object",
		Description: "Variable declarations for interpolation in resource configs.",
		Properties: map[string]*Schema{
			"type": {
				Type:        "string",
				Description: "Must be 'vars' for variable declarations.",
				Enum:        []string{"vars"},
			},
			"vars": {
				Type:        "object",
				Description: "Inline variable definitions (key-value pairs).",
				PatternProperties: map[string]*Schema{
					".*": {Type: "string"},
				},
				AdditionalProperties: &falseVal,
			},
			"files": {
				Type:        "array",
				Description: "Paths to .env files to load variables from.",
				Items:       &Schema{Type: "string"},
			},
			"commands": {
				Type:        "object",
				Description: "Shell commands whose stdout is used as the variable value (key-value pairs).",
				PatternProperties: map[string]*Schema{
					".*": {Type: "string"},
				},
				AdditionalProperties: &falseVal,
			},
			"global": {
				Type:        "boolean",
				Description: "If true, variables are shared across all files instead of being file-scoped.",
			},
		},
		Required:             []string{"type"},
		AdditionalProperties: &falseVal,
	}
}

func structProperties(t reflect.Type) map[string]*Schema {
	properties := make(map[string]*Schema)

	for i := range t.NumField() {
		field := t.Field(i)

		if field.Anonymous {
			for k, v := range structProperties(field.Type) {
				properties[k] = v
			}
			continue
		}

		yamlTag := field.Tag.Get("yaml")
		if yamlTag == "-" || yamlTag == "" {
			continue
		}

		name, _ := strings.CutSuffix(yamlTag, ",omitempty")
		description := fieldDescription(field)
		schema := goTypeToSchema(field.Type, description)
		if enum := fieldEnum(field); len(enum) > 0 {
			schema.Enum = enum
		}
		properties[name] = schema
	}

	return properties
}

func goTypeToSchema(t reflect.Type, description string) *Schema {
	if t.Kind() == reflect.Pointer {
		return goTypeToSchema(t.Elem(), description)
	}

	switch t.Kind() {
	case reflect.String:
		return &Schema{Type: "string", Description: description}
	case reflect.Bool:
		return &Schema{Type: "boolean", Description: description}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &Schema{Type: "integer", Description: description}
	case reflect.Float32, reflect.Float64:
		return &Schema{Type: "number", Description: description}
	case reflect.Slice:
		items := goTypeToSchema(t.Elem(), "")
		return &Schema{Type: "array", Description: description, Items: items}
	case reflect.Map:
		s := &Schema{Type: "object", Description: description}
		if t.Key().Kind() == reflect.String {
			valueSchema := goTypeToSchema(t.Elem(), "")
			s.PatternProperties = map[string]*Schema{
				".*": valueSchema,
			}
		}
		return s
	case reflect.Struct:
		return &Schema{Type: "object", Description: description, Properties: structProperties(t)}
	case reflect.Interface:
		return &Schema{Description: description}
	default:
		return &Schema{Description: description}
	}
}

func fieldDescription(f reflect.StructField) string {
	descriptions := map[string]string{
		"Type":          "Resource type",
		"Name":          "Resource name (unique within type)",
		"ListenAddress": "External listen address for a network forward",
		"Project":       "Incus project (can be overridden by --project flag)",
		"Config":        "Key-value configuration options",
		"Devices":       "Device configurations",
		"Description":   "Resource description",
		"Image":         "Image source for instances (e.g., images:debian/12, docker:caddy)",
		"Setup":         "Setup actions to run against an instance after create, update, or on every apply",
		"Action":        "Setup action type",
		"When":          "When to run the setup action: create, update, or always",
		"Required":      "If false, setup failures are reported and apply continues with later setup actions",
		"Skip":          "If true, the setup action is skipped",
		"Script":        "Shell script to execute inside the instance",
		"CWD":           "Working directory to use for an exec setup action",
		"Path":          "Absolute destination path inside the instance for a file_push action",
		"Content":       "Inline file content to push into the instance",
		"UID":           "File owner uid to set when pushing a file",
		"GID":           "File owner gid to set when pushing a file",
		"Mode":          "File mode to set when pushing a file",
		"VM":            "Create a virtual machine instead of a container",
		"Empty":         "Create an empty instance (no image)",
		"Profiles":      "Profiles to apply to the instance",
		"Storage":       "Storage pool for the instance root disk",
		"Network":       "Network name (for instances or the parent network of a network forward)",
		"Target":        "Cluster member target",
		"Pool":          "Storage pool name (required for storage volumes/buckets)",
		"Ports":         "Optional network forward port rules in the same shape as incus network forward edit",
		"NetworkType":   "Network type (bridge, ovn, macvlan, sriov, physical)",
		"Driver":        "Storage driver (dir, zfs, btrfs, lvm, ceph)",
		"Source":        "Source path or device for a storage pool, or a local file path for file_push setup actions",
		"Recursive":     "Pass --recursive to incus file push for a file_push setup action",
		"Force":         "Force the instance to restart without a clean shutdown (restart action only)",
		"Ingress":       "Ingress firewall rules",
		"Egress":        "Egress firewall rules",
	}
	if desc, ok := descriptions[f.Name]; ok {
		return desc
	}
	return ""
}

func fieldEnum(f reflect.StructField) []string {
	enums := map[string][]string{
		"Action": {
			string(config.SetupActionExec),
			string(config.SetupActionPushFile),
			string(config.SetupActionRestart),
			string(config.SetupActionStop),
		},
		"When": {
			string(config.SetupWhenCreate),
			string(config.SetupWhenUpdate),
			string(config.SetupWhenAlways),
		},
	}
	if vals, ok := enums[f.Name]; ok {
		return vals
	}
	return nil
}
