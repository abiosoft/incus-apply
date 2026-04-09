package incus

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/abiosoft/incus-apply/internal/config"
	"gopkg.in/yaml.v3"
)

func TestDesiredForApply_AddsTrackingState(t *testing.T) {
	res := &config.Resource{
		Base: config.Base{
			Type: "instance",
			Name: "test",
			Config: map[string]string{
				"user.key": "value",
			},
		},
	}

	prepared, snapshot, err := desiredForApply(res)
	if err != nil {
		t.Fatalf("desiredForApply() error = %v", err)
	}
	if prepared.Config[createdByKey] != "true" {
		t.Fatalf("created marker = %q, want %q", prepared.Config[createdByKey], "true")
	}
	if prepared.Config[currentStateKey] != snapshot {
		t.Fatalf("current state marker mismatch")
	}
	if strings.Contains(snapshot, createdByKey) || strings.Contains(snapshot, currentStateKey) {
		t.Fatalf("snapshot should not include tracking keys, got %q", snapshot)
	}
	if strings.Contains(snapshot, "\n") {
		t.Fatalf("snapshot should be compact, got %q", snapshot)
	}
	if !strings.HasPrefix(snapshot, "{") {
		t.Fatalf("snapshot = %q, want compact json", snapshot)
	}
	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(snapshot), &parsed); err != nil {
		t.Fatalf("snapshot should remain parseable as yaml/json, got error %v", err)
	}
}

func TestDesiredForApply_IncludesSetupSourceReference(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "instance.yaml")
	res := &config.Resource{
		Base: config.Base{Type: "instance", Name: "test", SourceFile: configPath},
		InstanceFields: config.InstanceFields{
			Image: "images:alpine/3.19",
			Setup: []config.SetupAction{{
				Action:    config.SetupActionPushFile,
				When:      config.SetupWhenUpdate,
				Path:      "/etc/caddy/Caddyfile",
				Source:    "./Caddyfile",
				Recursive: true,
			}},
		},
	}

	_, snapshot, err := desiredForApply(res)
	if err != nil {
		t.Fatalf("desiredForApply() error = %v", err)
	}
	if !strings.Contains(snapshot, "\"source\":\"./Caddyfile\"") {
		t.Fatalf("snapshot = %q, want relative source preserved", snapshot)
	}
	if !strings.Contains(snapshot, "\"recursive\":true") {
		t.Fatalf("snapshot = %q, want recursive flag preserved", snapshot)
	}
}

func TestDesiredForApply_HashesInlineSetupContent(t *testing.T) {
	res := &config.Resource{
		Base: config.Base{Type: "instance", Name: "test"},
		InstanceFields: config.InstanceFields{
			Image: "images:alpine/3.19",
			Setup: []config.SetupAction{{
				Action:  config.SetupActionPushFile,
				When:    config.SetupWhenUpdate,
				Path:    "/etc/app.conf",
				Content: "hello world",
			}},
		},
	}

	_, snapshot, err := desiredForApply(res)
	if err != nil {
		t.Fatalf("desiredForApply() error = %v", err)
	}
	if strings.Contains(snapshot, "hello world") {
		t.Fatalf("snapshot = %q, want raw inline content omitted", snapshot)
	}
	if !strings.Contains(snapshot, "\"content\":\"hash: b94d27b9934d3e08a52e52d711\"") {
		t.Fatalf("snapshot = %q, want inline content hash recorded", snapshot)
	}
}

func TestDesiredForApply_HashesSetupScript(t *testing.T) {
	res := &config.Resource{
		Base: config.Base{Type: "instance", Name: "test"},
		InstanceFields: config.InstanceFields{
			Image: "images:alpine/3.19",
			Setup: []config.SetupAction{{
				Action: config.SetupActionExec,
				When:   config.SetupWhenAlways,
				Script: "echo hello world",
			}},
		},
	}

	_, snapshot, err := desiredForApply(res)
	if err != nil {
		t.Fatalf("desiredForApply() error = %v", err)
	}
	if strings.Contains(snapshot, "echo hello world") {
		t.Fatalf("snapshot = %q, want raw script omitted", snapshot)
	}
	if !strings.Contains(snapshot, "\"script\":\"hash: 5e9f1ed0cbd05609e29817be16\"") {
		t.Fatalf("snapshot = %q, want script hash recorded", snapshot)
	}
}

func TestDesiredForApply_PreservesOptionalSetupState(t *testing.T) {
	required := false
	res := &config.Resource{
		Base: config.Base{Type: "instance", Name: "test"},
		InstanceFields: config.InstanceFields{
			Image: "images:alpine/3.19",
			Setup: []config.SetupAction{{
				Action:   config.SetupActionExec,
				When:     config.SetupWhenAlways,
				Required: &required,
				Script:   "echo hello world",
			}},
		},
	}

	_, snapshot, err := desiredForApply(res)
	if err != nil {
		t.Fatalf("desiredForApply() error = %v", err)
	}
	if !strings.Contains(snapshot, "\"required\":false") {
		t.Fatalf("snapshot = %q, want optional setup state preserved", snapshot)
	}
}

func TestDiffResource_CreateSetupChangeRequiresRecreate(t *testing.T) {
	current := "config:\n  user.incus-apply.created: \"true\"\n  user.incus-apply.current: '{\"image\":\"images:alpine/3.19\",\"setup\":[{\"action\":\"exec\",\"when\":\"create\",\"script\":\"hash: 819b561be4b01d042acf9c1528\"}]}'\n"
	desired := &config.Resource{
		Base: config.Base{Type: "instance", Name: "web"},
		InstanceFields: config.InstanceFields{
			Image: "images:alpine/3.19",
			Setup: []config.SetupAction{{
				Action: config.SetupActionExec,
				When:   config.SetupWhenCreate,
				Script: "echo new",
			}},
		},
	}

	changes, status, err := DiffResource(current, desired)
	if err != nil {
		t.Fatalf("DiffResource() error = %v", err)
	}
	if len(changes) == 0 {
		t.Fatal("changes = 0, want setup diff")
	}
	if status.Warning != ManagementWarningRecreate {
		t.Fatalf("warning = %q, want %q", status.Warning, ManagementWarningRecreate)
	}
	if len(status.UnsupportedChanges) == 0 {
		t.Fatal("unsupported changes = 0, want recreate-required setup change")
	}
}

func TestMergeConfigs_OverlaysUserConfig(t *testing.T) {
	current := `
config:
  user.key1: value1
  volatile.base_image: abc123
devices: {}
`
	desired := &config.Resource{
		Base: config.Base{
			Type: "instance",
			Name: "test",
			Config: map[string]string{
				"user.key1": "updated",
				"user.key2": "new",
			},
		},
	}

	merged, err := mergeConfigs(current, desired)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := yaml.Unmarshal(merged, &result); err != nil {
		t.Fatalf("failed to unmarshal merged: %v", err)
	}

	cfg := result["config"].(map[string]any)
	if cfg["user.key1"] != "updated" {
		t.Errorf("expected user.key1=updated, got %v", cfg["user.key1"])
	}
	if cfg["user.key2"] != "new" {
		t.Errorf("expected user.key2=new, got %v", cfg["user.key2"])
	}
	if cfg["volatile.base_image"] != "abc123" {
		t.Errorf("expected volatile.base_image preserved, got %v", cfg["volatile.base_image"])
	}
}

func TestMergeConfigs_ReplacesDevices(t *testing.T) {
	current := `
devices:
  eth0:
    type: nic
    network: old-net
`
	desired := &config.Resource{
		Base: config.Base{
			Type: "instance",
			Name: "test",
			Devices: map[string]map[string]any{
				"eth0": {"type": "nic", "network": "new-net"},
			},
		},
	}

	merged, err := mergeConfigs(current, desired)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := yaml.Unmarshal(merged, &result); err != nil {
		t.Fatalf("failed to unmarshal merged: %v", err)
	}

	devices := result["devices"].(map[string]any)
	eth0 := devices["eth0"].(map[string]any)
	if eth0["network"] != "new-net" {
		t.Errorf("expected network=new-net, got %v", eth0["network"])
	}
}

func TestMergeConfigs_InstanceProfiles(t *testing.T) {
	current := `
profiles:
  - default
`
	desired := &config.Resource{
		Base: config.Base{
			Type: "instance",
			Name: "test",
		},
		InstanceFields: config.InstanceFields{Profiles: []string{"default", "custom"}},
	}

	merged, err := mergeConfigs(current, desired)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := yaml.Unmarshal(merged, &result); err != nil {
		t.Fatalf("failed to unmarshal merged: %v", err)
	}

	profiles := result["profiles"].([]any)
	if len(profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(profiles))
	}
	if profiles[1] != "custom" {
		t.Errorf("expected second profile=custom, got %v", profiles[1])
	}
}

func TestMergeConfigs_NetworkACLRules(t *testing.T) {
	current := `
ingress:
  - action: allow
    source: 10.0.0.0/8
`
	desired := &config.Resource{
		Base: config.Base{
			Type: "network-acl",
			Name: "test-acl",
		},
		NetworkACLFields: config.NetworkACLFields{
			Ingress: []map[string]any{
				{"action": "allow", "source": "192.168.0.0/16"},
			},
			Egress: []map[string]any{
				{"action": "drop"},
			},
		},
	}

	merged, err := mergeConfigs(current, desired)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := yaml.Unmarshal(merged, &result); err != nil {
		t.Fatalf("failed to unmarshal merged: %v", err)
	}

	ingress := result["ingress"].([]any)
	if len(ingress) != 1 {
		t.Fatalf("expected 1 ingress rule (replaced), got %d", len(ingress))
	}
	rule := ingress[0].(map[string]any)
	if rule["source"] != "192.168.0.0/16" {
		t.Errorf("expected source=192.168.0.0/16, got %v", rule["source"])
	}

	egress := result["egress"].([]any)
	if len(egress) != 1 {
		t.Fatalf("expected 1 egress rule, got %d", len(egress))
	}
}

func TestMergeConfigs_NetworkForwardPorts(t *testing.T) {
	current := `
description: old
ports:
  - protocol: tcp
    listen_port: "80"
    target_address: 10.0.0.2
`
	desired := &config.Resource{
		Base: config.Base{
			Type:        "network-forward",
			Description: "updated",
		},
		InstanceFields: config.InstanceFields{Network: "uplink"},
		NetworkForwardFields: config.NetworkForwardFields{
			ListenAddress: "198.51.100.10",
			Ports: []map[string]any{
				{"protocol": "tcp", "listen_port": "443", "target_address": "10.0.0.3"},
			},
		},
	}

	merged, err := mergeConfigs(current, desired)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := yaml.Unmarshal(merged, &result); err != nil {
		t.Fatalf("failed to unmarshal merged: %v", err)
	}

	if result["description"] != "updated" {
		t.Fatalf("description = %v, want updated", result["description"])
	}
	ports := result["ports"].([]any)
	if len(ports) != 1 {
		t.Fatalf("expected 1 port rule, got %d", len(ports))
	}
	port := ports[0].(map[string]any)
	if port["listen_port"] != "443" {
		t.Fatalf("listen_port = %v, want 443", port["listen_port"])
	}
}

func TestMergeConfigs_PreservesUnmanagedFields(t *testing.T) {
	current := `
architecture: x86_64
status: Running
config:
  volatile.uuid: abc
`
	desired := &config.Resource{
		Base: config.Base{
			Type:        "instance",
			Name:        "test",
			Description: "updated desc",
		},
	}

	merged, err := mergeConfigs(current, desired)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := yaml.Unmarshal(merged, &result); err != nil {
		t.Fatalf("failed to unmarshal merged: %v", err)
	}

	if result["architecture"] != "x86_64" {
		t.Errorf("expected architecture preserved, got %v", result["architecture"])
	}
	if result["description"] != "updated desc" {
		t.Errorf("expected description=updated desc, got %v", result["description"])
	}
}

func TestMergeConfigs_NilConfigPreservesCurrent(t *testing.T) {
	current := `
config:
  user.key: value
`
	desired := &config.Resource{
		Base: config.Base{
			Type: "instance",
			Name: "test",
		},
	}

	merged, err := mergeConfigs(current, desired)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := yaml.Unmarshal(merged, &result); err != nil {
		t.Fatalf("failed to unmarshal merged: %v", err)
	}

	cfg := result["config"].(map[string]any)
	if cfg["user.key"] != "value" {
		t.Errorf("expected config preserved when desired is nil, got %v", cfg["user.key"])
	}
}

func TestMergeConfigs_ManagedStateReplacesPreviousManagedKeysOnly(t *testing.T) {
	current := `
config:
  user.old: old
  user.custom: keep
  user.incus-apply.created: "true"
  user.incus-apply.current: |
    config:
      user.old: old
    devices:
      old0:
        type: nic
        network: old-net
devices:
  old0:
    type: nic
    network: old-net
  extra0:
    type: nic
    network: keep-net
`
	desired := &config.Resource{
		Base: config.Base{
			Type: "instance",
			Name: "test",
			Config: map[string]string{
				"user.new": "new",
			},
			Devices: map[string]map[string]any{
				"new0": {"type": "nic", "network": "new-net"},
			},
		},
	}

	merged, err := mergeConfigs(current, desired)
	if err != nil {
		t.Fatalf("mergeConfigs() error = %v", err)
	}

	var result map[string]any
	if err := yaml.Unmarshal(merged, &result); err != nil {
		t.Fatalf("failed to unmarshal merged: %v", err)
	}

	cfg := result["config"].(map[string]any)
	if _, ok := cfg["user.old"]; ok {
		t.Fatalf("expected previous managed key to be removed")
	}
	if cfg["user.custom"] != "keep" {
		t.Fatalf("expected custom config preserved, got %v", cfg["user.custom"])
	}
	if cfg["user.new"] != "new" {
		t.Fatalf("expected new managed config applied, got %v", cfg["user.new"])
	}
	if cfg[createdByKey] != "true" {
		t.Fatalf("expected created marker to remain set, got %v", cfg[createdByKey])
	}

	devices := result["devices"].(map[string]any)
	if _, ok := devices["old0"]; ok {
		t.Fatalf("expected previous managed device to be removed")
	}
	if _, ok := devices["extra0"]; !ok {
		t.Fatalf("expected unmanaged device to be preserved")
	}
	if _, ok := devices["new0"]; !ok {
		t.Fatalf("expected new managed device to be applied")
	}
}

func TestCleanMap(t *testing.T) {
	m := map[string]any{
		"keep":   "value",
		"remove": nil,
		"nested": map[string]any{
			"keep":   "inner",
			"remove": nil,
		},
		"empty_string": "",
		"empty_map":    map[string]any{},
	}

	cleanMap(m)

	if _, ok := m["remove"]; ok {
		t.Error("expected nil key to be removed")
	}
	if m["keep"] != "value" {
		t.Error("expected non-nil key to be preserved")
	}
	nested := m["nested"].(map[string]any)
	if _, ok := nested["remove"]; ok {
		t.Error("expected nested nil key to be removed")
	}
	if nested["keep"] != "inner" {
		t.Error("expected nested non-nil key to be preserved")
	}
	if m["empty_string"] != "" {
		t.Error("expected empty string to be preserved")
	}
	if _, ok := m["empty_map"]; !ok {
		t.Error("expected empty map to be preserved")
	}
}
