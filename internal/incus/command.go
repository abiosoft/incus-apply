package incus

import (
	"fmt"

	"github.com/abiosoft/incus-apply/internal/config"
	"github.com/abiosoft/incus-apply/internal/resource"
	"gopkg.in/yaml.v3"
)

// instanceExecArgs builds the base argument slice for an `incus exec` command
// against an instance. The returned slice already includes the "--" separator;
// callers append the in-instance command directly.
//
// The instance is addressed as [remote:]name so that the command targets the
// correct remote server when one is configured.
//
//	args := c.instanceExecArgs(res, "test", "-f", "/path")
func (c client) instanceExecArgs(res *config.Resource, cmd ...string) []string {
	args := []string{"exec", res.QualifiedName(), "--disable-stdin", "--force-noninteractive"}
	args = append(args, c.globalFlags...)
	args = c.appendProjectFlag(args, res.Project)
	return append(append(args, "--"), cmd...)
}

// buildCommand constructs a standard incus CLI command.
func (c client) buildCommand(meta resource.TypeMeta, cmdParts []string, res *config.Resource, force bool) []string {
	args := make([]string, len(cmdParts))
	copy(args, cmdParts)

	if meta.PrependNetwork && res.Network != "" {
		// Network-scoped resources (e.g. network-forward) use the network name
		// as the primary scope locator. The remote (if any) travels on the network name.
		args = append(args, res.QualifiedNetwork())
	}

	// Storage resources require pool name before the resource name
	if meta.PrependPool && res.Pool != "" {
		// For pool-scoped resources (storage volumes / buckets) the remote
		// prefix travels on the pool name argument.
		args = append(args, res.QualifiedPool())
	}

	args = append(args, resourceIdentifier(res))

	if force {
		args = append(args, "--force")
	}

	args = append(args, c.globalFlags...)
	args = c.appendProjectFlag(args, res.Project)

	if res.Target != "" {
		args = append(args, "--target", res.Target)
	}

	return args
}

// buildCreateCommand constructs a create command with type-specific options.
// Returns both the command arguments and any stdin data required.
func (c client) buildCreateCommand(meta resource.TypeMeta, res *config.Resource) ([]string, []byte) {
	args := make([]string, len(meta.Create))
	copy(args, meta.Create)

	switch resource.Type(res.Type) {
	case resource.TypeInstance:
		args = c.buildInstanceCreateArgs(args, res)
	case resource.TypeStoragePool:
		args = c.buildStoragePoolCreateArgs(args, res)
	case resource.TypeStorageVolume, resource.TypeStorageBucket:
		args = c.buildStorageResourceCreateArgs(args, res)
	case resource.TypeNetworkForward:
		args = c.buildNetworkForwardCreateArgs(args, res)
	case resource.TypeNetwork:
		args = c.buildNetworkCreateArgs(args, res)
	default:
		args = append(args, resourceIdentifier(res))
	}

	args = append(args, c.globalFlags...)
	args = c.appendProjectFlag(args, res.Project)
	if res.Target != "" {
		args = append(args, "--target", res.Target)
	}

	stdin := c.buildStdinConfig(meta, res)
	return args, stdin
}

// --- Type-Specific Create Builders ---

func (c client) buildInstanceCreateArgs(args []string, res *config.Resource) []string {
	if !res.Empty && res.Image != "" {
		args = append(args, res.Image)
	}
	args = append(args, res.QualifiedName())
	if res.VM {
		args = append(args, "--vm")
	}
	if res.Empty {
		args = append(args, "--empty")
	}
	if res.Ephemeral {
		args = append(args, "--ephemeral")
	}
	if res.Storage != "" {
		args = append(args, "--storage", res.Storage)
	}
	if res.Network != "" {
		args = append(args, "--network", res.Network)
	}
	for _, profile := range res.Profiles {
		args = append(args, "--profile", profile)
	}
	return args
}

func (c client) buildStoragePoolCreateArgs(args []string, res *config.Resource) []string {
	args = append(args, res.QualifiedName())
	if res.Driver != "" {
		args = append(args, res.Driver)
	}
	if res.Source != "" {
		args = append(args, "source="+res.Source)
	}
	return args
}

func (c client) buildStorageResourceCreateArgs(args []string, res *config.Resource) []string {
	if res.Pool != "" {
		// Storage volumes and buckets are scoped under a pool. The remote (if any)
		// goes on the pool name; the volume/bucket name itself stays bare.
		args = append(args, res.QualifiedPool())
	}
	args = append(args, res.Name)
	if res.ContentType != "" {
		args = append(args, "--type", res.ContentType)
	}
	return args
}

func (c client) buildNetworkCreateArgs(args []string, res *config.Resource) []string {
	args = append(args, res.QualifiedName())
	if res.NetworkType != "" {
		args = append(args, "--type="+res.NetworkType)
	}
	return args
}

func (c client) buildNetworkForwardCreateArgs(args []string, res *config.Resource) []string {
	if res.Network != "" {
		// Network forwards are scoped under a network. The remote (if any) goes
		// on the network name; the listen address itself stays bare.
		args = append(args, res.QualifiedNetwork())
	}
	args = append(args, res.ListenAddress)
	return args
}

func resourceIdentifier(res *config.Resource) string {
	switch resource.Type(res.Type) {
	case resource.TypeNetworkForward:
		// Network-forward resources are identified by their listen address.
		// The remote is carried on the preceding network name argument, not here.
		return res.ListenAddress
	case resource.TypeStorageVolume, resource.TypeStorageBucket:
		// Pool-scoped resources are identified by bare name.
		// The remote is carried on the preceding pool name argument, not here.
		return res.Name
	default:
		// For all other resource types the remote prefix travels on the resource
		// name itself: incus accepts "[remote:]name" throughout its CLI.
		return res.QualifiedName()
	}
}

// --- Helper Methods ---

// getTypeMeta retrieves type metadata or returns an error for unknown types.
func (c client) getTypeMeta(t string) (resource.TypeMeta, error) {
	meta, ok := resource.GetTypeMeta(t)
	if !ok {
		return resource.TypeMeta{}, fmt.Errorf("unknown resource type: %s", t)
	}
	return meta, nil
}

// appendProjectFlag adds --project flag if project is specified.
func (c client) appendProjectFlag(args []string, project string) []string {
	if project != "" {
		return append(args, "--project", project)
	}
	return args
}

// buildStdinConfig builds the YAML configuration to pass via stdin for create/update.
// It uses the TypeMeta.StdinFields to decide which extra fields to include.
func (c client) buildStdinConfig(meta resource.TypeMeta, res *config.Resource) []byte {
	stdin := config.Stdin{
		Config:      res.Config,
		Devices:     res.Devices,
		Description: res.Description,
	}

	for _, field := range meta.StdinFields {
		switch field {
		case "profiles":
			if len(res.Profiles) > 0 {
				stdin.Profiles = res.Profiles
			}
		case "ingress":
			stdin.Ingress = res.Ingress
		case "egress":
			stdin.Egress = res.Egress
		case "ports":
			stdin.Ports = res.Ports
		}
	}

	data, err := yaml.Marshal(stdin)
	if err != nil {
		return nil
	}
	return data
}
