package renderer

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/abiosoft/incus-apply/internal/apply"
	"github.com/abiosoft/incus-apply/internal/incus"
)

func TestPlural(t *testing.T) {
	tests := []struct {
		word  string
		count int
		want  string
	}{
		{"resource", 0, "resources"},
		{"resource", 1, "resource"},
		{"resource", 2, "resources"},
		{"file", 1, "file"},
		{"file", 5, "files"},
	}
	for _, tt := range tests {
		got := plural(tt.word, tt.count)
		if got != tt.want {
			t.Errorf("plural(%q, %d) = %q, want %q", tt.word, tt.count, got, tt.want)
		}
	}
}

func TestTextRenderer_Render(t *testing.T) {
	var buf bytes.Buffer
	r := &TextRenderer{Writer: &buf}

	output := apply.Output{
		FileCount:     2,
		ResourceCount: 3,
		Groups: []apply.OutputGroup{
			{
				Action: apply.ActionCreate,
				Items: []apply.OutputItem{
					{ResourceID: "instance/web"},
					{ResourceID: "instance/db", Note: "launch"},
				},
			},
			{
				Action: apply.ActionUpdate,
				Items: []apply.OutputItem{
					{
						ResourceID: "network/br0",
						Changes: []incus.DiffChange{
							{Path: "config.ipv4.address", Old: "10.0.0.1/24", New: "10.0.1.1/24", Action: "modify"},
						},
					},
				},
			},
		},
		Summary: "Summary: 2 to create, 1 to update.",
	}

	if err := r.Render(output); err != nil {
		t.Fatal(err)
	}

	result := buf.String()

	checks := []struct {
		label    string
		contains string
	}{
		{"header", "Found 3 resources in 2 files."},
		{"create group", "create (2):"},
		{"update group", "update (1):"},
		{"create prefix", "+ instance/web"},
		{"update prefix", "~ network/br0"},
		{"launch note", "launch"},
		{"summary", "Summary: 2 to create, 1 to update."},
	}
	for _, c := range checks {
		if !strings.Contains(result, c.contains) {
			t.Errorf("missing %s: expected output to contain %q", c.label, c.contains)
		}
	}
}

func TestTextRenderer_RenderNoteAfterChangesUsesTreeBranch(t *testing.T) {
	var buf bytes.Buffer
	r := &TextRenderer{Writer: &buf}

	output := apply.Output{
		FileCount:     1,
		ResourceCount: 1,
		Groups: []apply.OutputGroup{{
			Action: apply.ActionReplace,
			Items: []apply.OutputItem{{
				ResourceID: "default:instance/wordpress",
				Changes: []incus.DiffChange{{
					Path:   "setup",
					Old:    []any{map[string]any{"action": "exec"}},
					New:    []any{map[string]any{"action": "exec"}, map[string]any{"action": "file_push"}},
					Action: "modify",
				}},
				Note: "setup, recreate required",
			}},
		}},
	}

	if err := r.Render(output); err != nil {
		t.Fatal(err)
	}

	result := buf.String()
	if !strings.Contains(result, "      ├─ setup:") {
		t.Fatalf("expected change line to use branch connector when note follows, got %q", result)
	}
	if !strings.Contains(result, "      └─ setup, recreate required") {
		t.Fatalf("expected note line to use last connector, got %q", result)
	}
}

func TestTextRenderer_Quiet(t *testing.T) {
	var buf bytes.Buffer
	r := &TextRenderer{Writer: &buf, Quiet: true}

	output := apply.Output{
		ResourceCount: 1,
		Groups: []apply.OutputGroup{
			{Action: apply.ActionCreate, Items: []apply.OutputItem{{ResourceID: "instance/x"}}},
		},
	}

	if err := r.Render(output); err != nil {
		t.Fatal(err)
	}

	if buf.Len() != 0 {
		t.Errorf("expected no output in quiet mode, got %q", buf.String())
	}
}

func TestJSONRenderer_Render(t *testing.T) {
	var buf bytes.Buffer
	r := &JSONRenderer{Writer: &buf}

	output := apply.Output{
		FileCount:     1,
		ResourceCount: 1,
		Groups: []apply.OutputGroup{
			{Action: apply.ActionCreate, Items: []apply.OutputItem{{ResourceID: "instance/test"}}},
		},
		Summary: "Summary: 1 to create.",
	}

	if err := r.Render(output); err != nil {
		t.Fatal(err)
	}

	var decoded apply.Output
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	if decoded.FileCount != 1 {
		t.Errorf("file_count = %d, want 1", decoded.FileCount)
	}
	if decoded.ResourceCount != 1 {
		t.Errorf("resource_count = %d, want 1", decoded.ResourceCount)
	}
	if len(decoded.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(decoded.Groups))
	}
	if decoded.Groups[0].Action != apply.ActionCreate {
		t.Errorf("action = %q, want %q", decoded.Groups[0].Action, apply.ActionCreate)
	}
	if decoded.Summary != "Summary: 1 to create." {
		t.Errorf("summary = %q, want %q", decoded.Summary, "Summary: 1 to create.")
	}
}
