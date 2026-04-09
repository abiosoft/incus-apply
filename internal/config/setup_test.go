package config

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveSetupSourcePathRelativeToConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "instance.yaml")
	resolved, err := ResolveSetupSourcePath("./files/Caddyfile", configPath)
	if err != nil {
		t.Fatalf("ResolveSetupSourcePath() error = %v", err)
	}
	want := filepath.Join(dir, "files", "Caddyfile")
	if resolved != want {
		t.Fatalf("resolved = %q, want %q", resolved, want)
	}
}

func TestResolveSetupSourcePathRejectsRelativeStdinSource(t *testing.T) {
	_, err := ResolveSetupSourcePath("./files/Caddyfile", "stdin")
	if err == nil {
		t.Fatal("ResolveSetupSourcePath() error = nil, want non-nil")
	}
}

func TestValidateSetupSourceAcceptsExistingPath(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "instance.yaml")
	sourcePath := filepath.Join(dir, "files", "Caddyfile")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte("content"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	if err := ValidateSetupSource(SetupAction{Source: "./files/Caddyfile"}, configPath); err != nil {
		t.Fatalf("ValidateSetupSource() error = %v", err)
	}
}

func TestValidateSetupSourceRejectsMissingPath(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "instance.yaml")
	err := ValidateSetupSource(SetupAction{Source: "./files/Caddyfile"}, configPath)
	if err == nil {
		t.Fatal("ValidateSetupSource() error = nil, want non-nil")
	}
}

func TestSetupActionSnapshotHashesInlineContent(t *testing.T) {
	action := SetupAction{
		Action:  SetupActionPushFile,
		When:    SetupWhenUpdate,
		Path:    "/etc/app.conf",
		Content: "hello world",
	}

	snapshot, err := SetupActionSnapshot(action, "")
	if err != nil {
		t.Fatalf("SetupActionSnapshot() error = %v", err)
	}
	if _, ok := snapshot["content"]; ok {
		if snapshot["content"] == "hello world" {
			t.Fatalf("snapshot = %#v, want raw content omitted", snapshot)
		}
	}
	sum := sha256.Sum256([]byte("hello world"))
	want := setupHashPrefix + hex.EncodeToString(sum[:])[:setupHashLength-2] + "11"
	if snapshot["content"] != want {
		t.Fatalf("content = %v, want %q", snapshot["content"], want)
	}
}

func TestSetupActionSnapshotHashesExecScript(t *testing.T) {
	action := SetupAction{
		Action: SetupActionExec,
		When:   SetupWhenAlways,
		Script: "echo hello world",
	}

	snapshot, err := SetupActionSnapshot(action, "")
	if err != nil {
		t.Fatalf("SetupActionSnapshot() error = %v", err)
	}
	if _, ok := snapshot["script"]; ok {
		if snapshot["script"] == "echo hello world" {
			t.Fatalf("snapshot = %#v, want raw script omitted", snapshot)
		}
	}
	sum := sha256.Sum256([]byte("echo hello world"))
	want := setupHashPrefix + hex.EncodeToString(sum[:])[:setupHashLength-2] + "16"
	if snapshot["script"] != want {
		t.Fatalf("script = %v, want %q", snapshot["script"], want)
	}
}

func TestSetupHashValueEmbedsOriginalLength(t *testing.T) {
	value := strings.Repeat("x", 103)
	hash := sha256.Sum256([]byte(value))
	want := setupHashPrefix + hex.EncodeToString(hash[:])[:setupHashLength-3] + "103"
	if got := setupHashValue(value); got != want {
		t.Fatalf("setupHashValue() = %q, want %q", got, want)
	}
}

func TestSetupActionSnapshotIncludesRequiredWhenFalse(t *testing.T) {
	required := false
	action := SetupAction{Action: SetupActionExec, When: SetupWhenAlways, Required: &required, Script: "echo hi"}

	snapshot, err := SetupActionSnapshot(action, "")
	if err != nil {
		t.Fatalf("SetupActionSnapshot() error = %v", err)
	}
	if snapshot["required"] != false {
		t.Fatalf("required = %v, want false", snapshot["required"])
	}
}
