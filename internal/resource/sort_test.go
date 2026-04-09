package resource

import (
	"strings"
	"testing"

	"github.com/abiosoft/incus-apply/internal/config"
)

func TestSortForApply(t *testing.T) {
	resources := []*config.Resource{
		{Base: config.Base{Type: "instance", Name: "app1"}},
		{Base: config.Base{Type: "storage-pool", Name: "pool1"}},
		{Base: config.Base{Type: "profile", Name: "profile1"}},
		{Base: config.Base{Type: "network", Name: "net1"}},
		{Base: config.Base{Type: "project", Name: "proj1"}},
	}

	sorted, err := SortForApply(resources)
	if err != nil {
		t.Fatalf("SortForApply() error = %v", err)
	}

	expectedOrder := []string{"project", "storage-pool", "network", "profile", "instance"}
	for i, expected := range expectedOrder {
		if sorted[i].Type != expected {
			t.Errorf("position %d: expected type %q, got %q", i, expected, sorted[i].Type)
		}
	}
}

func TestSortForDelete(t *testing.T) {
	resources := []*config.Resource{
		{Base: config.Base{Type: "project", Name: "proj1"}},
		{Base: config.Base{Type: "storage-pool", Name: "pool1"}},
		{Base: config.Base{Type: "instance", Name: "app1"}},
		{Base: config.Base{Type: "profile", Name: "profile1"}},
		{Base: config.Base{Type: "network", Name: "net1"}},
	}

	sorted := SortForDelete(resources)

	// Instances first (highest priority number), then down to project
	expectedOrder := []string{"instance", "profile", "network", "storage-pool", "project"}
	for i, expected := range expectedOrder {
		if sorted[i].Type != expected {
			t.Errorf("position %d: expected type %q, got %q", i, expected, sorted[i].Type)
		}
	}
}

func TestIsValidType(t *testing.T) {
	validTypes := []string{
		"instance", "profile", "network", "network-forward", "network-acl", "network-zone",
		"storage-pool", "storage-volume", "storage-bucket", "project", "cluster-group",
	}
	for _, typ := range validTypes {
		if !IsValidType(typ) {
			t.Errorf("expected %q to be valid", typ)
		}
	}

	invalidTypes := []string{"container", "vm", "volume", "unknown"}
	for _, typ := range invalidTypes {
		if IsValidType(typ) {
			t.Errorf("expected %q to be invalid", typ)
		}
	}
}

func TestGetTypeMeta(t *testing.T) {
	meta, ok := GetTypeMeta("instance")
	if !ok {
		t.Fatal("expected to find instance type")
	}
	if meta.Priority != 11 {
		t.Errorf("expected priority 11, got %d", meta.Priority)
	}

	_, ok = GetTypeMeta("unknown")
	if ok {
		t.Error("expected unknown type to not be found")
	}
}

func TestRegistryImmutability(t *testing.T) {
	// Verify built-in types cannot be overridden
	err := RegisterType(TypeMeta{
		Type:     TypeInstance,
		Priority: 999,
	})
	if err == nil {
		t.Error("expected error when trying to override built-in type")
	}
}

func TestSortForApplyAfterDependencies(t *testing.T) {
	resources := []*config.Resource{
		{Base: config.Base{Type: "instance", Name: "app"}, InstanceFields: config.InstanceFields{After: []string{"database"}}},
		{Base: config.Base{Type: "instance", Name: "database"}},
		{Base: config.Base{Type: "instance", Name: "cache"}},
	}

	sorted, err := SortForApply(resources)
	if err != nil {
		t.Fatalf("SortForApply() error = %v", err)
	}
	names := instanceNames(sorted)

	// database must come before app; cache has no constraints
	dbIdx := indexOf(names, "database")
	appIdx := indexOf(names, "app")
	if dbIdx >= appIdx {
		t.Errorf("expected database before app, got order: %v", names)
	}
}

func TestSortForApplyAfterChain(t *testing.T) {
	// app → api → database
	resources := []*config.Resource{
		{Base: config.Base{Type: "instance", Name: "app"}, InstanceFields: config.InstanceFields{After: []string{"api"}}},
		{Base: config.Base{Type: "instance", Name: "api"}, InstanceFields: config.InstanceFields{After: []string{"database"}}},
		{Base: config.Base{Type: "instance", Name: "database"}},
	}

	sorted, err := SortForApply(resources)
	if err != nil {
		t.Fatalf("SortForApply() error = %v", err)
	}
	names := instanceNames(sorted)

	dbIdx := indexOf(names, "database")
	apiIdx := indexOf(names, "api")
	appIdx := indexOf(names, "app")
	if !(dbIdx < apiIdx && apiIdx < appIdx) {
		t.Errorf("expected database < api < app, got order: %v", names)
	}
}

func TestSortForApplyAfterMultipleDeps(t *testing.T) {
	resources := []*config.Resource{
		{Base: config.Base{Type: "instance", Name: "app"}, InstanceFields: config.InstanceFields{After: []string{"database", "cache"}}},
		{Base: config.Base{Type: "instance", Name: "database"}},
		{Base: config.Base{Type: "instance", Name: "cache"}},
	}

	sorted, err := SortForApply(resources)
	if err != nil {
		t.Fatalf("SortForApply() error = %v", err)
	}
	names := instanceNames(sorted)

	appIdx := indexOf(names, "app")
	dbIdx := indexOf(names, "database")
	cacheIdx := indexOf(names, "cache")
	if dbIdx >= appIdx || cacheIdx >= appIdx {
		t.Errorf("expected database and cache before app, got order: %v", names)
	}
}

func TestSortForApplyAfterCycleReturnsError(t *testing.T) {
	resources := []*config.Resource{
		{Base: config.Base{Type: "instance", Name: "a"}, InstanceFields: config.InstanceFields{After: []string{"b"}}},
		{Base: config.Base{Type: "instance", Name: "b"}, InstanceFields: config.InstanceFields{After: []string{"a"}}},
	}

	_, err := SortForApply(resources)
	if err == nil {
		t.Fatal("expected error for cyclic dependency")
	}
	if !strings.Contains(err.Error(), "cyclic dependency") {
		t.Errorf("expected cycle error, got: %v", err)
	}
}

func TestSortForApplyAfterScopedToProject(t *testing.T) {
	resources := []*config.Resource{
		{Base: config.Base{Type: "instance", Name: "app", Project: "web"}, InstanceFields: config.InstanceFields{After: []string{"db"}}},
		{Base: config.Base{Type: "instance", Name: "db", Project: "web"}},
		{Base: config.Base{Type: "instance", Name: "app", Project: "api"}, InstanceFields: config.InstanceFields{After: []string{"db"}}},
		{Base: config.Base{Type: "instance", Name: "db", Project: "api"}},
	}

	sorted, err := SortForApply(resources)
	if err != nil {
		t.Fatalf("SortForApply() error = %v", err)
	}

	// Within each project, db should come before app
	for _, proj := range []string{"web", "api"} {
		var names []string
		for _, r := range sorted {
			p := r.Project
			if p == "" {
				p = "default"
			}
			if p == proj {
				names = append(names, r.Name)
			}
		}
		dbIdx := indexOf(names, "db")
		appIdx := indexOf(names, "app")
		if dbIdx >= appIdx {
			t.Errorf("project %q: expected db before app, got order: %v", proj, names)
		}
	}
}

func TestSortForApplyAfterMixedWithOtherTypes(t *testing.T) {
	resources := []*config.Resource{
		{Base: config.Base{Type: "instance", Name: "app"}, InstanceFields: config.InstanceFields{After: []string{"database"}}},
		{Base: config.Base{Type: "network", Name: "net1"}},
		{Base: config.Base{Type: "instance", Name: "database"}},
		{Base: config.Base{Type: "profile", Name: "prof1"}},
	}

	sorted, err := SortForApply(resources)
	if err != nil {
		t.Fatalf("SortForApply() error = %v", err)
	}

	// Non-instance types should still be ordered by priority (before instances)
	var instanceNames []string
	for _, r := range sorted {
		if r.Type == "instance" {
			instanceNames = append(instanceNames, r.Name)
		}
	}
	dbIdx := indexOf(instanceNames, "database")
	appIdx := indexOf(instanceNames, "app")
	if dbIdx >= appIdx {
		t.Errorf("expected database before app among instances, got: %v", instanceNames)
	}
}

func instanceNames(resources []*config.Resource) []string {
	var names []string
	for _, r := range resources {
		if r.Type == "instance" {
			names = append(names, r.Name)
		}
	}
	return names
}

func indexOf(names []string, name string) int {
	for i, n := range names {
		if n == name {
			return i
		}
	}
	return -1
}
