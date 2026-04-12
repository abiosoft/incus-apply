package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveVars_envFile(t *testing.T) {
	envFile := writeTempEnv(t, "FOO=bar\nBAZ=qux\n")

	got, err := ResolveVars(Vars{Files: []string{envFile}})
	if err != nil {
		t.Fatal(err)
	}
	if got["FOO"] != "bar" {
		t.Errorf("FOO = %q, want %q", got["FOO"], "bar")
	}
	if got["BAZ"] != "qux" {
		t.Errorf("BAZ = %q, want %q", got["BAZ"], "qux")
	}
}

func TestResolveVars_multipleEnvFilesLaterOverrides(t *testing.T) {
	fileA := writeTempEnv(t, "KEY=from_a\nONLY_A=yes\n")
	fileB := writeTempEnv(t, "KEY=from_b\nONLY_B=yes\n")

	got, err := ResolveVars(Vars{Files: []string{fileA, fileB}})
	if err != nil {
		t.Fatal(err)
	}
	if got["KEY"] != "from_b" {
		t.Errorf("KEY = %q, want %q (later file should win)", got["KEY"], "from_b")
	}
	if got["ONLY_A"] != "yes" {
		t.Errorf("ONLY_A = %q, want %q", got["ONLY_A"], "yes")
	}
}

func TestResolveVars_inlineVarsOverrideEnvFile(t *testing.T) {
	envFile := writeTempEnv(t, "KEY=from_file\n")

	got, err := ResolveVars(Vars{
		Files: []string{envFile},
		Vars:  map[string]string{"KEY": "from_vars"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got["KEY"] != "from_vars" {
		t.Errorf("KEY = %q, want %q (vars should win over env_files)", got["KEY"], "from_vars")
	}
}

func TestResolveVars_shellEnvInVarsValues(t *testing.T) {
	t.Setenv("MY_SECRET", "s3cret")

	got, err := ResolveVars(Vars{
		Vars: map[string]string{"DB_PASS": "${MY_SECRET}"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got["DB_PASS"] != "s3cret" {
		t.Errorf("DB_PASS = %q, want %q", got["DB_PASS"], "s3cret")
	}
}

func TestResolveVars_missingEnvFile(t *testing.T) {
	_, err := ResolveVars(Vars{Files: []string{"/nonexistent/path/.env"}})
	if err == nil {
		t.Fatal("expected error for missing env file, got nil")
	}
	if _, ok := err.(*EnvFileError); !ok {
		t.Errorf("expected *EnvFileError, got %T: %v", err, err)
	}
}

// writeTempEnv writes content to a temp env file and returns its path.
func writeTempEnv(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestResolveVars_computedFile(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "cert.pem")
	if err := os.WriteFile(tmp, []byte("MYCERT\n"), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := ResolveVars(Vars{
		Computed: map[string]DynamicEntry{
			"CERT": {File: tmp},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got["CERT"] != "MYCERT" {
		t.Errorf("CERT = %q, want %q", got["CERT"], "MYCERT")
	}
}

func TestResolveVars_computedFileBase64(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "cert.pem")
	if err := os.WriteFile(tmp, []byte("MYCERT\n"), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := ResolveVars(Vars{
		Computed: map[string]DynamicEntry{
			"CERT_B64": {File: tmp, Format: "base64"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	// base64("MYCERT") = "TVlDRVJU"
	if got["CERT_B64"] != "TVlDRVJU" {
		t.Errorf("CERT_B64 = %q, want %q", got["CERT_B64"], "TVlDRVJU")
	}
}

func TestResolveVars_computedMissingFile(t *testing.T) {
	_, err := ResolveVars(Vars{
		Computed: map[string]DynamicEntry{
			"X": {File: "/nonexistent/file.txt"},
		},
	})
	if err == nil {
		t.Fatal("expected error for missing dynamic file, got nil")
	}
}

func TestResolveVars_computedNoSource(t *testing.T) {
	_, err := ResolveVars(Vars{
		Computed: map[string]DynamicEntry{
			"X": {Format: "base64"},
		},
	})
	if err == nil {
		t.Fatal("expected error for dynamic entry with no source, got nil")
	}
}

func TestResolveVars_computedUnsupportedFormat(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "data.txt")
	if err := os.WriteFile(tmp, []byte("value"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := ResolveVars(Vars{
		Computed: map[string]DynamicEntry{
			"X": {File: tmp, Format: "hex"},
		},
	})
	if err == nil {
		t.Fatal("expected error for unsupported format, got nil")
	}
}
