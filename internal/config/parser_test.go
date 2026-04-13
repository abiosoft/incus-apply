package config

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseStdinSetsSourceFile(t *testing.T) {
	input := `---
kind: vars
vars:
  NAME: world
---
kind: instance
name: web
image: images:alpine/3.19
`

	result, err := NewParser(0).ParseStdin(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseStdin() error = %v", err)
	}
	if result.SourceFile != "stdin" {
		t.Fatalf("SourceFile = %q, want %q", result.SourceFile, "stdin")
	}
	if len(result.Vars) != 1 || result.Vars[0].SourceFile != "stdin" {
		t.Fatalf("vars source file = %#v, want stdin", result.Vars)
	}
	if len(result.Resources) != 1 || result.Resources[0].SourceFile != "stdin" {
		t.Fatalf("resources source file = %#v, want stdin", result.Resources)
	}
}

func TestParseURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("kind: instance\nname: web\nimage: images:alpine/3.19\n"))
	}))
	defer server.Close()

	result, err := NewParser(0).ParseURL(server.URL)
	if err != nil {
		t.Fatalf("ParseURL() error = %v", err)
	}
	if result.SourceFile != server.URL {
		t.Fatalf("SourceFile = %q, want %q", result.SourceFile, server.URL)
	}
	if len(result.Resources) != 1 {
		t.Fatalf("resources = %d, want 1", len(result.Resources))
	}
}

func TestParseURLTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte("kind: instance\nname: web\nimage: images:alpine/3.19\n"))
	}))
	defer server.Close()

	_, err := NewParser(10 * time.Millisecond).ParseURL(server.URL)
	if err == nil {
		t.Fatal("ParseURL() error = nil, want timeout error")
	}
}

func TestParseNetworkForwardFields(t *testing.T) {
	input := `kind: network-forward
listen_address: 198.51.100.10
network: uplink
config:
  target_address: 10.0.0.2
ports:
  - protocol: tcp
    listen_port: "443"
    target_address: 10.0.0.3
`

	result, err := NewParser(0).ParseStdin(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseStdin() error = %v", err)
	}
	if len(result.Resources) != 1 {
		t.Fatalf("resources = %d, want 1", len(result.Resources))
	}
	res := result.Resources[0]
	if res.Network != "uplink" {
		t.Fatalf("network = %q, want uplink", res.Network)
	}
	if res.ListenAddress != "198.51.100.10" {
		t.Fatalf("listen_address = %q, want 198.51.100.10", res.ListenAddress)
	}
	if len(res.Ports) != 1 {
		t.Fatalf("ports = %#v, want one rule", res.Ports)
	}
}

func TestParseInstanceApplyAfterField(t *testing.T) {
	input := "kind: instance\n" +
		"name: web\n" +
		"image: images:alpine/3.19\n" +
		"apply.after:\n" +
		"  - db\n"

	result, err := NewParser(0).ParseStdin(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseStdin() error = %v", err)
	}
	res := result.Resources[0]
	if len(res.After) != 1 || res.After[0] != "db" {
		t.Fatalf("after = %#v, want [db]", res.After)
	}
}

func TestParseSkipsDocumentsWithUnknownType(t *testing.T) {
	// A multi-document YAML where some documents are not incus resources.
	input := `---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
---
kind: instance
name: web
image: images:alpine/3.19
---
type: UnknownThing
name: foo
---
name: no-type-field
value: 42
`

	result, err := NewParser(0).ParseStdin(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseStdin() error = %v, want no error", err)
	}
	if len(result.Resources) != 1 {
		t.Fatalf("resources = %d, want 1 (only the instance)", len(result.Resources))
	}
	if result.Resources[0].Name != "web" {
		t.Fatalf("resource name = %q, want web", result.Resources[0].Name)
	}
}

func TestParseSkipsEmptyTypeDocuments(t *testing.T) {
	input := `---
name: something
value: 42
---
kind: profile
name: default
`

	result, err := NewParser(0).ParseStdin(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseStdin() error = %v", err)
	}
	if len(result.Resources) != 1 || result.Resources[0].Type != "profile" {
		t.Fatalf("resources = %v, want [profile/default]", result.Resources)
	}
}

func TestCloudInitYAMLMappingConvertedToString(t *testing.T) {
	// cloud-init.user-data written as inline YAML mapping — the parser must
	// convert it to a plain string so the rest of the pipeline sees a string.
	input := `kind: instance
name: web
image: images:debian/13/cloud
config:
  cloud-init.user-data:
    #cloud-config
    packages:
      - nginx
    runcmd:
      - systemctl enable nginx
`
	result, err := NewParser(0).ParseStdin(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseStdin() error = %v", err)
	}
	if len(result.Resources) != 1 {
		t.Fatalf("resources = %d, want 1", len(result.Resources))
	}
	got := result.Resources[0].Config["cloud-init.user-data"]
	if got == "" {
		t.Fatal("cloud-init.user-data = empty, want non-empty string")
	}
	// The YAML comment #cloud-config must appear as an actual text line.
	if !strings.Contains(got, "#cloud-config") {
		t.Errorf("cloud-init.user-data does not contain #cloud-config:\n%s", got)
	}
	if !strings.Contains(got, "nginx") {
		t.Errorf("cloud-init.user-data does not contain package 'nginx':\n%s", got)
	}
}

func TestCloudInitPlainStringUnchanged(t *testing.T) {
	// A cloud-init value written as a plain block scalar must pass through
	// unchanged; the normalizer must not alter it.
	input := `kind: instance
name: web
image: images:debian/13/cloud
config:
  cloud-init.user-data: |
    #cloud-config
    packages:
      - nginx
`
	result, err := NewParser(0).ParseStdin(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseStdin() error = %v", err)
	}
	got := result.Resources[0].Config["cloud-init.user-data"]
	if !strings.HasPrefix(got, "#cloud-config\n") {
		t.Errorf("cloud-init.user-data = %q, want prefix '#cloud-config\\n'", got)
	}
}

func TestNonCloudInitConfigKeyUnchanged(t *testing.T) {
	// Non-cloud-init keys in config must not be affected by normalisation.
	input := `kind: instance
name: web
image: images:debian/13
config:
  limits.cpu: "2"
  limits.memory: 1GiB
`
	result, err := NewParser(0).ParseStdin(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseStdin() error = %v", err)
	}
	res := result.Resources[0]
	if res.Config["limits.cpu"] != "2" {
		t.Errorf("limits.cpu = %q, want '2'", res.Config["limits.cpu"])
	}
	if res.Config["limits.memory"] != "1GiB" {
		t.Errorf("limits.memory = %q, want '1GiB'", res.Config["limits.memory"])
	}
}
