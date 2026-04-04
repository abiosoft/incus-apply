package resource

import (
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

	sorted := SortForApply(resources)

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
