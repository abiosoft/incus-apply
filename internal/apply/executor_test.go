package apply

import (
	"strings"
	"testing"

	"github.com/abiosoft/incus-apply/internal/config"
)

func TestUpsertSummary(t *testing.T) {
	tests := []struct {
		creates int
		updates int
		want    string
	}{
		{0, 0, ""},
		{3, 0, "Summary: 3 to create."},
		{0, 2, "Summary: 2 to update."},
		{5, 3, "Summary: 5 to create, 3 to update."},
	}
	for _, tt := range tests {
		got := (result{created: tt.creates, updated: tt.updates}).upsertSummary()
		if got != tt.want {
			t.Errorf("result.upsertSummary(%d, %d) = %q, want %q", tt.creates, tt.updates, got, tt.want)
		}
	}
}

func TestDeleteSummary(t *testing.T) {
	tests := []struct {
		deletes int
		want    string
	}{
		{0, ""},
		{1, "Summary: 1 to delete."},
		{5, "Summary: 5 to delete."},
	}
	for _, tt := range tests {
		got := (result{deleted: tt.deletes}).deleteSummary()
		if got != tt.want {
			t.Errorf("result.deleteSummary(%d) = %q, want %q", tt.deletes, got, tt.want)
		}
	}
}

func TestOutput_AddGroup(t *testing.T) {
	var output Output

	output.AddGroup(ActionCreate, nil)
	if len(output.Groups) != 0 {
		t.Fatalf("expected 0 groups for nil items, got %d", len(output.Groups))
	}

	output.AddGroup(ActionCreate, []OutputItem{})
	if len(output.Groups) != 0 {
		t.Fatalf("expected 0 groups for empty items, got %d", len(output.Groups))
	}

	output.AddGroup(ActionCreate, []OutputItem{{ResourceID: "instance/test"}})
	if len(output.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(output.Groups))
	}
	if output.Groups[0].Action != ActionCreate {
		t.Errorf("expected action %q, got %q", ActionCreate, output.Groups[0].Action)
	}
}

func TestOptions_IsDiffOnly(t *testing.T) {
	tests := []struct {
		diff string
		want bool
	}{
		{"", false},
		{"text", true},
		{"json", true},
	}
	for _, tt := range tests {
		opts := &Options{Diff: tt.diff}
		if got := opts.IsDiffOnly(); got != tt.want {
			t.Errorf("Options{Diff: %q}.IsDiffOnly() = %v, want %v", tt.diff, got, tt.want)
		}
	}
}

func TestOptions_IsJSONDiff(t *testing.T) {
	tests := []struct {
		diff string
		want bool
	}{
		{"", false},
		{"text", false},
		{"json", true},
	}
	for _, tt := range tests {
		opts := &Options{Diff: tt.diff}
		if got := opts.IsJSONDiff(); got != tt.want {
			t.Errorf("Options{Diff: %q}.IsJSONDiff() = %v, want %v", tt.diff, got, tt.want)
		}
	}
}

func TestConfirmApplyYesBypassesPrompt(t *testing.T) {
	executor := defaultExecutor{opts: Options{Yes: true}}

	if err := executor.confirmApply("Proceed to apply these changes"); err != nil {
		t.Fatalf("confirmApply() error = %v", err)
	}
}

func TestConfirmApplyNonInteractiveRequiresYes(t *testing.T) {
	executor := defaultExecutor{opts: Options{}, interactive: false}

	err := executor.confirmApply("Proceed to apply these changes")
	if err == nil {
		t.Fatal("confirmApply() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "--yes or --diff") {
		t.Fatalf("confirmApply() error = %q, want message to mention --yes or --diff", err.Error())
	}
}

func TestFormatResourceID_ProjectScopedUsesDefaultProject(t *testing.T) {
	res := &config.Resource{Base: config.Base{Type: "instance", Name: "web"}}
	if got := formatResourceID(res); got != "default:instance/web" {
		t.Fatalf("formatResourceID() = %q, want %q", got, "default:instance/web")
	}
}

func TestFormatResourceID_GlobalResourceOmitsProject(t *testing.T) {
	res := &config.Resource{Base: config.Base{Type: "storage-pool", Name: "fast", Project: "other"}}
	if got := formatResourceID(res); got != "storage-pool/fast" {
		t.Fatalf("formatResourceID() = %q, want %q", got, "storage-pool/fast")
	}
}

func TestFormatResourceID_StorageVolumeIncludesProjectAndPool(t *testing.T) {
	res := &config.Resource{Base: config.Base{Type: "storage-volume", Name: "data"}, StorageResourceFields: config.StorageResourceFields{Pool: "pool1"}}
	if got := formatResourceID(res); got != "default:storage-volume/pool1/data" {
		t.Fatalf("formatResourceID() = %q, want %q", got, "default:storage-volume/pool1/data")
	}
}

func TestFormatResourceID_NetworkForwardIncludesProjectAndNetwork(t *testing.T) {
	res := &config.Resource{Base: config.Base{Type: "network-forward"}, InstanceFields: config.InstanceFields{Network: "uplink"}, NetworkForwardFields: config.NetworkForwardFields{ListenAddress: "198.51.100.10"}}
	if got := formatResourceID(res); got != "default:network-forward/uplink/198.51.100.10" {
		t.Fatalf("formatResourceID() = %q, want %q", got, "default:network-forward/uplink/198.51.100.10")
	}
}

func TestValidateUniqueResources_DuplicateSameProjectFails(t *testing.T) {
	resources := []*config.Resource{
		{Base: config.Base{Type: "instance", Name: "web", SourceFile: "one.yaml"}},
		{Base: config.Base{Type: "instance", Name: "web", SourceFile: "two.yaml"}},
	}

	err := validateUniqueResources(resources)
	if err == nil {
		t.Fatal("validateUniqueResources() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "default:instance/web") {
		t.Fatalf("validateUniqueResources() error = %q, want duplicate resource id", err.Error())
	}
	if !strings.Contains(err.Error(), "one.yaml") || !strings.Contains(err.Error(), "two.yaml") {
		t.Fatalf("validateUniqueResources() error = %q, want both file names", err.Error())
	}
}

func TestValidateUniqueResources_DifferentProjectsAllowed(t *testing.T) {
	resources := []*config.Resource{
		{Base: config.Base{Type: "instance", Name: "web", Project: "app1", SourceFile: "one.yaml"}},
		{Base: config.Base{Type: "instance", Name: "web", Project: "app2", SourceFile: "two.yaml"}},
	}

	if err := validateUniqueResources(resources); err != nil {
		t.Fatalf("validateUniqueResources() error = %v", err)
	}
}
