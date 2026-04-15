package incus

import (
	"strings"
	"testing"

	"github.com/abiosoft/incus-apply/internal/config"
)

func TestDiffResource_ManagedUsesStoredSnapshot(t *testing.T) {
	current := `
config:
  user.key: value
  user.custom: keep
  user.incus-apply.created: "true"
  user.incus-apply.current: |
    config:
      user.key: value
`
	desired := &config.Resource{
		Base: config.Base{
			Type: "instance",
			Name: "test",
			Config: map[string]string{
				"user.key": "value",
			},
		},
	}

	changes, status, err := DiffResource(current, desired)
	if err != nil {
		t.Fatalf("DiffResource() error = %v", err)
	}
	if !status.Managed {
		t.Fatalf("expected managed status, got %#v", status)
	}
	if len(changes) != 0 {
		t.Fatalf("expected no changes, got %#v", changes)
	}
}

func TestDiffResource_UnmanagedFallsBackToLiveState(t *testing.T) {
	current := `
config:
  user.key: value
`
	desired := &config.Resource{
		Base: config.Base{
			Type: "instance",
			Name: "test",
			Config: map[string]string{
				"user.key": "updated",
			},
		},
	}

	changes, status, err := DiffResource(current, desired)
	if err != nil {
		t.Fatalf("DiffResource() error = %v", err)
	}
	if status.Managed {
		t.Fatalf("expected unmanaged status, got %#v", status)
	}
	if status.Warning != ManagementWarningUnmanaged {
		t.Fatalf("warning = %q, want %q", status.Warning, ManagementWarningUnmanaged)
	}
	if len(changes) == 0 {
		t.Fatal("expected fallback diff to contain changes")
	}
}

func TestDiffResource_UnmanagedTrackingKeysAreExcludedFromDiff(t *testing.T) {
	current := `
config:
  user.key: value
`
	desired := &config.Resource{
		Base: config.Base{
			Type: "instance",
			Name: "test",
			Config: map[string]string{
				"user.key":                 "value",
				"user.incus-apply.created": "true",
				"user.incus-apply.current": "config:\n  user.key: value\n",
			},
		},
	}

	changes, status, err := DiffResource(current, desired)
	if err != nil {
		t.Fatalf("DiffResource() error = %v", err)
	}
	if status.Managed {
		t.Fatalf("expected unmanaged status, got %#v", status)
	}
	if len(changes) != 0 {
		t.Fatalf("expected tracking keys to be excluded, got %#v", changes)
	}
}

func TestDiffResource_ManagedCreateOnlyChangeRequiresRecreate(t *testing.T) {
	current := `
config:
  user.incus-apply.created: "true"
  user.incus-apply.current: |
    image: images:alpine/3.19
    config:
      user.key: value
`
	desired := &config.Resource{
		Base: config.Base{
			Type: "instance",
			Name: "test",
			Config: map[string]string{
				"user.key": "value",
			},
		},
		InstanceFields: config.InstanceFields{Image: "images:alpine/3.20"},
	}

	changes, status, err := DiffResource(current, desired)
	if err != nil {
		t.Fatalf("DiffResource() error = %v", err)
	}
	if !status.Managed {
		t.Fatalf("expected managed status, got %#v", status)
	}
	if status.Warning != ManagementWarningRecreate {
		t.Fatalf("warning = %q, want %q", status.Warning, ManagementWarningRecreate)
	}
	if len(status.UnsupportedChanges) != 1 {
		t.Fatalf("unsupported changes = %#v, want one change", status.UnsupportedChanges)
	}
	if status.UnsupportedChanges[0].Path != "image" {
		t.Fatalf("unsupported change path = %q, want %q", status.UnsupportedChanges[0].Path, "image")
	}
	if len(changes) == 0 {
		t.Fatal("expected diff to contain changes")
	}
}

func TestDiffResource_NetworkForwardNetworkChangeRequiresRecreate(t *testing.T) {
	current := "config:\n" +
		"  user.incus-apply.created: \"true\"\n" +
		"  user.incus-apply.current: |\n" +
		"    listen_address: 198.51.100.10\n" +
		"    network: uplink-a\n" +
		"    config:\n" +
		"      target_address: 10.0.0.2\n"
	desired := &config.Resource{
		Base: config.Base{
			Type: "network-forward",
			Config: map[string]string{
				"target_address": "10.0.0.2",
			},
		},
		InstanceFields:       config.InstanceFields{Network: "uplink-b"},
		NetworkForwardFields: config.NetworkForwardFields{ListenAddress: "198.51.100.10"},
	}

	changes, status, err := DiffResource(current, desired)
	if err != nil {
		t.Fatalf("DiffResource() error = %v", err)
	}
	if !status.Managed {
		t.Fatalf("expected managed status, got %#v", status)
	}
	if status.Warning != ManagementWarningRecreate {
		t.Fatalf("warning = %q, want %q", status.Warning, ManagementWarningRecreate)
	}
	if len(status.UnsupportedChanges) != 1 {
		t.Fatalf("unsupported changes = %#v, want one change", status.UnsupportedChanges)
	}
	if status.UnsupportedChanges[0].Path != "network" {
		t.Fatalf("unsupported change path = %q, want network", status.UnsupportedChanges[0].Path)
	}
	if len(changes) == 0 {
		t.Fatal("expected diff to contain changes")
	}
}

func TestDiffResource_CloudInitConfigChangeRequiresRecreate(t *testing.T) {
	current := `
config:
  user.incus-apply.created: "true"
  user.incus-apply.current: |
    config:
      cloud-init.user-data: "#cloud-config\npackages: [curl]\n"
`
	desired := &config.Resource{
		Base: config.Base{
			Type: "instance",
			Name: "test",
			Config: map[string]string{
				"cloud-init.user-data": "#cloud-config\npackages: [curl, git]\n",
			},
		},
	}

	changes, status, err := DiffResource(current, desired)
	if err != nil {
		t.Fatalf("DiffResource() error = %v", err)
	}
	if !status.Managed {
		t.Fatalf("expected managed status, got %#v", status)
	}
	if status.Warning != ManagementWarningRecreate {
		t.Fatalf("warning = %q, want %q", status.Warning, ManagementWarningRecreate)
	}
	if len(status.UnsupportedChanges) != 1 {
		t.Fatalf("unsupported changes = %#v, want one change", status.UnsupportedChanges)
	}
	if !strings.HasPrefix(status.UnsupportedChanges[0].Path, "config.cloud-init.") {
		t.Fatalf("unsupported change path = %q, want config.cloud-init.* prefix", status.UnsupportedChanges[0].Path)
	}
	if len(changes) == 0 {
		t.Fatal("expected diff to contain changes")
	}
}

func TestDiffResource_CloudInitConfigChangeNotUnsupportedForNonInstance(t *testing.T) {
	current := `
config:
  user.incus-apply.created: "true"
  user.incus-apply.current: |
    config:
      cloud-init.user-data: "#cloud-config\npackages: [curl]\n"
`
	desired := &config.Resource{
		Base: config.Base{
			Type: "storage-pool",
			Name: "test",
			Config: map[string]string{
				"cloud-init.user-data": "#cloud-config\npackages: [curl, git]\n",
			},
		},
	}

	_, status, err := DiffResource(current, desired)
	if err != nil {
		t.Fatalf("DiffResource() error = %v", err)
	}
	if status.Warning == ManagementWarningRecreate {
		t.Fatal("expected no recreate warning for non-instance resource")
	}
}

func TestDiff(t *testing.T) {
	tests := []struct {
		name           string
		current        string
		desired        string
		wantEmpty      bool
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:      "identical configs",
			current:   "name: test\nvalue: 123\n",
			desired:   "name: test\nvalue: 123\n",
			wantEmpty: true,
		},
		{
			name:         "simple value change",
			current:      "size: 10GiB\n",
			desired:      "size: 12GiB\n",
			wantContains: []string{"size: \"10GiB\"", "→", "\"12GiB\""},
		},
		{
			name:         "nested value change",
			current:      "devices:\n  root:\n    size: 10GiB\n",
			desired:      "devices:\n  root:\n    size: 12GiB\n",
			wantContains: []string{"devices.root.size: \"10GiB\"", "→", "\"12GiB\""},
		},
		{
			name:         "deep nested change",
			current:      "a:\n  b:\n    c:\n      d: old\n",
			desired:      "a:\n  b:\n    c:\n      d: new\n",
			wantContains: []string{"a.b.c.d: \"old\"", "→", "\"new\""},
		},
		{
			name:           "only changed field shown",
			current:        "name: test\nsize: 10GiB\ndescription: hello\n",
			desired:        "name: test\nsize: 12GiB\ndescription: hello\n",
			wantContains:   []string{"size:"},
			wantNotContain: []string{"name:", "description:"},
		},
		{
			name:         "added field",
			current:      "name: test\n",
			desired:      "name: test\nnewfield: value\n",
			wantContains: []string{"newfield: \"value\""},
		},
		{
			name:         "removed field",
			current:      "name: test\noldfield: value\n",
			desired:      "name: test\n",
			wantContains: []string{"oldfield: \"value\""},
		},
		{
			name:         "boolean change",
			current:      "enabled: false\n",
			desired:      "enabled: true\n",
			wantContains: []string{"enabled: false", "→", "true"},
		},
		{
			name:         "numeric change",
			current:      "count: 5\n",
			desired:      "count: 10\n",
			wantContains: []string{"count: 5", "→", "10"},
		},
		{
			name:           "large multiline string summarized",
			current:        "config:\n  cloud-init.user-data: |\n    #cloud-config\n    package_update: true\n    packages:\n      - nginx\n    runcmd:\n      - echo old\n",
			desired:        "config:\n  cloud-init.user-data: |\n    #cloud-config\n    package_update: true\n    packages:\n      - nginx\n    runcmd:\n      - echo new\n",
			wantContains:   []string{"config.cloud-init.user-data:", "<76 chars>", "→"},
			wantNotContain: []string{"#cloud-config", "echo old", "echo new"},
		},
		{
			name:           "long single line string summarized",
			current:        "config:\n  user.script: abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz\n",
			desired:        "config:\n  user.script: abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz123\n",
			wantContains:   []string{"config.user.script:", "<78 chars>", "<81 chars>", "→"},
			wantNotContain: []string{"abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz123"},
		},
		{
			name:         "multiple changes",
			current:      "cpu: 2\nmemory: 1GiB\n",
			desired:      "cpu: 4\nmemory: 2GiB\n",
			wantContains: []string{"cpu:", "memory:"},
		},
		{
			name:         "nested addition",
			current:      "config:\n  cpu: 2\n",
			desired:      "config:\n  cpu: 2\n  memory: 1GiB\n",
			wantContains: []string{"config.memory: \"1GiB\""},
		},
		{
			name:         "nested removal",
			current:      "config:\n  cpu: 2\n  memory: 1GiB\n",
			desired:      "config:\n  cpu: 2\n",
			wantContains: []string{"config.memory: \"1GiB\""},
		},
		{
			name:         "array change",
			current:      "profiles:\n  - default\n",
			desired:      "profiles:\n  - default\n  - custom\n",
			wantContains: []string{"profiles[1]: \"custom\""},
		},
		{
			name: "array of maps change",
			current: "ports:\n" +
				"  - protocol: tcp\n" +
				"    listen_port: \"443\"\n" +
				"    target_address: 10.0.0.2\n",
			desired: "ports:\n" +
				"  - protocol: tcp\n" +
				"    listen_port: \"443\"\n" +
				"    target_address: 10.0.0.3\n",
			wantContains: []string{"ports[0].target_address: \"10.0.0.2\"", "→", "\"10.0.0.3\""},
		},
		{
			name:           "unchanged nested structure",
			current:        "config:\n  a: 1\n  b: 2\ndevices:\n  root:\n    size: 10GiB\n",
			desired:        "config:\n  a: 1\n  b: 2\ndevices:\n  root:\n    size: 12GiB\n",
			wantContains:   []string{"devices.root.size:"},
			wantNotContain: []string{"config.a:", "config.b:"},
		},
		{
			name:      "whitespace normalized",
			current:   "key: value\n",
			desired:   "key:   value\n",
			wantEmpty: true, // YAML normalizes whitespace
		},
		// Immutable fields tests
		{
			name:           "immutable field name excluded",
			current:        "name: old-name\ndescription: test\n",
			desired:        "name: new-name\ndescription: updated\n",
			wantContains:   []string{"description:"},
			wantNotContain: []string{"name:"},
		},
		{
			name:      "immutable field type excluded",
			current:   "type: container\n",
			desired:   "type: vm\n",
			wantEmpty: true,
		},
		{
			name:      "immutable field architecture excluded",
			current:   "architecture: aarch64\n",
			desired:   "architecture: x86_64\n",
			wantEmpty: true,
		},
		{
			name:      "immutable field created_at excluded",
			current:   "created_at: 2024-01-01T00:00:00Z\n",
			desired:   "created_at: 2025-01-01T00:00:00Z\n",
			wantEmpty: true,
		},
		{
			name:      "immutable field stateful excluded",
			current:   "stateful: false\n",
			desired:   "stateful: true\n",
			wantEmpty: true,
		},
		{
			name:      "immutable field status excluded",
			current:   "status: Running\n",
			desired:   "status: Stopped\n",
			wantEmpty: true,
		},
		{
			name:      "immutable field project excluded",
			current:   "project: default\n",
			desired:   "project: other\n",
			wantEmpty: true,
		},
		{
			name:      "immutable field location excluded",
			current:   "location: node1\n",
			desired:   "location: node2\n",
			wantEmpty: true,
		},
		// volatile.* config keys excluded
		{
			name:      "volatile config keys excluded",
			current:   "config:\n  volatile.uuid: abc123\n  volatile.last_state.power: RUNNING\n",
			desired:   "config:\n  volatile.uuid: def456\n  volatile.last_state.power: STOPPED\n",
			wantEmpty: true,
		},
		{
			name:           "volatile excluded but other config shown",
			current:        "config:\n  limits.cpu: 2\n  volatile.uuid: abc123\n",
			desired:        "config:\n  limits.cpu: 4\n  volatile.uuid: def456\n",
			wantContains:   []string{"config.limits.cpu:"},
			wantNotContain: []string{"volatile"},
		},
		// image.* config keys excluded
		{
			name:      "image config keys excluded",
			current:   "config:\n  image.os: Debian\n  image.release: trixie\n",
			desired:   "config:\n  image.os: Ubuntu\n  image.release: noble\n",
			wantEmpty: true,
		},
		{
			name:           "image excluded but other config shown",
			current:        "config:\n  limits.memory: 1GiB\n  image.os: Debian\n",
			desired:        "config:\n  limits.memory: 2GiB\n  image.os: Ubuntu\n",
			wantContains:   []string{"config.limits.memory:"},
			wantNotContain: []string{"image.os"},
		},
		// Combined test
		{
			name:           "all immutable and managed fields excluded",
			current:        "name: test\narchitecture: arm64\ntype: container\nconfig:\n  limits.cpu: 2\n  volatile.uuid: abc\n  image.os: Debian\n",
			desired:        "name: other\narchitecture: x86_64\ntype: vm\nconfig:\n  limits.cpu: 4\n  volatile.uuid: def\n  image.os: Ubuntu\n",
			wantContains:   []string{"config.limits.cpu:"},
			wantNotContain: []string{"name:", "architecture:", "type:", "volatile", "image.os"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := Diff(tt.current, tt.desired)
			if err != nil {
				t.Fatalf("Diff() error = %v", err)
			}

			// Strip ANSI color codes for easier testing
			diff = stripColors(diff)

			if tt.wantEmpty {
				if diff != "" {
					t.Errorf("Diff() expected empty, got:\n%s", diff)
				}
				return
			}

			if diff == "" {
				t.Errorf("Diff() expected non-empty diff")
				return
			}

			for _, substr := range tt.wantContains {
				if !strings.Contains(diff, substr) {
					t.Errorf("Diff() output missing expected substring %q:\n%s", substr, diff)
				}
			}

			for _, substr := range tt.wantNotContain {
				if strings.Contains(diff, substr) {
					t.Errorf("Diff() output should not contain %q:\n%s", substr, diff)
				}
			}
		})
	}
}

func TestDiff_InvalidYAML(t *testing.T) {
	tests := []struct {
		name    string
		current string
		desired string
	}{
		{
			name:    "invalid current - bad indentation",
			current: "key:\n  nested: value\n bad: indent\n",
			desired: "valid: yaml\n",
		},
		{
			name:    "invalid desired - unclosed bracket",
			current: "valid: yaml\n",
			desired: "key: [unclosed\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Diff(tt.current, tt.desired)
			if err == nil {
				t.Errorf("Diff() expected error for invalid YAML")
			}
		})
	}
}

func TestHasChanges(t *testing.T) {
	tests := []struct {
		name    string
		current string
		desired string
		want    bool
	}{
		{
			name:    "no changes",
			current: "key: value\n",
			desired: "key: value\n",
			want:    false,
		},
		{
			name:    "has changes",
			current: "key: old\n",
			desired: "key: new\n",
			want:    true,
		},
		{
			name:    "nested no changes",
			current: "config:\n  cpu: 2\n",
			desired: "config:\n  cpu: 2\n",
			want:    false,
		},
		{
			name:    "nested has changes",
			current: "config:\n  cpu: 2\n",
			desired: "config:\n  cpu: 4\n",
			want:    true,
		},
		{
			name:    "immutable field name - no change reported",
			current: "name: old\n",
			desired: "name: new\n",
			want:    false,
		},
		{
			name:    "volatile config - no change reported",
			current: "config:\n  volatile.uuid: abc\n",
			desired: "config:\n  volatile.uuid: def\n",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := HasChanges(tt.current, tt.desired)
			if err != nil {
				t.Fatalf("HasChanges() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("HasChanges() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindChanges(t *testing.T) {
	tests := []struct {
		name     string
		current  map[string]any
		desired  map[string]any
		wantLen  int
		wantPath string
	}{
		{
			name:    "empty maps",
			current: map[string]any{},
			desired: map[string]any{},
			wantLen: 0,
		},
		{
			name:     "single addition",
			current:  map[string]any{},
			desired:  map[string]any{"key": "value"},
			wantLen:  1,
			wantPath: "key",
		},
		{
			name:     "single removal",
			current:  map[string]any{"key": "value"},
			desired:  map[string]any{},
			wantLen:  1,
			wantPath: "key",
		},
		{
			name:     "single modification",
			current:  map[string]any{"key": "old"},
			desired:  map[string]any{"key": "new"},
			wantLen:  1,
			wantPath: "key",
		},
		{
			name:    "unchanged",
			current: map[string]any{"key": "value"},
			desired: map[string]any{"key": "value"},
			wantLen: 0,
		},
		{
			name:     "nested change",
			current:  map[string]any{"parent": map[string]any{"child": "old"}},
			desired:  map[string]any{"parent": map[string]any{"child": "new"}},
			wantLen:  1,
			wantPath: "parent.child",
		},
		// Immutable fields
		{
			name:    "immutable name field excluded",
			current: map[string]any{"name": "old"},
			desired: map[string]any{"name": "new"},
			wantLen: 0,
		},
		{
			name:    "immutable type field excluded",
			current: map[string]any{"type": "container"},
			desired: map[string]any{"type": "vm"},
			wantLen: 0,
		},
		{
			name:    "immutable architecture field excluded",
			current: map[string]any{"architecture": "arm64"},
			desired: map[string]any{"architecture": "x86_64"},
			wantLen: 0,
		},
		{
			name:     "mutable description field included",
			current:  map[string]any{"description": "old"},
			desired:  map[string]any{"description": "new"},
			wantLen:  1,
			wantPath: "description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := findChanges(tt.current, tt.desired, "")
			if len(changes) != tt.wantLen {
				t.Errorf("findChanges() returned %d changes, want %d", len(changes), tt.wantLen)
			}
			if tt.wantPath != "" && len(changes) > 0 {
				if changes[0].path != tt.wantPath {
					t.Errorf("findChanges() path = %q, want %q", changes[0].path, tt.wantPath)
				}
			}
		})
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{
			name:  "string",
			value: "hello",
			want:  `"hello"`,
		},
		{
			name:  "int",
			value: 42,
			want:  "42",
		},
		{
			name:  "bool true",
			value: true,
			want:  "true",
		},
		{
			name:  "bool false",
			value: false,
			want:  "false",
		},
		{
			name:  "map",
			value: map[string]any{"key": "value"},
			want:  "{...}",
		},
		{
			name:  "array of strings",
			value: []any{"a", "b", "c"},
			want:  `["a", "b", "c"]`,
		},
		{
			name:  "empty array",
			value: []any{},
			want:  "[]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatValue(tt.value)
			if got != tt.want {
				t.Errorf("formatValue(%v) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestJoinPath(t *testing.T) {
	tests := []struct {
		prefix string
		key    string
		want   string
	}{
		{"", "key", "key"},
		{"parent", "child", "parent.child"},
		{"a.b", "c", "a.b.c"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := joinPath(tt.prefix, tt.key)
			if got != tt.want {
				t.Errorf("joinPath(%q, %q) = %q, want %q", tt.prefix, tt.key, got, tt.want)
			}
		})
	}
}

func TestJoinIndexPath(t *testing.T) {
	tests := []struct {
		prefix string
		index  int
		want   string
	}{
		{"profiles", 1, "profiles[1]"},
		{"ports[0].targets", 2, "ports[0].targets[2]"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := joinIndexPath(tt.prefix, tt.index)
			if got != tt.want {
				t.Errorf("joinIndexPath(%q, %d) = %q, want %q", tt.prefix, tt.index, got, tt.want)
			}
		})
	}
}

// stripColors removes ANSI color codes from a string for testing.
func stripColors(s string) string {
	s = strings.ReplaceAll(s, colorRed, "")
	s = strings.ReplaceAll(s, colorGreen, "")
	s = strings.ReplaceAll(s, colorReset, "")
	return s
}
