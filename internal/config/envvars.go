package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

	sourceDir := ""
	if v.SourceFile != "" {
		sourceDir = filepath.Dir(v.SourceFile)
	}

	for _, path := range v.Files {
		// Interpolate shell env in the path itself (e.g., files: ["${HOME}/.env"])
		resolved, err := Interpolate([]byte(path), shellEnv)
		if err != nil {
			return nil, err
		}
		envPath := string(resolved)
		if sourceDir != "" && !filepath.IsAbs(envPath) {
			envPath = filepath.Join(sourceDir, envPath)
		}
		vars, err := godotenv.Read(envPath)
		if err != nil {
			return nil, &EnvFileError{Path: envPath, Err: err}
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
		val, err := resolveDynamicEntry(entry, sourceDir)
		if err != nil {
			return nil, fmt.Errorf("resolving computed var %q: %w", key, err)
		}
		merged[key] = val
	}

	return merged, nil
}

// resolveDynamicEntry executes the source processor for a DynamicEntry and
// applies any requested output format transformation.
// sourceDir is the directory of the vars document; relative File paths are
// resolved against it. An empty sourceDir leaves relative paths as-is.
func resolveDynamicEntry(entry DynamicEntry, sourceDir string) (string, error) {
	var raw string
	switch {
	case entry.File != "":
		path := entry.File
		if sourceDir != "" && !filepath.IsAbs(path) {
			path = filepath.Join(sourceDir, path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("reading file: %w", err)
		}
		raw = strings.TrimRight(string(data), "\n")
	case entry.Incus != "":
		args, err := validateIncusCommand(entry.Incus)
		if err != nil {
			return "", err
		}
		out, err := exec.Command("incus", args...).Output()
		if err != nil {
			return "", fmt.Errorf("running incus %s: %w", entry.Incus, err)
		}
		raw = strings.TrimRight(string(out), "\n")
	default:
		return "", fmt.Errorf("computed entry has no source processor (file, incus)")
	}
	return applyDynamicFormat(raw, entry.Format)
}

// allowedIncusPatterns defines the commands permitted in computed.incus entries.
// Each pattern matches against the space-split tokens of the incus argument string.
// tokenMatch is the fixed prefix tokens that must match exactly; extraArgs is
// the number of additional safe arguments permitted (0 = none, -1 = exactly 1 required).
//
// To extend: add a new incusPattern entry below.
var allowedIncusPatterns = []incusPattern{
	// incus remote get-<subcommand>   (e.g. get-client-certificate, get-default)
	// The first token is "remote", second must start with "get-". No extra args.
	{matchFn: matchRemoteGet},
	// incus config get <key>   — exactly one extra safe argument required
	{matchFn: matchConfigGet},
}

type incusPattern struct {
	// matchFn validates the token slice and returns the final arg list if valid.
	matchFn func(tokens []string) ([]string, bool)
}

func matchRemoteGet(tokens []string) ([]string, bool) {
	if len(tokens) == 2 &&
		tokens[0] == "remote" &&
		strings.HasPrefix(tokens[1], "get-") &&
		isSafeArg(tokens[1]) {
		return tokens, true
	}
	return nil, false
}

func matchConfigGet(tokens []string) ([]string, bool) {
	if len(tokens) == 3 &&
		tokens[0] == "config" &&
		tokens[1] == "get" &&
		isSafeArg(tokens[2]) {
		return tokens, true
	}
	return nil, false
}

// validateIncusCommand checks that cmd matches an allowed incus pattern and
// returns the split argument slice ready to pass to exec.Command.
func validateIncusCommand(cmd string) ([]string, error) {
	tokens := strings.Fields(strings.TrimSpace(cmd))
	for _, p := range allowedIncusPatterns {
		if args, ok := p.matchFn(tokens); ok {
			return args, nil
		}
	}
	return nil, fmt.Errorf("incus command not allowed: %q", cmd)
}

// isSafeArg reports whether s is a safe single argument: only alphanumeric
// characters, dots, hyphens, and underscores.
func isSafeArg(s string) bool {
	for _, c := range s {
		if ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') ||
			('0' <= c && c <= '9') || c == '.' || c == '-' || c == '_' {
			continue
		}
		return false
	}
	return len(s) > 0
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
