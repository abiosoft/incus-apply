package incus

import (
	"strings"
	"testing"

	"github.com/abiosoft/incus-apply/internal/config"
)

func TestDiffResource_ZoneRecordEntriesChange(t *testing.T) {
	current := `description: ""
entries:
  - type: A
    value: 192.0.2.1
config: {}
name: ns
`
	desired := &config.Resource{
		Base: config.Base{
			Type: "network-zone-record",
			Name: "ns",
		},
		NetworkZoneRecordFields: config.NetworkZoneRecordFields{
			Zone: "example.net",
			Entries: []map[string]any{
				{"type": "A", "value": "192.0.2.2"},
			},
		},
	}

	changes, _, err := DiffResource(current, desired)
	if err != nil {
		t.Fatalf("DiffResource() error = %v", err)
	}
	if len(changes) == 0 {
		t.Fatal("expected an entries change, got none")
	}

	found := false
	for _, change := range changes {
		if strings.HasPrefix(change.Path, "entries") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("changes = %#v, want an entries change", changes)
	}
}
