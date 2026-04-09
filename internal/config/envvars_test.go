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

func TestResolveVars_commandOutput(t *testing.T) {
	got, err := ResolveVars(Vars{
		Commands: map[string]string{"GREETING": "echo hello"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got["GREETING"] != "hello" {
		t.Errorf("GREETING = %q, want %q", got["GREETING"], "hello")
	}
}

func TestResolveVars_commandOverridesVars(t *testing.T) {
	got, err := ResolveVars(Vars{
		Vars:     map[string]string{"KEY": "from_vars"},
		Commands: map[string]string{"KEY": "echo from_command"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got["KEY"] != "from_command" {
		t.Errorf("KEY = %q, want %q (commands should win over vars)", got["KEY"], "from_command")
	}
}

func TestResolveVars_commandFailure(t *testing.T) {
	_, err := ResolveVars(Vars{
		Commands: map[string]string{"BAD": "exit 1"},
	})
	if err == nil {
		t.Fatal("expected error for failing command, got nil")
	}
	if _, ok := err.(*CommandError); !ok {
		t.Errorf("expected *CommandError, got %T: %v", err, err)
	}
}

func TestResolveVars_commandTrailingNewlineStripped(t *testing.T) {
	got, err := ResolveVars(Vars{
		Commands: map[string]string{"VAL": "printf 'no-newline'"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got["VAL"] != "no-newline" {
		t.Errorf("VAL = %q, want %q", got["VAL"], "no-newline")
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
