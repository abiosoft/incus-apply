package apply

import (
	"fmt"
	"os"
	"strings"

	"github.com/abiosoft/incus-apply/internal/config"
	"github.com/abiosoft/incus-apply/internal/resource"
	"gopkg.in/yaml.v3"
)

// isURL checks if a path is a URL (http:// or https://).
func isURL(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}

// loadResources discovers and parses all config files from the given paths.
// Returns nil (no error) if no files or resources are found.
// Supports "-" for stdin and URLs (http:// or https://).
//
// The loading flow:
//  1. Parse all files, collecting vars and resource configs (each tagged with SourceFile).
//  2. Resolve vars: build a global env from global vars, then per-file env (global + file-scoped).
//  3. Interpolate each resource config using the env for its source file.
func loadResources(opts *Options) ([]*config.Resource, error) {
	parser := config.NewParser(opts.FetchTimeout)
	var results []*config.FileResult

	var filePaths []string
	for _, f := range opts.Files {
		switch {
		case f == "-":
			result, err := parser.ParseStdin(os.Stdin)
			if err != nil {
				return nil, err
			}
			opts.FileCount++
			results = append(results, result)
		case isURL(f):
			result, err := parser.ParseURL(f)
			if err != nil {
				return nil, err
			}
			opts.FileCount++
			results = append(results, result)
		default:
			filePaths = append(filePaths, f)
		}
	}

	if len(filePaths) > 0 {
		discovery := config.NewDiscovery(opts.Recursive)
		files, err := discovery.FindFiles(filePaths)
		if err != nil {
			return nil, fmt.Errorf("discovering files: %w", err)
		}
		for _, path := range files {
			result, err := parser.ParseFile(path)
			if err != nil {
				return nil, fmt.Errorf("parsing resources: %w", err)
			}
			opts.FileCount++
			results = append(results, result)
		}
	}

	// Resolve variables and interpolate resources
	allResources, err := resolveAndInterpolate(results)
	if err != nil {
		return nil, err
	}

	if len(allResources) == 0 {
		printInfo(opts.Quiet, "No resources found")
		return nil, nil
	}

	if err := validateResourceTypes(allResources); err != nil {
		return nil, err
	}

	return allResources, nil
}

// resolveAndInterpolate resolves vars from all file results and interpolates
// resource configs. Global vars apply to all files; file-scoped vars apply
// only to resources in the same file.
func resolveAndInterpolate(results []*config.FileResult) ([]*config.Resource, error) {
	// Pass 1: resolve all global vars
	globalEnv := map[string]string{}
	for _, r := range results {
		for _, v := range r.Vars {
			if !v.Global {
				continue
			}
			resolved, err := config.ResolveVars(*v)
			if err != nil {
				return nil, fmt.Errorf("resolving global vars in %s: %w", r.SourceFile, err)
			}
			for k, val := range resolved {
				globalEnv[k] = val
			}
		}
	}

	// Pass 2: for each file, merge global + file-scoped vars, then interpolate resources
	var allResources []*config.Resource
	for _, r := range results {
		// Build file-scoped env: start with global, then overlay file-scoped vars
		fileEnv := make(map[string]string, len(globalEnv))
		for k, v := range globalEnv {
			fileEnv[k] = v
		}
		for _, v := range r.Vars {
			if v.Global {
				continue
			}
			resolved, err := config.ResolveVars(*v)
			if err != nil {
				return nil, fmt.Errorf("resolving vars in %s: %w", r.SourceFile, err)
			}
			for k, val := range resolved {
				fileEnv[k] = val
			}
		}

		// Interpolate each resource config's raw fields using the merged env
		for _, res := range r.Resources {
			interpolated, err := interpolateResource(res, fileEnv)
			if err != nil {
				return nil, fmt.Errorf("interpolating %s %q in %s: %w", res.Type, res.Name, res.SourceFile, err)
			}
			setPreviewRedaction(interpolated)
			allResources = append(allResources, interpolated)
		}
	}

	return allResources, nil
}

// interpolateResource re-marshals a ResourceConfig to YAML, expands declared
// variables, preserves undeclared references, and unmarshals back. This
// handles variable substitution in all fields (config values, device
// settings, cloud-init scripts, etc.).
func interpolateResource(res *config.Resource, env map[string]string) (*config.Resource, error) {
	if len(env) == 0 {
		return res, nil
	}

	data, err := yaml.Marshal(res)
	if err != nil {
		return nil, err
	}

	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, err
	}

	if err := interpolateYAMLNode(&node, env); err != nil {
		return nil, err
	}

	data, err = yaml.Marshal(&node)
	if err != nil {
		return nil, err
	}

	var result config.Resource
	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	result.SourceFile = res.SourceFile
	return &result, nil
}

func interpolateYAMLNode(node *yaml.Node, env map[string]string) error {
	if node == nil {
		return nil
	}

	if node.Kind == yaml.ScalarNode && node.Tag == "!!str" {
		interpolated, err := config.InterpolateDeclared([]byte(node.Value), env)
		if err != nil {
			return err
		}
		node.Value = string(interpolated)
	}

	for _, child := range node.Content {
		if err := interpolateYAMLNode(child, env); err != nil {
			return err
		}
	}

	return nil
}

// validateResourceTypes ensures all resources have valid types.
func validateResourceTypes(resources []*config.Resource) error {
	for _, res := range resources {
		if !resource.IsValidType(res.Type) {
			return fmt.Errorf("unknown resource type %q in %s", res.Type, res.SourceFile)
		}
	}
	return nil
}

func setPreviewRedaction(res *config.Resource) {
	if resource.Type(res.Type) == resource.TypeInstance {
		res.PreviewRedactPrefixes = []string{"config.environment."}
		return
	}
	res.PreviewRedactPrefixes = nil
}
