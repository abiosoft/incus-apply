package config

import (
	"bytes"
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
//  3. Each entry in Commands, executed via sh -c and stdout used as the value
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

	// Commands: run each value via sh -c and use stdout as the variable value
	for k, cmdStr := range v.Commands {
		out, err := runCommand(cmdStr)
		if err != nil {
			return nil, &CommandError{Key: k, Command: cmdStr, Err: err}
		}
		merged[k] = out
	}

	return merged, nil
}

// runCommand executes cmdStr as a single argument to sh -c and returns
// the trimmed stdout output.
func runCommand(cmdStr string) (string, error) {
	var buf bytes.Buffer
	cmd := exec.Command("sh", "-c", cmdStr)
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimRight(buf.String(), "\n"), nil
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

// CommandError is returned when a commands entry fails to execute.
type CommandError struct {
	Key     string
	Command string
	Err     error
}

func (e *CommandError) Error() string {
	return fmt.Sprintf("running command for %q (%s): %s", e.Key, e.Command, e.Err.Error())
}

func (e *CommandError) Unwrap() error { return e.Err }
