package resource

import (
	"fmt"
	"sync"
)

// Type represents an Incus resource type.
type Type string

// Built-in resource types supported by Incus.
const (
	TypeInstance       Type = "instance"
	TypeProfile        Type = "profile"
	TypeNetwork        Type = "network"
	TypeNetworkACL     Type = "network-acl"
	TypeNetworkForward Type = "network-forward"
	TypeNetworkZone    Type = "network-zone"
	TypeStoragePool    Type = "storage-pool"
	TypeStorageVolume  Type = "storage-volume"
	TypeStorageBucket  Type = "storage-bucket"
	TypeProject        Type = "project"
	TypeClusterGroup   Type = "cluster-group"

	// CLI command constants to avoid repetition and typos.
	cmdProject = "project"
	cmdStorage = "storage"
	cmdNetwork = "network"
	cmdProfile = "profile"
	cmdCluster = "cluster"
	cmdConfig  = "config"
	cmdCreate  = "create"
	cmdShow    = "show"
	cmdEdit    = "edit"
	cmdDelete  = "delete"
	cmdInit    = "init"
	cmdACL     = "acl"
	cmdForward = "forward"
	cmdZone    = "zone"
	cmdVolume  = "volume"
	cmdBucket  = "bucket"
	cmdGroup   = "group"
)

// TypeMeta contains metadata about a resource type including the CLI commands
// used to manage it and its priority for dependency ordering.
type TypeMeta struct {
	Type     Type
	Priority int      // Lower = created first, deleted last
	Create   []string // CLI subcommands for create (e.g., ["storage", "create"])
	Show     []string // CLI subcommands for show
	Edit     []string // CLI subcommands for edit
	Delete   []string // CLI subcommands for delete

	// PrependPool indicates that pool name should be prepended before the
	// resource name in show/edit/delete commands (storage volumes and buckets).
	PrependPool bool

	// PrependNetwork indicates that network name should be prepended before the
	// resource name in show/edit/delete commands (network forwards).
	PrependNetwork bool

	// StdinFields declares which extra fields (beyond config/devices/description)
	// should be included in the stdin YAML for create/edit commands.
	// Supported values: "profiles", "ingress", "egress".
	StdinFields []string
}

// Registry provides thread-safe, immutable access to resource type metadata
// with support for registering custom types at runtime.
type Registry struct {
	mu      sync.RWMutex
	custom  map[Type]TypeMeta // User-registered custom types
	builtin map[Type]TypeMeta // Immutable built-in types (reference only, never modified)
}

// builtinTypes contains the immutable set of default resource types.
// This map is never modified after initialization.
var builtinTypes = map[Type]TypeMeta{
	TypeProject: {
		Type:     TypeProject,
		Priority: 1,
		Create:   []string{cmdProject, cmdCreate},
		Show:     []string{cmdProject, cmdShow},
		Edit:     []string{cmdProject, cmdEdit},
		Delete:   []string{cmdProject, cmdDelete},
	},
	TypeStoragePool: {
		Type:     TypeStoragePool,
		Priority: 2,
		Create:   []string{cmdStorage, cmdCreate},
		Show:     []string{cmdStorage, cmdShow},
		Edit:     []string{cmdStorage, cmdEdit},
		Delete:   []string{cmdStorage, cmdDelete},
	},
	TypeNetwork: {
		Type:     TypeNetwork,
		Priority: 3,
		Create:   []string{cmdNetwork, cmdCreate},
		Show:     []string{cmdNetwork, cmdShow},
		Edit:     []string{cmdNetwork, cmdEdit},
		Delete:   []string{cmdNetwork, cmdDelete},
	},
	TypeNetworkForward: {
		Type:           TypeNetworkForward,
		Priority:       4,
		Create:         []string{cmdNetwork, cmdForward, cmdCreate},
		Show:           []string{cmdNetwork, cmdForward, cmdShow},
		Edit:           []string{cmdNetwork, cmdForward, cmdEdit},
		Delete:         []string{cmdNetwork, cmdForward, cmdDelete},
		PrependNetwork: true,
		StdinFields:    []string{"ports"},
	},
	TypeNetworkACL: {
		Type:        TypeNetworkACL,
		Priority:    5,
		Create:      []string{cmdNetwork, cmdACL, cmdCreate},
		Show:        []string{cmdNetwork, cmdACL, cmdShow},
		Edit:        []string{cmdNetwork, cmdACL, cmdEdit},
		Delete:      []string{cmdNetwork, cmdACL, cmdDelete},
		StdinFields: []string{"ingress", "egress"},
	},
	TypeNetworkZone: {
		Type:     TypeNetworkZone,
		Priority: 6,
		Create:   []string{cmdNetwork, cmdZone, cmdCreate},
		Show:     []string{cmdNetwork, cmdZone, cmdShow},
		Edit:     []string{cmdNetwork, cmdZone, cmdEdit},
		Delete:   []string{cmdNetwork, cmdZone, cmdDelete},
	},
	TypeStorageVolume: {
		Type:        TypeStorageVolume,
		Priority:    7,
		Create:      []string{cmdStorage, cmdVolume, cmdCreate},
		Show:        []string{cmdStorage, cmdVolume, cmdShow},
		Edit:        []string{cmdStorage, cmdVolume, cmdEdit},
		Delete:      []string{cmdStorage, cmdVolume, cmdDelete},
		PrependPool: true,
	},
	TypeStorageBucket: {
		Type:        TypeStorageBucket,
		Priority:    8,
		Create:      []string{cmdStorage, cmdBucket, cmdCreate},
		Show:        []string{cmdStorage, cmdBucket, cmdShow},
		Edit:        []string{cmdStorage, cmdBucket, cmdEdit},
		Delete:      []string{cmdStorage, cmdBucket, cmdDelete},
		PrependPool: true,
	},
	TypeClusterGroup: {
		Type:     TypeClusterGroup,
		Priority: 9,
		Create:   []string{cmdCluster, cmdGroup, cmdCreate},
		Show:     []string{cmdCluster, cmdGroup, cmdShow},
		Edit:     []string{cmdCluster, cmdGroup, cmdEdit},
		Delete:   []string{cmdCluster, cmdGroup, cmdDelete},
	},
	TypeProfile: {
		Type:     TypeProfile,
		Priority: 10,
		Create:   []string{cmdProfile, cmdCreate},
		Show:     []string{cmdProfile, cmdShow},
		Edit:     []string{cmdProfile, cmdEdit},
		Delete:   []string{cmdProfile, cmdDelete},
	},
	TypeInstance: {
		Type:        TypeInstance,
		Priority:    11,
		Create:      []string{cmdInit}, // Special: uses "init" not "instance create"
		Show:        []string{cmdConfig, cmdShow},
		Edit:        []string{cmdConfig, cmdEdit},
		Delete:      []string{cmdDelete},
		StdinFields: []string{"profiles"},
	},
}

// defaultRegistry is the package-level registry instance.
var defaultRegistry = &Registry{
	builtin: builtinTypes,
	custom:  make(map[Type]TypeMeta),
}

// Get retrieves metadata for a resource type.
// Returns the metadata and true if found, or zero value and false if not.
func (r *Registry) Get(t Type) (TypeMeta, bool) {
	// Check built-in types first (no lock needed, immutable)
	if meta, ok := r.builtin[t]; ok {
		return meta, true
	}

	// Check custom types (requires lock)
	r.mu.RLock()
	defer r.mu.RUnlock()
	meta, ok := r.custom[t]
	return meta, ok
}

// Register adds a custom resource type to the registry.
// Returns an error if the type already exists (built-in or custom).
func (r *Registry) Register(meta TypeMeta) error {
	// Prevent overriding built-in types
	if _, exists := r.builtin[meta.Type]; exists {
		return fmt.Errorf("cannot override built-in type: %s", meta.Type)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.custom[meta.Type]; exists {
		return fmt.Errorf("type already registered: %s", meta.Type)
	}

	r.custom[meta.Type] = meta
	return nil
}

// All returns all registered types (built-in and custom).
// Returns a new slice on each call; modifications do not affect the registry.
func (r *Registry) All() []Type {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]Type, 0, len(r.builtin)+len(r.custom))
	for t := range r.builtin {
		types = append(types, t)
	}
	for t := range r.custom {
		types = append(types, t)
	}
	return types
}

// IsValid checks if a type string is a valid registered resource type.
func (r *Registry) IsValid(t string) bool {
	_, ok := r.Get(Type(t))
	return ok
}

// --- Package-level convenience functions ---
// These functions operate on the default registry for backward compatibility.

// GetTypeMeta returns metadata for a resource type from the default registry.
func GetTypeMeta(t string) (TypeMeta, bool) {
	return defaultRegistry.Get(Type(t))
}

// RegisterType adds a custom resource type to the default registry.
func RegisterType(meta TypeMeta) error {
	return defaultRegistry.Register(meta)
}

// ValidTypes returns all valid resource type strings from the default registry.
func ValidTypes() []string {
	types := defaultRegistry.All()
	result := make([]string, len(types))
	for i, t := range types {
		result[i] = string(t)
	}
	return result
}

// IsValidType checks if a string is a valid resource type in the default registry.
func IsValidType(t string) bool {
	return defaultRegistry.IsValid(t)
}

// DefaultRegistry returns the package-level registry instance.
// Use this when you need direct registry access.
func DefaultRegistry() *Registry {
	return defaultRegistry
}
