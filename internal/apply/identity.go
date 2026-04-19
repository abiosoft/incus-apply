package apply

import (
	"fmt"

	"github.com/abiosoft/incus-apply/internal/config"
	"github.com/abiosoft/incus-apply/internal/resource"
)

// formatResourceID creates a scope-aware display identifier for a resource.
// The format is: [remote:]type[/scope]/name
func formatResourceID(res *config.Resource) string {
	resourcePath := res.Type
	if usesNetworkScope(resource.Type(res.Type)) && res.Network != "" {
		resourcePath += "/" + res.Network
	}
	if usesPoolScope(resource.Type(res.Type)) && res.Pool != "" {
		resourcePath += "/" + res.Pool
	}
	if usesBucketScope(resource.Type(res.Type)) && res.Bucket != "" {
		resourcePath += "/" + res.Bucket
	}
	resourcePath += "/" + resourceIdentifier(res)

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

func usesPoolScope(resourceType resource.Type) bool {
	switch resourceType {
	case resource.TypeStorageVolume, resource.TypeStorageBucket, resource.TypeStorageBucketKey:
		return true
	default:
		return false
	}
}

func usesBucketScope(resourceType resource.Type) bool {
	return resourceType == resource.TypeStorageBucketKey
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
