package incus

import (
	"strings"
	"testing"

	"github.com/abiosoft/incus-apply/internal/config"
	"github.com/abiosoft/incus-apply/internal/resource"
)

func zoneRecordResource() *config.Resource {
	return &config.Resource{
		Base: config.Base{Type: "network-zone-record", Name: "ns"},
		NetworkZoneRecordFields: config.NetworkZoneRecordFields{
			Zone: "zelda.incus.at",
			Entries: []map[string]any{
				{"type": "A", "value": "10.20.30.2"},
				{"type": "AAAA", "value": "fd42:53::2"},
			},
		},
	}
}

func TestNetworkZoneRecordCreateCommand(t *testing.T) {
	meta, ok := resource.GetTypeMeta("network-zone-record")
	if !ok {
		t.Fatal("network-zone-record not registered")
	}

	args, stdin := client{}.buildCreateCommand(meta, zoneRecordResource())
	cmd := strings.Join(args, " ")
	want := "network zone record create zelda.incus.at ns"
	if cmd != want {
		t.Fatalf("command = %q, want %q", cmd, want)
	}
	if !strings.Contains(string(stdin), "entries:") {
		t.Fatalf("stdin = %q, want entries field", string(stdin))
	}
	if !strings.Contains(string(stdin), "fd42:53::2") {
		t.Fatalf("stdin = %q, want AAAA entry", string(stdin))
	}
}

func TestNetworkZoneRecordShowAndDeleteCommands(t *testing.T) {
	meta, _ := resource.GetTypeMeta("network-zone-record")
	res := zoneRecordResource()

	show := strings.Join(client{}.buildCommand(meta, meta.Show, res, false), " ")
	if want := "network zone record show zelda.incus.at ns"; show != want {
		t.Fatalf("show command = %q, want %q", show, want)
	}

	del := strings.Join(client{}.buildCommand(meta, meta.Delete, res, true), " ")
	if want := "network zone record delete zelda.incus.at ns --force"; del != want {
		t.Fatalf("delete command = %q, want %q", del, want)
	}
}
