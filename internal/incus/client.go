package incus

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/abiosoft/incus-apply/internal/config"
	"github.com/abiosoft/incus-apply/internal/resource"
)

const DefaultCommandTimeout = 5 * time.Minute

// Client is the interface for performing operations on Incus.
type Client interface {
	// Ping verifies connectivity to the Incus daemon.
	Ping() error

	// Create creates a new resource in Incus.
	Create(res *config.Resource) *Result
	// Update updates an existing resource in Incus.
	Update(res *config.Resource) *Result
	// Delete removes a resource from Incus.
	Delete(res *config.Resource) *Result

	// Exists checks whether the resource already exists in Incus.
	Exists(res *config.Resource) (bool, error)
	// CurrentConfig retrieves the current YAML configuration of a resource.
	CurrentConfig(res *config.Resource) (string, error)
	// MergedConfig returns the merged config that would be applied during an update.
	MergedConfig(res *config.Resource) (string, error)

	// Start starts an instance.
	Start(res *config.Resource) *Result
	// Stop stops a running instance.
	Stop(res *config.Resource) *Result
	// Running checks if an instance is currently running.
	Running(res *config.Resource) bool
	// WaitInstanceAgent waits for a VM instance agent to become available.
	WaitInstanceAgent(res *config.Resource) *Result

	// WaitCloudInit waits for cloud-init to finish inside the instance.
	WaitCloudInit(res *config.Resource) *Result
}

type client struct {
	globalFlags []string
	stop        bool
	verbose     bool
	timeout     time.Duration
}

// New creates a new Incus client.
func New(globalFlags []string, stop, verbose bool, timeout time.Duration) Client {
	return &client{
		globalFlags: globalFlags,
		stop:        stop,
		verbose:     verbose,
		timeout:     timeout,
	}
}

// Result represents the result of an incus command execution.
type Result struct {
	Command  string
	Stdout   string
	Stderr   string
	ExitCode int
	Error    error
}

// --- Public Operations ---

// Ping verifies connectivity to the Incus daemon by querying the server API.
func (c client) Ping() error {
	result := c.runQuiet(append(c.globalFlags, "query", "/1.0"), nil)
	if result.Error != nil {
		return fmt.Errorf("cannot connect to Incus daemon: %w", result.Error)
	}
	return nil
}

// Create creates a new resource in Incus.
func (c client) Create(res *config.Resource) *Result {
	prepared, _, err := desiredForApply(res)
	if err != nil {
		return &Result{Error: err}
	}

	meta, err := c.getTypeMeta(prepared.Type)
	if err != nil {
		return &Result{Error: err}
	}
	args, stdin := c.buildCreateCommand(meta, prepared)
	// Network forwards are created first, then updated with full YAML state.
	if resource.Type(prepared.Type) == resource.TypeNetworkForward {
		result := c.runQuiet(args, nil)
		if result.Error != nil {
			return result
		}
		// Reuse Update to apply config, ports, and managed-state markers.
		updateResult := c.Update(prepared)
		if updateResult.Error != nil {
			return &Result{Error: fmt.Errorf("updating network forward after create: %w", updateResult.Error)}
		}
		return result
	}
	if resource.Type(prepared.Type) == resource.TypeInstance {
		return c.run(args, stdin)
	}
	return c.runQuiet(args, stdin)
}

// Update updates an existing resource in Incus.
func (c client) Update(res *config.Resource) *Result {
	meta, err := c.getTypeMeta(res.Type)
	if err != nil {
		return &Result{Error: err}
	}
	currentYAML, err := c.CurrentConfig(res)
	if err != nil {
		return &Result{Error: fmt.Errorf("getting current config for merge: %w", err)}
	}
	mergedStdin, err := mergeConfigs(currentYAML, res)
	if err != nil {
		return &Result{Error: fmt.Errorf("merging configs: %w", err)}
	}
	args := c.buildCommand(meta, meta.Edit, res, false)
	if c.stop && resource.Type(res.Type) == resource.TypeInstance && c.Running(res) {
		return c.stopUpdateStartInstance(res, args, mergedStdin)
	}
	return c.runQuiet(args, mergedStdin)
}

// stopUpdateStartInstance stops a running instance, applies the update, then restarts it.
func (c client) stopUpdateStartInstance(res *config.Resource, args []string, stdin []byte) *Result {
	if stopResult := c.Stop(res); stopResult.Error != nil {
		return &Result{Error: fmt.Errorf("stopping instance for update: %w", stopResult.Error)}
	}
	result := c.runQuiet(args, stdin)
	if result.Error != nil {
		c.Start(res)
		return result
	}
	if startResult := c.Start(res); startResult.Error != nil {
		return &Result{Error: fmt.Errorf("restarting instance after update: %w", startResult.Error)}
	}
	return result
}

// Delete removes a resource from Incus.
func (c client) Delete(res *config.Resource) *Result {
	meta, err := c.getTypeMeta(res.Type)
	if err != nil {
		return &Result{Error: err}
	}
	t := resource.Type(res.Type)
	// instances and projects require --force to ascertain deletion
	force := t == resource.TypeInstance || t == resource.TypeProject
	args := c.buildCommand(meta, meta.Delete, res, force)
	var stdin []byte
	if t == resource.TypeProject {
		stdin = []byte("yes\n")
	}
	return c.runQuiet(args, stdin)
}

// Exists checks if a resource exists in Incus.
func (c client) Exists(res *config.Resource) (bool, error) {
	meta, err := c.getTypeMeta(res.Type)
	if err != nil {
		return false, err
	}
	args := c.buildCommand(meta, meta.Show, res, false)
	result := c.runQuiet(args, nil)
	if result.Error != nil {
		if result.ExitCode != 0 {
			return false, nil
		}
		return false, result.Error
	}
	return true, nil
}

// CurrentConfig retrieves the current YAML configuration of a resource.
func (c client) CurrentConfig(res *config.Resource) (string, error) {
	meta, err := c.getTypeMeta(res.Type)
	if err != nil {
		return "", err
	}
	args := c.buildCommand(meta, meta.Show, res, false)
	result := c.runQuiet(args, nil)
	if result.Error != nil {
		return "", fmt.Errorf("getting current config: %w", result.Error)
	}
	return result.Stdout, nil
}

// MergedConfig returns the merged config that would be applied during an update.
func (c client) MergedConfig(res *config.Resource) (string, error) {
	currentYAML, err := c.CurrentConfig(res)
	if err != nil {
		return "", err
	}
	merged, err := mergeConfigs(currentYAML, res)
	if err != nil {
		return "", err
	}
	return string(merged), nil
}

// Start starts an instance.
func (c client) Start(res *config.Resource) *Result {
	args := []string{"start", res.Name}
	args = append(args, c.globalFlags...)
	args = c.appendProjectFlag(args, res.Project)
	return c.runQuiet(args, nil)
}

// Stop stops an instance with --force flag.
func (c client) Stop(res *config.Resource) *Result {
	args := []string{"stop", res.Name, "--force"}
	args = append(args, c.globalFlags...)
	args = c.appendProjectFlag(args, res.Project)
	return c.runQuiet(args, nil)
}

// Running checks if an instance is currently running.
func (c client) Running(res *config.Resource) bool {
	// Use anchored regex to match the exact instance name, avoiding partial
	// matches against instances whose names share a common prefix (e.g.
	// "wordpress" matching both "wordpress" and "wordpress-db").
	args := []string{"list", fmt.Sprintf("^%s$", res.Name), "--format=csv", "-c", "s"}
	args = append(args, c.globalFlags...)
	args = c.appendProjectFlag(args, res.Project)
	result := c.runQuiet(args, nil)
	if result.Error != nil {
		return false
	}
	return strings.ToLower(strings.TrimSpace(result.Stdout)) == "running"
}

func (c client) WaitInstanceAgent(res *config.Resource) *Result {
	args := []string{"wait", res.Name, "agent", "--interval", "1"}
	if c.timeout > 0 {
		args = append(args, "--timeout", strconv.Itoa(int(math.Ceil(c.timeout.Seconds()))))
	}
	args = append(args, c.globalFlags...)
	args = c.appendProjectFlag(args, res.Project)
	return c.runWithProgress(args, nil, waitForAgentProgressLabel())
}
