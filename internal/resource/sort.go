package resource

import (
	"fmt"
	"sort"
	"strings"

	"github.com/abiosoft/incus-apply/internal/config"
)

// defaultPriority is assigned to unknown resource types.
// Higher value means processed later during apply, earlier during delete.
const defaultPriority = 100

// SortForApply returns a copy of resources sorted by priority for creation.
// Resources with lower priority values are created first to satisfy dependencies
// (e.g., projects/networks before instances that use them).
// Within the same priority level, instances are ordered by their "after" dependencies.
func SortForApply(resources []*config.Resource) ([]*config.Resource, error) {
	sorted := sortResources(resources, func(a, b int) bool { return a < b })
	return orderByAfter(sorted)
}

// SortForDelete returns a copy of resources sorted by priority for deletion.
// Resources with higher priority values are deleted first (reverse of creation order)
// to respect dependency relationships.
func SortForDelete(resources []*config.Resource) []*config.Resource {
	return sortResources(resources, func(a, b int) bool { return a > b })
}

// sortResources creates a sorted copy using the provided comparison function.
// Uses stable sort to preserve relative order of same-priority resources.
func sortResources(resources []*config.Resource, less func(a, b int) bool) []*config.Resource {
	sorted := make([]*config.Resource, len(resources))
	copy(sorted, resources)

	sort.SliceStable(sorted, func(i, j int) bool {
		pi := getPriority(sorted[i].Type)
		pj := getPriority(sorted[j].Type)
		return less(pi, pj)
	})

	return sorted
}

// getPriority returns the priority for a resource type.
// Unknown types get defaultPriority, ensuring they are processed last during
// apply and first during delete (safest behavior for unknown resources).
func getPriority(t string) int {
	if meta, ok := GetTypeMeta(t); ok {
		return meta.Priority
	}
	return defaultPriority
}

// instanceProject returns the effective project for an instance resource.
func instanceProject(res *config.Resource) string {
	if p := strings.TrimSpace(res.Project); p != "" {
		return p
	}
	return "default"
}

// instanceEntry tracks an instance resource and its position in the full resource list.
type instanceEntry struct {
	index int
	res   *config.Resource
}

// orderByAfter reorders instance resources to respect "after" dependencies
// using topological sort (Kahn's algorithm). Instances are grouped by project
// since "after" references are scoped to the same project. Non-instance
// resources and instances without dependencies preserve their relative order.
// If a cycle is detected, an error is returned.
func orderByAfter(resources []*config.Resource) ([]*config.Resource, error) {
	// Separate instances by project, tracking their positions in the full list.
	groups := map[string][]instanceEntry{}
	hasAfter := false
	for i, res := range resources {
		if Type(res.Type) == TypeInstance {
			proj := instanceProject(res)
			groups[proj] = append(groups[proj], instanceEntry{i, res})
			if len(res.After) > 0 {
				hasAfter = true
			}
		}
	}
	if !hasAfter {
		return resources, nil
	}

	result := make([]*config.Resource, len(resources))
	copy(result, resources)

	for _, entries := range groups {
		if len(entries) < 2 {
			continue
		}
		sorted, err := topoSort(entries)
		if err != nil {
			return nil, err
		}
		// Write sorted instances back into their original positions.
		positions := make([]int, len(entries))
		for i, e := range entries {
			positions[i] = e.index
		}
		sort.Ints(positions)
		for i, pos := range positions {
			result[pos] = sorted[i]
		}
	}

	return result, nil
}

// topoSort performs a topological sort on instance entries using Kahn's algorithm.
func topoSort(entries []instanceEntry) ([]*config.Resource, error) {
	// Build name → entry index map.
	nameIdx := make(map[string]int, len(entries))
	for i, e := range entries {
		nameIdx[e.res.Name] = i
	}

	// Build in-degree counts and adjacency list.
	inDegree := make([]int, len(entries))
	dependents := make([][]int, len(entries)) // dependents[i] = entries that depend on i
	for i, e := range entries {
		for _, dep := range e.res.After {
			j, ok := nameIdx[dep]
			if !ok {
				continue // dependency not in this group; skip
			}
			inDegree[i]++
			dependents[j] = append(dependents[j], i)
		}
	}

	// Kahn's algorithm: process entries with no remaining dependencies.
	var queue []int
	for i, d := range inDegree {
		if d == 0 {
			queue = append(queue, i)
		}
	}

	sorted := make([]*config.Resource, 0, len(entries))
	for len(queue) > 0 {
		idx := queue[0]
		queue = queue[1:]
		sorted = append(sorted, entries[idx].res)
		for _, dep := range dependents[idx] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	if len(sorted) != len(entries) {
		// Identify the instances involved in the cycle.
		visited := make(map[string]bool, len(sorted))
		for _, r := range sorted {
			visited[r.Name] = true
		}
		var cycled []string
		for _, e := range entries {
			if !visited[e.res.Name] {
				cycled = append(cycled, e.res.Name)
			}
		}
		return nil, fmt.Errorf("cyclic dependency detected among instances: %s", strings.Join(cycled, ", "))
	}
	return sorted, nil
}
