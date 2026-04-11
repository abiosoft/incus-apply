package apply

import (
	"errors"
	"fmt"
	"os"

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
