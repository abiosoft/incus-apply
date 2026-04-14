package apply

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/abiosoft/incus-apply/internal/config"
	"github.com/abiosoft/incus-apply/internal/incus"
	"github.com/abiosoft/incus-apply/internal/resource"
	"github.com/abiosoft/incus-apply/internal/terminal"
)

var errCancelled = errors.New("cancelled")

// Executor orchestrates create/update/delete operations against Incus.
type Executor interface {
	// Upsert creates or updates resources based on config.
	Upsert() error
	// Delete removes resources based on config.
	Delete() error
	// Reset deletes all resources then recreates them from config.
	Reset() error
}

// Renderer renders preview output to the user.
type Renderer interface {
	// Render formats and displays the preview output.
	Render(Output) error
}

type defaultExecutor struct {
	opts     Options
	client   incus.Client
	renderer Renderer
	// interactive controls whether confirmation prompts may be shown.
	interactive bool
}

// NewExecutor creates a new Executor.
func NewExecutor(opts Options, client incus.Client, renderer Renderer) Executor {
	return &defaultExecutor{
		opts:        opts,
		client:      client,
		renderer:    renderer,
		interactive: terminal.IsTerminal(os.Stdin) && terminal.IsTerminal(os.Stdout),
	}
}

// loadAndValidate loads resources from config files, applies project override,
// and validates uniqueness. Returns nil resources (no error) when none are found.
func (a *defaultExecutor) loadAndValidate() ([]*config.Resource, error) {
	resources, err := loadResources(&a.opts)
	if err != nil {
		return nil, err
	}
	if resources == nil {
		return nil, nil
	}
	a.applyProjectOverride(resources)
	a.applyRemoteOverride(resources)
	if err := validateUniqueResources(resources); err != nil {
		return nil, err
	}
	return resources, nil
}

// Upsert creates or updates resources based on config.
func (a *defaultExecutor) Upsert() error {
	resources, err := a.loadAndValidate()
	if err != nil {
		return err
	}
	if resources == nil {
		return nil
	}
	if a.opts.Select {
		resources, err = a.doMultiSelect(resources)
		if err != nil {
			if errors.Is(err, errCancelled) {
				return nil
			}
			return err
		}
		if resources == nil {
			return nil
		}
	}
	sorted, err := resource.SortForApply(resources)
	if err != nil {
		return err
	}
	output, preview, plans := computeUpsertDiff(&a.opts, a.client, sorted)

	if err := a.renderer.Render(output); err != nil {
		return err
	}

	if a.opts.IsDiffOnly() {
		return preview.errorResult()
	}
	if preview.hasErrors() {
		printInfo(a.opts.Quiet, "Not applying changes because planning encountered errors.")
		return preview.errorResult()
	}
	if preview.created == 0 && preview.updated == 0 && preview.replaced == 0 {
		return preview.errorResult()
	}

	if err := a.confirmApply("Proceed to apply these changes"); err != nil {
		if errors.Is(err, errCancelled) {
			return nil
		}
		return err
	}

	r := &runner{opts: &a.opts, client: a.client, printer: upsertPrinter{}}
	for _, p := range plans {
		if err := r.upsert(p); err != nil {
			return err
		}
	}
	r.printSummary()
	return r.result.errorResult()
}

// Delete removes resources based on config.
func (a *defaultExecutor) Delete() error {
	resources, err := a.loadAndValidate()
	if err != nil {
		return err
	}
	if resources == nil {
		return nil
	}
	if a.opts.Select {
		resources, err = a.doMultiSelect(resources)
		if err != nil {
			if errors.Is(err, errCancelled) {
				return nil
			}
			return err
		}
		if resources == nil {
			return nil
		}
	}
	sorted := resource.SortForDelete(resources)
	output, preview, plans := computeDeleteDiff(&a.opts, a.client, sorted)

	if err := a.renderer.Render(output); err != nil {
		return err
	}

	if a.opts.IsDiffOnly() {
		return preview.errorResult()
	}
	if preview.hasErrors() {
		printInfo(a.opts.Quiet, "Not deleting resources because planning encountered errors.")
		return preview.errorResult()
	}
	if preview.deleted == 0 {
		return preview.errorResult()
	}

	if err := a.confirmApply("Proceed to delete these resources"); err != nil {
		if errors.Is(err, errCancelled) {
			return nil
		}
		return err
	}

	r := &runner{opts: &a.opts, client: a.client, printer: deletePrinter{}}
	for _, p := range plans {
		if err := r.delete(p); err != nil {
			return err
		}
	}
	r.printSummary()
	return r.result.errorResult()
}

// Reset deletes all resources then recreates them from config.
// Shows a combined diff with a single confirmation prompt.
func (a *defaultExecutor) Reset() error {
	resources, err := a.loadAndValidate()
	if err != nil {
		return err
	}
	if resources == nil {
		return nil
	}
	if a.opts.Select {
		resources, err = a.doMultiSelect(resources)
		if err != nil {
			if errors.Is(err, errCancelled) {
				return nil
			}
			return err
		}
		if resources == nil {
			return nil
		}
	}

	deleteSorted := resource.SortForDelete(resources)
	createSorted, err := resource.SortForApply(resources)
	if err != nil {
		return err
	}
	output, delPreview, delPlans, _, createPlans := computeResetDiff(&a.opts, a.client, deleteSorted, createSorted)

	if err := a.renderer.Render(output); err != nil {
		return err
	}
	if a.opts.IsDiffOnly() {
		return delPreview.errorResult()
	}
	if delPreview.hasErrors() {
		printInfo(a.opts.Quiet, "Not applying reset because planning encountered errors.")
		return delPreview.errorResult()
	}

	if err := a.confirmApply("Proceed to reset (delete then recreate) these resources"); err != nil {
		if errors.Is(err, errCancelled) {
			return nil
		}
		return err
	}

	dr := &runner{opts: &a.opts, client: a.client, printer: deletePrinter{}}
	for _, p := range delPlans {
		if err := dr.delete(p); err != nil {
			return err
		}
	}
	dr.printSummary()
	if err := dr.result.errorResult(); err != nil {
		return err
	}

	printInfo(a.opts.Quiet, "")
	cr := &runner{opts: &a.opts, client: a.client, printer: upsertPrinter{}}
	for _, p := range createPlans {
		if err := cr.upsert(p); err != nil {
			return err
		}
	}
	cr.printSummary()
	return cr.result.errorResult()
}

// applyProjectOverride sets the project on all resources if specified.
func (a defaultExecutor) applyProjectOverride(resources []*config.Resource) {
	if a.opts.Project != "" {
		for _, res := range resources {
			res.Project = a.opts.Project
		}
	}
}

// applyRemoteOverride resolves the effective Incus remote for each resource.
//
// A resource may specify its remote inline by prefixing its name with the
// remote, e.g. "name: server-a:ubuntu". When such a prefix is present the
// remote is extracted from the name and stored in res.Remote, and the
// bare name is kept in res.Name. This per-resource remote takes precedence
// over the CLI-level remote passed via the trailing "remote:" argument.
//
// If no per-resource remote is found and a CLI remote was specified, that
// remote is applied as the fallback for every resource.
func (a defaultExecutor) applyRemoteOverride(resources []*config.Resource) {
	for _, res := range resources {
		if remote, name, ok := splitRemoteName(res.Name); ok {
			// Resource-level remote: strip the prefix from the name and store it.
			res.Remote = remote
			res.Name = name
		} else if a.opts.Remote != "" {
			// CLI-level remote: apply as fallback when no resource-level remote is present.
			res.Remote = a.opts.Remote
		}
	}
}

// splitRemoteName parses a "remote:name" string into its remote and name
// components. Returns ok=false when no remote prefix is present.
func splitRemoteName(name string) (remote, resourceName string, ok bool) {
	before, after, ok := strings.Cut(name, ":")
	if !ok {
		return "", "", false
	}
	return before, after, true
}

// doMultiSelect shows an interactive multi-select UI and returns only the
// resources the user chose. Returns errCancelled when the user aborts.
func (a defaultExecutor) doMultiSelect(resources []*config.Resource) ([]*config.Resource, error) {
	labels := make([]string, len(resources))
	for i, res := range resources {
		labels[i] = formatResourceID(res)
	}

	chosen, result, err := terminal.MultiSelect("Select resources", labels)
	if err != nil {
		return nil, err
	}
	if result == terminal.MultiSelectCancelled {
		return nil, errCancelled
	}
	if result == terminal.MultiSelectAll {
		return resources, nil
	}
	if len(chosen) == 0 {
		printInfo(a.opts.Quiet, "No resources selected.")
		return nil, nil
	}

	filtered := make([]*config.Resource, len(chosen))
	for i, idx := range chosen {
		filtered[i] = resources[idx]
	}
	return filtered, nil
}

func (a defaultExecutor) confirmApply(prompt string) error {
	if a.opts.Yes {
		return nil
	}
	if !a.interactive {
		return fmt.Errorf("confirmation required for %q in non-interactive mode; rerun with --yes or --diff", prompt)
	}
	if !terminal.ConfirmPrompt(prompt) {
		printInfo(a.opts.Quiet, "Cancelled.")
		return errCancelled
	}
	return nil
}
