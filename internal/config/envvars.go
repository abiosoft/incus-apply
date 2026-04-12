package config

import (
	"os"
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

	return merged, nil
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
