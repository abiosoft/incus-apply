package incus

import (
	"fmt"
	"maps"

	"github.com/abiosoft/incus-apply/internal/config"
	"github.com/abiosoft/incus-apply/internal/resource"
	"gopkg.in/yaml.v3"
)

// mergeConfigs merges the desired config with the current config.
// It preserves incus-managed fields (volatile.*, architecture, etc.)
// while applying user-specified changes.
func mergeConfigs(currentYAML string, desired *config.Resource) ([]byte, error) {
	merged, _, err := mergedConfigWithStatus(currentYAML, desired)
	return merged, err
}

// legacyMergeConfigs applies the pre-managed-state merge strategy.
// It is retained as a fallback for resources not previously created by incus-apply.
func legacyMergeConfigs(currentYAML string, desired *config.Resource) ([]byte, error) {
	var current map[string]any
	if err := yaml.Unmarshal([]byte(currentYAML), &current); err != nil {
		return nil, fmt.Errorf("parsing current config: %w", err)
	}

	// Start with the full current state as the base so that Incus-managed
	// fields (architecture, status, volatile.*, etc.) are preserved verbatim.
	merged := current

	if desired.Config != nil {
		// Merge at the key level rather than replacing the map wholesale.
		// This keeps volatile.* and image.* keys that Incus manages internally
		// while still applying every user-specified key on top.
		currentConfig, _ := current["config"].(map[string]any)
		if currentConfig == nil {
			currentConfig = make(map[string]any)
		}
		mergedConfig := make(map[string]any)
		maps.Copy(mergedConfig, currentConfig) // copy all current keys (incl. volatile.*)
		for k, v := range desired.Config {
			mergedConfig[k] = v // overlay desired keys
		}
		merged["config"] = mergedConfig
	}

	// Devices are fully user-controlled: replace the current set entirely.
	if desired.Devices != nil {
		merged["devices"] = desired.Devices
	}

	if desired.Description != "" {
		merged["description"] = desired.Description
	}

	// Profiles are instance-only; only replace when the user has specified them.
	if resource.Type(desired.Type) == resource.TypeInstance && len(desired.Profiles) > 0 {
		merged["profiles"] = desired.Profiles
	}

	// Network ACL ingress/egress rules are replaced per-direction when provided.
	if resource.Type(desired.Type) == resource.TypeNetworkACL {
		if len(desired.Ingress) > 0 {
			merged["ingress"] = desired.Ingress
		}
		if len(desired.Egress) > 0 {
			merged["egress"] = desired.Egress
		}
	}

	if resource.Type(desired.Type) == resource.TypeNetworkForward && desired.Ports != nil {
		merged["ports"] = desired.Ports
	}

	cleanMap(merged)
	return yaml.Marshal(merged)
}

// cleanMap removes nil values from a map recursively.
// Empty maps, slices, and strings are preserved to avoid spurious diffs.
func cleanMap(m map[string]any) {
	for k, v := range m {
		if v == nil {
			delete(m, k)
			continue
		}
		if subMap, ok := v.(map[string]any); ok {
			cleanMap(subMap)
		}
	}
}
