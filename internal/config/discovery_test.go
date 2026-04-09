package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoveryFindsPlainYAMLFiles(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"instance.yaml":       "type: instance\nname: web\n",
		"network.yml":         "type: network\nname: net0\n",
		"stack.json":          `{"type":"profile","name":"base"}`,
		"README.md":           "# not a config file",
		"docker-compose.yaml": "version: \"3\"\nservices:\n  web:\n    image: nginx\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("os.WriteFile(%s): %v", name, err)
		}
	}

	found, err := NewDiscovery(false).FindFiles([]string{dir})
	if err != nil {
		t.Fatalf("FindFiles() error = %v", err)
	}

	// .yaml, .yml, .json should be found; .md should not
	if len(found) != 4 {
		t.Fatalf("found %d files, want 4 (.yaml, .yml, .json x2); got %v", len(found), found)
	}
	for _, f := range found {
		ext := filepath.Ext(f)
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			t.Errorf("unexpected file in result: %s", f)
		}
	}
}

func TestDiscoveryExplicitFilePassedRegardlessOfExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "myconfig")
	if err := os.WriteFile(path, []byte("type: instance\nname: app\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}

	found, err := NewDiscovery(false).FindFiles([]string{path})
	if err != nil {
		t.Fatalf("FindFiles() error = %v", err)
	}
	if len(found) != 1 || found[0] != path {
		t.Fatalf("FindFiles() = %v, want [%s]", found, path)
	}
}
