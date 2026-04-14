package apply

import (
	"fmt"
	"strings"

	"github.com/abiosoft/incus-apply/internal/config"
	"github.com/abiosoft/incus-apply/internal/resource"
)

// formatResourceID creates a scope-aware display identifier for a resource.
// The format is: [remote:][project:]type[/scope]/name
// where the remote and project segments are only shown when set.
func formatResourceID(res *config.Resource) string {
	resourcePath := res.Type
	if usesNetworkScope(resource.Type(res.Type)) && res.Network != "" {
		resourcePath += "/" + res.Network
	}
	if usesPoolScope(resource.Type(res.Type)) && res.Pool != "" {
		resourcePath += "/" + res.Pool
	}
	resourcePath += "/" + resourceIdentifier(res)

	if project := displayProject(res); project != "" {
		resourcePath = project + ":" + resourcePath
	}
	if res.Remote != "" {
		resourcePath = res.Remote + ":" + resourcePath
	}
	return resourcePath
}

func resourceIdentifier(res *config.Resource) string {
	if resource.Type(res.Type) == resource.TypeNetworkForward {
		return res.ListenAddress
	}
	return res.Name
}

func displayProject(res *config.Resource) string {
	if !usesProjectScope(resource.Type(res.Type)) {
		return ""
	}
	if strings.TrimSpace(res.Project) == "" {
		return "default"
	}
	return res.Project
}

func usesProjectScope(resourceType resource.Type) bool {
	switch resourceType {
	case resource.TypeProject, resource.TypeStoragePool, resource.TypeClusterGroup:
		return false
	default:
		return true
	}
}

func usesPoolScope(resourceType resource.Type) bool {
	switch resourceType {
	case resource.TypeStorageVolume, resource.TypeStorageBucket:
		return true
	default:
		return false
	}
}

func usesNetworkScope(resourceType resource.Type) bool {
	switch resourceType {
	case resource.TypeNetworkForward:
		return true
	default:
		return false
	}
}

func validateUniqueResources(resources []*config.Resource) error {
	seen := make(map[string]*config.Resource, len(resources))
	for _, res := range resources {
		key := formatResourceID(res)
		if previous, ok := seen[key]; ok {
			return fmt.Errorf("duplicate resource %q defined in %s and %s", key, previous.SourceFile, res.SourceFile)
		}
		seen[key] = res
	}
	return nil
}
