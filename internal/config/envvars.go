package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/joho/godotenv"
)

// ResolveVars builds a variable map from a VarsConfig document.
//
// Resolution order (later sources win):
//  1. Each path in Files, loaded in order via godotenv
//  2. Each entry in Vars, applied in declaration order
//
// Within this function, shell environment variables may be referenced
// via $VAR / ${VAR} syntax in files paths and in vars values.
func ResolveVars(v Vars) (map[string]string, error) {
	shellEnv := shellEnvironment()
	merged := map[string]string{}

	for _, path := range v.Files {
		// Interpolate shell env in the path itself (e.g., files: ["${HOME}/.env"])
		resolved, err := Interpolate([]byte(path), shellEnv)
		if err != nil {
			return nil, err
		}
		vars, err := godotenv.Read(string(resolved))
		if err != nil {
			return nil, &EnvFileError{Path: string(resolved), Err: err}
		}
		for k, val := range vars {
			merged[k] = val
		}
	}

	// Vars declared inline; values can reference shell env
	for k, raw := range v.Vars {
		resolved, err := Interpolate([]byte(raw), shellEnv)
		if err != nil {
			return nil, err
		}
		merged[k] = string(resolved)
	}

	// Computed entries: resolved last so they always win
	for key, entry := range v.Computed {
		val, err := resolveDynamicEntry(entry)
		if err != nil {
			return nil, fmt.Errorf("resolving computed var %q: %w", key, err)
		}
		merged[key] = val
	}

	return merged, nil
}

// resolveDynamicEntry executes the source processor for a DynamicEntry and
// applies any requested output format transformation.
func resolveDynamicEntry(entry DynamicEntry) (string, error) {
	var raw string
	switch {
	case entry.File != "":
		data, err := os.ReadFile(entry.File)
		if err != nil {
			return "", fmt.Errorf("reading file: %w", err)
		}
		raw = strings.TrimRight(string(data), "\n")
	case entry.Incus != "":
		args := strings.Fields(entry.Incus)
		out, err := exec.Command("incus", args...).Output() // #nosec G204 -- args from trusted config file
		if err != nil {
			return "", fmt.Errorf("running incus %s: %w", entry.Incus, err)
		}
		raw = strings.TrimRight(string(out), "\n")
	default:
		return "", fmt.Errorf("computed entry has no source processor (file, incus)")
	}
	return applyDynamicFormat(raw, entry.Format)
}

// applyDynamicFormat transforms raw into the requested format.
// Supported formats: "" (raw, no-op) and "base64".
func applyDynamicFormat(val, format string) (string, error) {
	switch format {
	case "":
		return val, nil
	case "base64":
		return base64.StdEncoding.EncodeToString([]byte(val)), nil
	default:
		return "", fmt.Errorf("unsupported format %q (supported: base64)", format)
	}
}

// shellEnvironment returns the current process environment as a map.
func shellEnvironment() map[string]string {
	env := map[string]string{}
	for _, entry := range os.Environ() {
		k, v, _ := strings.Cut(entry, "=")
		env[k] = v
	}
	return env
}

// EnvFileError is returned when an files entry cannot be read.
type EnvFileError struct {
	Path string
	Err  error
}

func (e *EnvFileError) Error() string {
	return "reading env file " + e.Path + ": " + e.Err.Error()
}

func (e *EnvFileError) Unwrap() error { return e.Err }
