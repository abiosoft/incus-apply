package apply

import (
	"testing"

	"github.com/abiosoft/incus-apply/internal/config"
)

func TestResolveAndInterpolateScopesVarsPerFile(t *testing.T) {
	results := []*config.FileResult{
		{
			SourceFile: "shared.yaml",
			Vars: []*config.Vars{
				{Vars: map[string]string{"GLOBAL": "global"}, Global: true, SourceFile: "shared.yaml"},
			},
		},
		{
			SourceFile: "one.yaml",
			Vars: []*config.Vars{
				{Vars: map[string]string{"NAME": "one"}, SourceFile: "one.yaml"},
			},
			Resources: []*config.Resource{
				{Base: config.Base{Type: "instance", Name: "one", SourceFile: "one.yaml", Config: map[string]string{
					"user.global": "$GLOBAL",
					"user.name":   "$NAME",
					"user.home":   "$HOME",
				}}},
			},
		},
		{
			SourceFile: "two.yaml",
			Vars: []*config.Vars{
				{Vars: map[string]string{"NAME": "two"}, SourceFile: "two.yaml"},
			},
			Resources: []*config.Resource{
				{Base: config.Base{Type: "instance", Name: "two", SourceFile: "two.yaml", Config: map[string]string{
					"user.global": "$GLOBAL",
					"user.name":   "$NAME",
					"user.home":   "$HOME",
				}}},
			},
		},
	}

	resources, err := resolveAndInterpolate(results)
	if err != nil {
		t.Fatalf("resolveAndInterpolate() error = %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(resources))
	}

	if got := resources[0].Config["user.global"]; got != "global" {
		t.Fatalf("resource 0 global var = %q, want %q", got, "global")
	}
	if got := resources[0].Config["user.name"]; got != "one" {
		t.Fatalf("resource 0 file var = %q, want %q", got, "one")
	}
	if got := resources[0].Config["user.home"]; got != "$HOME" {
		t.Fatalf("resource 0 undeclared var = %q, want %q", got, "$HOME")
	}
	if len(resources[0].PreviewRedactPrefixes) != 1 || resources[0].PreviewRedactPrefixes[0] != "config.environment." {
		t.Fatalf("resource 0 preview redact prefixes = %#v, want [config.environment.]", resources[0].PreviewRedactPrefixes)
	}

	if got := resources[1].Config["user.global"]; got != "global" {
		t.Fatalf("resource 1 global var = %q, want %q", got, "global")
	}
	if got := resources[1].Config["user.name"]; got != "two" {
		t.Fatalf("resource 1 file var = %q, want %q", got, "two")
	}
	if got := resources[1].Config["user.home"]; got != "$HOME" {
		t.Fatalf("resource 1 undeclared var = %q, want %q", got, "$HOME")
	}
	if len(resources[1].PreviewRedactPrefixes) != 1 || resources[1].PreviewRedactPrefixes[0] != "config.environment." {
		t.Fatalf("resource 1 preview redact prefixes = %#v, want [config.environment.]", resources[1].PreviewRedactPrefixes)
	}
}

func TestResolveAndInterpolate_SetsPreviewRedactionOnlyForInstances(t *testing.T) {
	results := []*config.FileResult{
		{
			SourceFile: "resources.yaml",
			Resources: []*config.Resource{
				{Base: config.Base{Type: "instance", Name: "web", SourceFile: "resources.yaml"}},
				{Base: config.Base{Type: "profile", Name: "base", SourceFile: "resources.yaml"}},
			},
		},
	}

	resources, err := resolveAndInterpolate(results)
	if err != nil {
		t.Fatalf("resolveAndInterpolate() error = %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(resources))
	}
	if len(resources[0].PreviewRedactPrefixes) != 1 || resources[0].PreviewRedactPrefixes[0] != "config.environment." {
		t.Fatalf("instance preview redact prefixes = %#v, want [config.environment.]", resources[0].PreviewRedactPrefixes)
	}
	if len(resources[1].PreviewRedactPrefixes) != 0 {
		t.Fatalf("profile preview redact prefixes = %#v, want none", resources[1].PreviewRedactPrefixes)
	}
}

func TestResolveAndInterpolate_AllowsYAMLContentInSingleLineScalar(t *testing.T) {
	seed := "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: app\n"
	results := []*config.FileResult{
		{
			SourceFile: "app.yaml",
			Vars: []*config.Vars{
				{Vars: map[string]string{"SEED": seed}, SourceFile: "app.yaml"},
			},
			Resources: []*config.Resource{
				{
					Base: config.Base{
						Type:       "instance",
						Name:       "app",
						SourceFile: "app.yaml",
						Config:     map[string]string{"cloud-init.user-data": "$SEED"},
					},
					InstanceFields: config.InstanceFields{
						Image: "images:alpine/3.19",
					},
				},
			},
		},
	}

	resources, err := resolveAndInterpolate(results)
	if err != nil {
		t.Fatalf("resolveAndInterpolate() error = %v", err)
	}
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	if got := resources[0].Config["cloud-init.user-data"]; got != seed {
		t.Fatalf("cloud-init.user-data = %q, want %q", got, seed)
	}
}

func TestResolveAndInterpolate_AllowsJSONContentInSingleLineScalar(t *testing.T) {
	seed := "{\n  \"name\": \"app\",\n  \"enabled\": true\n}"
	results := []*config.FileResult{
		{
			SourceFile: "app.yaml",
			Vars: []*config.Vars{
				{Vars: map[string]string{"SEED": seed}, SourceFile: "app.yaml"},
			},
			Resources: []*config.Resource{
				{
					Base: config.Base{
						Type:       "instance",
						Name:       "app",
						SourceFile: "app.yaml",
						Config:     map[string]string{"user.data": "$SEED"},
					},
					InstanceFields: config.InstanceFields{
						Image: "images:alpine/3.19",
					},
				},
			},
		},
	}

	resources, err := resolveAndInterpolate(results)
	if err != nil {
		t.Fatalf("resolveAndInterpolate() error = %v", err)
	}
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	if got := resources[0].Config["user.data"]; got != seed {
		t.Fatalf("user.data = %q, want %q", got, seed)
	}
}
