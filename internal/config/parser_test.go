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
type: vars
vars:
  NAME: world
---
type: instance
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
		_, _ = w.Write([]byte("type: instance\nname: web\nimage: images:alpine/3.19\n"))
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
		_, _ = w.Write([]byte("type: instance\nname: web\nimage: images:alpine/3.19\n"))
	}))
	defer server.Close()

	_, err := NewParser(10 * time.Millisecond).ParseURL(server.URL)
	if err == nil {
		t.Fatal("ParseURL() error = nil, want timeout error")
	}
}

func TestParseNetworkForwardFields(t *testing.T) {
	input := `type: network-forward
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

func TestParseInstanceSetupFields(t *testing.T) {
	input := "type: instance\n" +
		"name: web\n" +
		"image: images:alpine/3.19\n" +
		"setup:\n" +
		"  - action: exec\n" +
		"    when: create\n" +
		"    required: false\n" +
		"    cwd: /root\n" +
		"    script: echo hi\n" +
		"  - action: file_push\n" +
		"    when: update\n" +
		"    path: /etc/app.conf\n" +
		"    source: ./app.conf\n" +
		"    recursive: true\n" +
		"    uid: 0\n" +
		"    gid: 0\n" +
		"    mode: \"0644\"\n"

	result, err := NewParser(0).ParseStdin(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseStdin() error = %v", err)
	}
	res := result.Resources[0]
	if len(res.Setup) != 2 {
		t.Fatalf("setup = %#v, want 2 actions", res.Setup)
	}
	if res.Setup[0].Action != SetupActionExec || res.Setup[0].When != SetupWhenCreate {
		t.Fatalf("first setup action = %#v, want exec/create", res.Setup[0])
	}
	if res.Setup[0].CWD != "/root" {
		t.Fatalf("cwd = %q, want /root", res.Setup[0].CWD)
	}
	if res.Setup[0].IsRequired() {
		t.Fatal("required = true, want false")
	}
	if res.Setup[1].Mode != "0644" {
		t.Fatalf("mode = %q, want 0644", res.Setup[1].Mode)
	}
	if !res.Setup[1].Recursive {
		t.Fatal("recursive = false, want true")
	}
	if !res.Setup[1].IsRequired() {
		t.Fatal("required = false, want default true")
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
type: instance
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
type: profile
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
