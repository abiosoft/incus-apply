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

// cloudInitConfigSchema is the schema for the `config` map on resource types
// that support cloud-init (instance, profile). Keys beginning with
// "cloud-init." accept either a plain string or an inline YAML mapping/sequence
// (which incus-apply serialises to a string at parse time); all other keys
// must be strings.
var cloudInitConfigSchema = &Schema{
	Type:        "object",
	Description: "Key-value configuration options",
	PatternProperties: map[string]*Schema{
		`^cloud-init\.`: {
			Description: "cloud-init configuration value — may be a plain string or an inline YAML mapping/sequence. incus-apply converts inline YAML to the string form that Incus expects, preserving comments such as #cloud-config.",
			OneOf: []Schema{
				{Type: "string"},
				{Type: "object"},
			},
		},
		`^(?!cloud-init\.)`: {
			Type:        "string",
			Description: "Configuration value",
		},
	},
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

	// The schema only enforces constraints when `kind` is present and matches a
	// known incus-apply value. This allows the schema to be applied broadly
	// (e.g. to all *.yaml files) without flagging unrelated YAML documents.
	return Schema{
		Schema:      "https://json-schema.org/draft/2020-12/schema",
		ID:          "https://raw.githubusercontent.com/abiosoft/incus-apply/refs/heads/main/schema/incus-apply.schema.json",
		Title:       "incus-apply configuration",
		Description: "Schema for incus-apply configuration files. Each YAML document in the file is either a resource definition (identified by a supported `kind`) or a vars declaration. Documents without a recognized `kind` value are unconstrained.",
		If: &Schema{
			Properties: map[string]*Schema{
				"kind": {
					Enum: allTypes,
				},
			},
			Required: []string{"kind"},
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

// storageContentTypeEnum is the set of valid --type values for storage volume / bucket create.
var storageContentTypeEnum = []string{"block", "filesystem"}

func generateResourceSchema(resourceTypes []string) Schema {
	properties := structProperties(reflect.TypeOf(config.Resource{}))
	// Remove the ContentType field (yaml:"type") from the base properties;
	// it is added only to storage-volume and storage-bucket variants below.
	delete(properties, "type")

	var variants []Schema
	for _, resourceType := range resourceTypes {
		required := []string{"kind", "name"}
		if resourceType == string(resource.TypeNetworkForward) {
			required = []string{"kind", "listen_address", "network"}
		}

		variantProperties := cloneProperties(properties)
		variantProperties["kind"] = &Schema{
			Type:        "string",
			Description: "Resource kind",
			Enum:        []string{resourceType},
		}

		// storage-volume and storage-bucket accept an optional `type` field that
		// maps to the --type flag of `incus storage volume create`.
		switch resource.Type(resourceType) {
		case resource.TypeStorageVolume, resource.TypeStorageBucket:
			variantProperties["type"] = &Schema{
				Type:        "string",
				Description: "Storage content type (passed as --type to incus storage volume/bucket create).",
				Enum:        storageContentTypeEnum,
			}
		}

		// instance and profile support cloud-init config keys whose values may
		// be inline YAML mappings in addition to plain strings.
		switch resource.Type(resourceType) {
		case resource.TypeInstance, resource.TypeProfile:
			cp := *cloudInitConfigSchema
			variantProperties["config"] = &cp
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
			"kind": {
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
			"computed": {
				Type:        "object",
				Description: "Computed variable definitions resolved at load time by running a command or reading a file.",
				PatternProperties: map[string]*Schema{
					".*": generateComputedEntrySchema(),
				},
				AdditionalProperties: &falseVal,
			},
			"files": {
				Type:        "array",
				Description: "Paths to .env files to load variables from.",
				Items:       &Schema{Type: "string"},
			},
			"global": {
				Type:        "boolean",
				Description: "If true, variables are shared across all files instead of being file-scoped.",
			},
		},
		Required:             []string{"kind"},
		AdditionalProperties: &falseVal,
	}
}

func generateComputedEntrySchema() *Schema {
	falseVal := false
	return &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"file": {
				Type:        "string",
				Description: "Read the file at this path as the variable value.",
			},
			"incus": {
				Type:        "string",
				Description: "Run `incus <args>` and use stdout as the variable value.",
			},
			"format": {
				Type:        "string",
				Description: "Optional output format transformation. Supported: base64.",
				Enum:        []string{"base64"},
			},
		},
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
		"After":         "List of instance names that must be applied before this one",
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
		"Source":        "Source path or device for a storage pool",
		"Force":         "Force the action without a clean shutdown",
		"Timeout":       "Timeout in seconds passed as --timeout to the incus command",
		"Ingress":       "Ingress firewall rules",
		"Egress":        "Egress firewall rules",
	}
	if desc, ok := descriptions[f.Name]; ok {
		return desc
	}
	return ""
}

func fieldEnum(f reflect.StructField) []string {
	return nil
}
