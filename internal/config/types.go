// Package config handles parsing and validation of incus-apply configuration files.
package config

import (
	"fmt"
)

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
	Image     string        `yaml:"image,omitempty" json:"image,omitempty"`
	VM        bool          `yaml:"vm,omitempty" json:"vm,omitempty"`
	Empty     bool          `yaml:"empty,omitempty" json:"empty,omitempty"`
	Ephemeral bool          `yaml:"ephemeral,omitempty" json:"ephemeral,omitempty"`
	Profiles  []string      `yaml:"profiles,omitempty" json:"profiles,omitempty"`
	Storage   string        `yaml:"storage,omitempty" json:"storage,omitempty"`
	Network   string        `yaml:"network,omitempty" json:"network,omitempty"`
	Target    string        `yaml:"target,omitempty" json:"target,omitempty"`
	After     []string      `yaml:"after,omitempty" json:"after,omitempty"`
	Setup     []SetupAction `yaml:"setup,omitempty" json:"setup,omitempty"`
}

// SetupActionType identifies the supported setup action kinds.
type SetupActionType string

const (
	SetupActionExec     SetupActionType = "exec"
	SetupActionPushFile SetupActionType = "file_push"
	SetupActionRestart  SetupActionType = "restart"
	SetupActionStop     SetupActionType = "stop"
)

// SetupWhen controls when a setup action runs during apply.
type SetupWhen string

const (
	SetupWhenCreate SetupWhen = "create"
	SetupWhenUpdate SetupWhen = "update"
	SetupWhenAlways SetupWhen = "always"
)

// SetupAction defines an imperative action to run against an instance.
type SetupAction struct {
	Action    SetupActionType `yaml:"action" json:"action"`
	When      SetupWhen       `yaml:"when" json:"when"`
	Required  *bool           `yaml:"required,omitempty" json:"required,omitempty"`
	Skip      bool            `yaml:"skip,omitempty" json:"skip,omitempty"`
	Force     bool            `yaml:"force,omitempty" json:"force,omitempty"`
	Script    string          `yaml:"script,omitempty" json:"script,omitempty"`
	CWD       string          `yaml:"cwd,omitempty" json:"cwd,omitempty"`
	Path      string          `yaml:"path,omitempty" json:"path,omitempty"`
	Content   string          `yaml:"content,omitempty" json:"content,omitempty"`
	Source    string          `yaml:"source,omitempty" json:"source,omitempty"`
	Recursive bool            `yaml:"recursive,omitempty" json:"recursive,omitempty"`
	UID       *int            `yaml:"uid,omitempty" json:"uid,omitempty"`
	GID       *int            `yaml:"gid,omitempty" json:"gid,omitempty"`
	Mode      string          `yaml:"mode,omitempty" json:"mode,omitempty"`
}

// IsRequired reports whether setup failure should stop apply.
func (a SetupAction) IsRequired() bool {
	return a.Required == nil || *a.Required
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
	for i := range r.Setup {
		if r.Setup[i].When == "" {
			r.Setup[i].When = SetupWhenCreate
		}
	}
}

// Validate checks if required fields are present in the resource configuration.
func (r Resource) Validate() error {
	if r.Type == "" {
		return &ValidationError{Field: "type", Message: "type is required"}
	}
	if len(r.Setup) > 0 && r.Type != "instance" {
		return &ValidationError{Field: "setup", Message: "setup is only supported for instances"}
	}
	if len(r.After) > 0 && r.Type != "instance" {
		return &ValidationError{Field: "after", Message: "after is only supported for instances"}
	}
	if r.Type == "network-forward" {
		if r.ListenAddress == "" {
			return &ValidationError{Field: "listen_address", Message: "listen_address is required"}
		}
	} else if r.Type != "vars" && r.Name == "" {
		return &ValidationError{Field: "name", Message: "name is required"}
	}
	for i, action := range r.Setup {
		if err := action.Validate(i); err != nil {
			return err
		}
	}
	return nil
}

func (a SetupAction) Validate(index int) error {
	field := func(name string) string {
		return fmt.Sprintf("setup[%d].%s", index, name)
	}

	switch a.When {
	case SetupWhenCreate, SetupWhenUpdate, SetupWhenAlways:
	case "":
		return &ValidationError{Field: field("when"), Message: "when is required"}
	default:
		return &ValidationError{Field: field("when"), Message: "when must be one of create, update, always"}
	}

	switch a.Action {
	case SetupActionExec:
		if a.Script == "" {
			return &ValidationError{Field: field("script"), Message: "script is required for exec actions"}
		}
	case SetupActionPushFile:
		if a.Path == "" {
			return &ValidationError{Field: field("path"), Message: "path is required for file_push actions"}
		}
		if a.Path[0] != '/' {
			return &ValidationError{Field: field("path"), Message: "path must be absolute"}
		}
		if a.Content != "" && a.Source != "" {
			return &ValidationError{Field: field("content"), Message: "content and source are mutually exclusive for file_push actions"}
		}
		if a.Recursive && a.Source == "" {
			return &ValidationError{Field: field("recursive"), Message: "recursive is only supported when source is set"}
		}
	case SetupActionRestart:
		// no required fields; force is optional
	case SetupActionStop:
		// no required fields; force is optional
	default:
		return &ValidationError{Field: field("action"), Message: "action must be one of exec, file_push, restart, stop"}
	}

	return nil
}

// Vars represents a `type: vars` document that declares variables
// for interpolation in resource configs within the same file (or globally).
type Vars struct {
	Vars     map[string]string `yaml:"vars,omitempty"`
	Files    []string          `yaml:"files,omitempty"`    // .env files to load
	Commands map[string]string `yaml:"commands,omitempty"` // shell commands whose stdout becomes the value
	Global   bool              `yaml:"global,omitempty"`

	SourceFile string `yaml:"-"`
}

// ValidationError represents a field-level configuration validation error.
type ValidationError struct {
	Field   string // The field that failed validation
	Message string // Description of the validation failure
}

func (e ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
