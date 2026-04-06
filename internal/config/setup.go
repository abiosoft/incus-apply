package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"
)

const setupHashLength = 26
const setupHashPrefix = "hash: "

func ResolveSetupSourcePath(source, sourceFile string) (string, error) {
	if source == "" {
		return "", nil
	}
	if filepath.IsAbs(source) {
		return source, nil
	}
	if sourceFile == "" {
		return "", fmt.Errorf("relative setup source %q requires a source file", source)
	}
	if sourceFile == "stdin" || strings.HasPrefix(sourceFile, "http://") || strings.HasPrefix(sourceFile, "https://") {
		return "", fmt.Errorf("relative setup source %q is not supported for %s", source, sourceFile)
	}

	resolved := filepath.Join(filepath.Dir(sourceFile), source)
	abs, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("resolving setup source %q: %w", source, err)
	}
	return abs, nil
}

func SetupActionSnapshot(action SetupAction, sourceFile string) (map[string]any, error) {
	state := map[string]any{
		"action": action.Action,
		"when":   action.When,
	}
	if !action.IsRequired() {
		state["required"] = false
	}
	if action.Skip {
		state["skip"] = true
	}

	switch action.Action {
	case SetupActionExec:
		state["script"] = setupHashValue(action.Script)
		if action.CWD != "" {
			state["cwd"] = action.CWD
		}
	case SetupActionPushFile:
		state["path"] = action.Path
		if action.Content != "" {
			state["content"] = setupHashValue(action.Content)
		}
		if action.Source != "" {
			state["source"] = action.Source
		}
		if action.Recursive {
			state["recursive"] = true
		}
		if action.UID != nil {
			state["uid"] = *action.UID
		}
		if action.GID != nil {
			state["gid"] = *action.GID
		}
		if action.Mode != "" {
			state["mode"] = string(action.Mode)
		}
	}

	return state, nil
}

func sha256Hex(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// setupHashValue stores a shortened digest with the original character count
// spliced into the end, for example: "hash: 7772eeb2b835fcacc5f43ce103".
func setupHashValue(text string) string {
	hash := sha256Hex(text)
	lengthSuffix := strconv.Itoa(utf8.RuneCountInString(text))
	if len(lengthSuffix) >= setupHashLength {
		return fmt.Sprintf("%s%s", setupHashPrefix, lengthSuffix[len(lengthSuffix)-setupHashLength:])
	}
	prefixLength := setupHashLength - len(lengthSuffix)
	return fmt.Sprintf("%s%s%s", setupHashPrefix, hash[:prefixLength], lengthSuffix)
}

func ValidateSetupSource(action SetupAction, sourceFile string) error {
	if action.Source == "" {
		return nil
	}
	resolved, err := ResolveSetupSourcePath(action.Source, sourceFile)
	if err != nil {
		return err
	}
	if _, err := os.Stat(resolved); err != nil {
		return fmt.Errorf("checking setup source %q: %w", resolved, err)
	}
	return nil
}
