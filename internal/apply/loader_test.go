package apply

import (
	"testing"

	"github.com/abiosoft/incus-apply/internal/config"
)

func TestResolveAndInterpolateScopesVarsPerFile(t *testing.T) {
	results := []*config.FileResult{
		{
			SourceFile: "shared.incus.yaml",
			Vars: []*config.Vars{
				{Vars: map[string]string{"GLOBAL": "global"}, Global: true, SourceFile: "shared.incus.yaml"},
			},
		},
		{
			SourceFile: "one.incus.yaml",
			Vars: []*config.Vars{
				{Vars: map[string]string{"NAME": "one"}, SourceFile: "one.incus.yaml"},
			},
			Resources: []*config.Resource{
				{Base: config.Base{Type: "instance", Name: "one", SourceFile: "one.incus.yaml", Config: map[string]string{
					"user.global": "$GLOBAL",
					"user.name":   "$NAME",
					"user.home":   "$HOME",
				}}},
			},
		},
		{
			SourceFile: "two.incus.yaml",
			Vars: []*config.Vars{
				{Vars: map[string]string{"NAME": "two"}, SourceFile: "two.incus.yaml"},
			},
			Resources: []*config.Resource{
				{Base: config.Base{Type: "instance", Name: "two", SourceFile: "two.incus.yaml", Config: map[string]string{
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
			SourceFile: "resources.incus.yaml",
			Resources: []*config.Resource{
				{Base: config.Base{Type: "instance", Name: "web", SourceFile: "resources.incus.yaml"}},
				{Base: config.Base{Type: "profile", Name: "base", SourceFile: "resources.incus.yaml"}},
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
			SourceFile: "app.incus.yaml",
			Vars: []*config.Vars{
				{Vars: map[string]string{"SEED": seed}, SourceFile: "app.incus.yaml"},
			},
			Resources: []*config.Resource{
				{
					Base: config.Base{Type: "instance", Name: "app", SourceFile: "app.incus.yaml"},
					InstanceFields: config.InstanceFields{
						Image: "images:alpine/3.19",
						Setup: []config.SetupAction{{
							Action:  config.SetupActionPushFile,
							When:    config.SetupWhenCreate,
							Path:    "/seed/incus.yaml",
							Content: "$SEED",
						}},
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
	if got := resources[0].Setup[0].Content; got != seed {
		t.Fatalf("setup content = %q, want %q", got, seed)
	}
	if got := resources[0].Setup[0].Path; got != "/seed/incus.yaml" {
		t.Fatalf("setup path = %q, want %q", got, "/seed/incus.yaml")
	}
}

func TestResolveAndInterpolate_AllowsJSONContentInSingleLineScalar(t *testing.T) {
	seed := "{\n  \"name\": \"app\",\n  \"enabled\": true\n}"
	results := []*config.FileResult{
		{
			SourceFile: "app.incus.yaml",
			Vars: []*config.Vars{
				{Vars: map[string]string{"SEED": seed}, SourceFile: "app.incus.yaml"},
			},
			Resources: []*config.Resource{
				{
					Base: config.Base{Type: "instance", Name: "app", SourceFile: "app.incus.yaml"},
					InstanceFields: config.InstanceFields{
						Image: "images:alpine/3.19",
						Setup: []config.SetupAction{{
							Action:  config.SetupActionPushFile,
							When:    config.SetupWhenCreate,
							Path:    "/seed/data.json",
							Content: "$SEED",
						}},
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
	if got := resources[0].Setup[0].Content; got != seed {
		t.Fatalf("setup content = %q, want %q", got, seed)
	}
}
