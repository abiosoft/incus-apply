package config

import (
	"strings"
	"testing"
)

func TestParseNetworkZoneRecord(t *testing.T) {
	input := `kind: network-zone-record
zone: zelda.incus.at
name: ns
entries:
  - type: A
    value: 10.20.30.2
  - type: AAAA
    value: fd42:53::2
`

	result, err := NewParser(0).ParseStdin(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseStdin() error = %v", err)
	}
	if len(result.Resources) != 1 {
		t.Fatalf("resources = %d, want 1", len(result.Resources))
	}

	res := result.Resources[0]
	if res.Zone != "zelda.incus.at" {
		t.Fatalf("zone = %q, want zelda.incus.at", res.Zone)
	}
	if res.Name != "ns" {
		t.Fatalf("name = %q, want ns", res.Name)
	}
	if len(res.Entries) != 2 {
		t.Fatalf("entries = %#v, want 2 entries", res.Entries)
	}
	if got := res.Entries[0]["type"]; got != "A" {
		t.Fatalf("entries[0].type = %v, want A", got)
	}
	if got := res.Entries[1]["value"]; got != "fd42:53::2" {
		t.Fatalf("entries[1].value = %v, want fd42:53::2", got)
	}
}

func TestNetworkZoneRecordRequiresZone(t *testing.T) {
	input := "kind: network-zone-record\nname: ns\n"

	_, err := NewParser(0).ParseStdin(strings.NewReader(input))
	if err == nil {
		t.Fatal("ParseStdin() error = nil, want zone validation error")
	}
	if !strings.Contains(err.Error(), "zone is required") {
		t.Fatalf("error = %v, want zone is required", err)
	}
}

func TestParseMultipleZoneRecordsInOneFile(t *testing.T) {
	input := `kind: network-zone-record
zone: zelda.incus.at
name: ns
entries:
  - type: A
    value: 10.20.30.2
---
kind: network-zone-record
zone: zelda.incus.at
name: grafana
entries:
  - type: A
    value: 10.60.241.175
`

	result, err := NewParser(0).ParseStdin(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseStdin() error = %v", err)
	}
	if len(result.Resources) != 2 {
		t.Fatalf("resources = %d, want 2", len(result.Resources))
	}
	if result.Resources[0].Name != "ns" || result.Resources[1].Name != "grafana" {
		t.Fatalf("names = %q, %q; want ns, grafana", result.Resources[0].Name, result.Resources[1].Name)
	}
}
