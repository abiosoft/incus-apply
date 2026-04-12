// Package config handles parsing and validation of incus-apply configuration files.
package config

// Base contains fields common to all Incus resource types.
type Base struct {
	Type        string                    `yaml:"type" json:"type"`                                   // Resource type: instance, profile, network, etc.
	Name        string                    `yaml:"name" json:"name"`                                   // Resource name (unique within type)
	Project     string                    `yaml:"project,omitempty" json:"project,omitempty"`         // --project flag (can be overridden by CLI)
	Config      map[string]string         `yaml:"config,omitempty" json:"config,omitempty"`           // Key-value config options
	Devices     map[string]map[string]any `yaml:"devices,omitempty" json:"devices,omitempty"`         // Device configurations. Kept here for simplicity, only instances and profiles support devices.
	Description string                    `yaml:"description,omitempty" json:"description,omitempty"` // Resource description
	SourceFile  string                    `yaml:"-" json:"-"`                                         // Path to source file (set during parsing)
}

// Resource represents a single resource configuration from a .yaml file.
// It embeds common fields plus the resource-specific field groups used
// based on the resource Type.
type Resource struct {
	Base                  `yaml:",inline"`
	InstanceFields        `yaml:",inline"`
	StoragePoolFields     `yaml:",inline"`
	StorageResourceFields `yaml:",inline"`
	NetworkFields         `yaml:",inline"`
	NetworkACLFields      `yaml:",inline"`
	NetworkForwardFields  `yaml:",inline"`

	PreviewRedactPrefixes []string `yaml:"-" json:"-"`
}

// InstanceFields captures the fields specific to Incus instances.
type InstanceFields struct {
	Image     string   `yaml:"image,omitempty" json:"image,omitempty"`
	VM        bool     `yaml:"vm,omitempty" json:"vm,omitempty"`
	Empty     bool     `yaml:"empty,omitempty" json:"empty,omitempty"`
	Ephemeral bool     `yaml:"ephemeral,omitempty" json:"ephemeral,omitempty"`
	Profiles  []string `yaml:"profiles,omitempty" json:"profiles,omitempty"`
	Storage   string   `yaml:"storage,omitempty" json:"storage,omitempty"`
	Network   string   `yaml:"network,omitempty" json:"network,omitempty"`
	Target    string   `yaml:"target,omitempty" json:"target,omitempty"`
	After     []string `yaml:"apply.after,omitempty" json:"apply.after,omitempty"`
}

// StoragePoolFields captures the fields specific to storage pools.
type StoragePoolFields struct {
	Driver string `yaml:"driver,omitempty" json:"driver,omitempty"`
	Source string `yaml:"source,omitempty" json:"source,omitempty"`
}

// StorageResourceFields captures the fields specific to storage volumes and buckets.
type StorageResourceFields struct {
	Pool string `yaml:"pool,omitempty" json:"pool,omitempty"`
}

// NetworkFields captures the fields specific to networks.
type NetworkFields struct {
	NetworkType string `yaml:"networkType,omitempty" json:"networkType,omitempty"`
}

// NetworkACLFields captures the fields specific to network ACLs.
type NetworkACLFields struct {
	Ingress []map[string]any `yaml:"ingress,omitempty" json:"ingress,omitempty"`
	Egress  []map[string]any `yaml:"egress,omitempty" json:"egress,omitempty"`
}

// NetworkForwardFields captures the fields specific to network forwards.
type NetworkForwardFields struct {
	ListenAddress string           `yaml:"listen_address,omitempty" json:"listen_address,omitempty"`
	Ports         []map[string]any `yaml:"ports,omitempty" json:"ports,omitempty"`
}

// Stdin represents configuration data passed to incus commands via stdin.
// Incus edit commands accept YAML on stdin to modify resource configuration.
type Stdin struct {
	Config      map[string]string         `yaml:"config,omitempty"`
	Devices     map[string]map[string]any `yaml:"devices,omitempty"`
	Description string                    `yaml:"description,omitempty"`
	Profiles    []string                  `yaml:"profiles,omitempty"`
	Ingress     []map[string]any          `yaml:"ingress,omitempty"` // Network ACL ingress rules
	Egress      []map[string]any          `yaml:"egress,omitempty"`  // Network ACL egress rules
	Ports       []map[string]any          `yaml:"ports,omitempty"`   // Network forward port rules
}

// knownResourceTypes is the set of valid incus resource type strings.
// "vars" is intentionally excluded — it is handled separately by the parser.
var knownResourceTypes = map[string]struct{}{
	"instance":        {},
	"profile":         {},
	"network":         {},
	"network-forward": {},
	"network-acl":     {},
	"network-zone":    {},
	"storage-pool":    {},
	"storage-volume":  {},
	"storage-bucket":  {},
	"project":         {},
	"cluster-group":   {},
}

// isKnownResourceType reports whether s is a supported incus resource type.
func isKnownResourceType(s string) bool {
	_, ok := knownResourceTypes[s]
	return ok
}

// applyDefaults sets default values for optional fields.
func (r *Resource) applyDefaults() {
}

// Validate checks if required fields are present in the resource configuration.
func (r Resource) Validate() error {
	if r.Type == "" {
		return &ValidationError{Field: "type", Message: "type is required"}
	}
	if len(r.After) > 0 && r.Type != "instance" {
		return &ValidationError{Field: "apply.after", Message: "apply.after is only supported for instances"}
	}
	if r.Type == "network-forward" {
		if r.ListenAddress == "" {
			return &ValidationError{Field: "listen_address", Message: "listen_address is required"}
		}
	} else if r.Type != "vars" && r.Name == "" {
		return &ValidationError{Field: "name", Message: "name is required"}
	}
	return nil
}

// Vars represents a `type: vars` document that declares variables
// for interpolation in resource configs within the same file (or globally).
type Vars struct {
	Vars     map[string]string       `yaml:"vars,omitempty"`
	Computed map[string]DynamicEntry `yaml:"computed,omitempty"` // computed (dynamically resolved) variables
	Files    []string                `yaml:"files,omitempty"`    // .env files to load
	Global   bool                    `yaml:"global,omitempty"`

	SourceFile string `yaml:"-"`
}

// DynamicEntry defines how to resolve a single dynamic variable.
// Exactly one source processor (File, Incus) must be set.
// Format is applied to the raw output after resolution.
type DynamicEntry struct {
	File   string `yaml:"file,omitempty"`   // read the file at this path as the value
	Incus  string `yaml:"incus,omitempty"`  // run: incus <args> and use stdout as the value
	Format string `yaml:"format,omitempty"` // output format: "" (raw) or "base64"
}

// ValidationError represents a field-level configuration validation error.
type ValidationError struct {
	Field   string // The field that failed validation
	Message string // Description of the validation failure
}

func (e ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
