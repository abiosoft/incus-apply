package config

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Supported file extensions for incus configuration files.
const (
	extYAML = ".yaml"
	extYML  = ".yml"
	extJSON = ".json"
)

// Discovery finds YAML and JSON configuration files containing incus resources.
type Discovery struct {
	recursive bool
}

// NewDiscovery creates a new file discovery instance.
// If recursive is true, directories are searched recursively.
func NewDiscovery(recursive bool) *Discovery {
	return &Discovery{recursive: recursive}
}

// FindFiles finds all incus config files in the given paths.
// Paths can be individual files or directories.
// Returns a sorted, deduplicated list of absolute file paths.
func (d Discovery) FindFiles(paths []string) ([]string, error) {
	var files []string
	seen := make(map[string]bool)

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}

		if info.IsDir() {
			dirFiles, err := d.findInDirectory(path)
			if err != nil {
				return nil, err
			}
			for _, f := range dirFiles {
				if !seen[f] {
					seen[f] = true
					files = append(files, f)
				}
			}
		} else {
			if !seen[path] {
				seen[path] = true
				files = append(files, path)
			}
		}
	}

	// Sort for deterministic processing order
	sort.Strings(files)
	return files, nil
}

// findInDirectory finds config files within a directory.
// Respects the recursive setting and skips hidden directories.
func (d Discovery) findInDirectory(dir string) ([]string, error) {
	var files []string

	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return d.handleDirectory(path, dir, info)
		}

		if isIncusConfigFile(info.Name()) {
			files = append(files, path)
		}
		return nil
	}

	if err := filepath.Walk(dir, walkFn); err != nil {
		return nil, err
	}

	return files, nil
}

// handleDirectory determines whether to enter a subdirectory during traversal.
// Skips hidden directories and non-recursive subdirectory traversal.
func (d Discovery) handleDirectory(path, rootDir string, info os.FileInfo) error {
	name := info.Name()

	// Skip hidden directories (except root ".")
	if strings.HasPrefix(name, ".") && name != "." {
		return filepath.SkipDir
	}

	// Skip subdirectories if not in recursive mode
	if !d.recursive && path != rootDir {
		return filepath.SkipDir
	}

	return nil
}

// isIncusConfigFile checks if the filename has a yaml or json extension.
func isIncusConfigFile(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, extYAML) ||
		strings.HasSuffix(lower, extYML) ||
		strings.HasSuffix(lower, extJSON)
}
