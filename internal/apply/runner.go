package apply

import (
	"strconv"

	"github.com/abiosoft/incus-apply/internal/config"
	"github.com/abiosoft/incus-apply/internal/incus"
	"github.com/abiosoft/incus-apply/internal/resource"
)

// upsertAction is the planned action for a single resource.
type upsertAction int

const (
	upsertSkip      upsertAction = iota // no changes detected
	upsertCreate                        // resource does not exist
	upsertUpdate                        // resource exists and has changes
	upsertReplace                       // resource must be deleted and recreated
	upsertSetupOnly                     // setup runs without a config diff
)

// upsertPlan captures the create/update/skip decisions made during the diff
// phase so the execution phase can act on them without recomputing.
type upsertPlan struct {
	res    *config.Resource
	action upsertAction
}

// deletePlan captures the delete/skip decisions made during the diff phase
// so the execution phase can act on them without re-checking existence.
type deletePlan struct {
	res  *config.Resource
	skip bool // true when the resource was not found during diff
}

// runner executes planned operations against the Incus client.
type runner struct {
	opts    *Options
	client  incus.Client
	result  result
	printer printer
}

func (r *runner) printSummary() {
	r.printer.Print(r.opts.Quiet, r.result)
}

// upsert handles create-or-update logic for a single resource.
// Returns an error only if FailFast is enabled and an error occurs.
func (r *runner) upsert(plan upsertPlan) error {
	switch plan.action {
	case upsertCreate:
		return r.create(plan.res, formatResourceID(plan.res))
	case upsertUpdate:
		return r.update(plan.res, formatResourceID(plan.res))
	case upsertReplace:
		return r.replace(plan.res, formatResourceID(plan.res))
	case upsertSetupOnly:
		return r.setupOnly(plan.res, formatResourceID(plan.res))
	default: // upsertSkip
		r.result.unchanged++
		return nil
	}
}

func (r *runner) replace(res *config.Resource, resourceID string) error {
	deleteResult := r.client.Delete(res)
	if deleteResult.Error != nil {
		return r.result.recordError(r.opts.FailFast, resourceID, "replace delete failed", deleteResult.Error)
	}
	createResult := r.client.Create(res)
	if createResult.Error != nil {
		return r.result.recordError(r.opts.FailFast, resourceID, "replace create failed", createResult.Error)
	}
	printColored(r.opts.Quiet, colorYellow, "! %s replaced", resourceID)
	r.result.replaced++
	return r.finishCreatedInstance(res, resourceID, upsertReplace)
}

// update applies an update for a resource that was already confirmed to have changes.
func (r *runner) update(res *config.Resource, resourceID string) error {
	wasRunning := resource.Type(res.Type) == resource.TypeInstance && r.client.Running(res)
	result := r.client.Update(res)
	if result.Error != nil {
		return r.result.recordError(r.opts.FailFast, resourceID, "update failed", result.Error)
	}
	printColored(r.opts.Quiet, colorYellow, "~ %s updated", resourceID)
	r.result.updated++
	if err := r.runSetupForExistingInstance(res, resourceID, upsertUpdate, wasRunning); err != nil {
		return err
	}
	return nil
}

func (r *runner) setupOnly(res *config.Resource, resourceID string) error {
	printColored(r.opts.Quiet, colorYellow, "~ %s setup", resourceID)
	wasRunning := resource.Type(res.Type) == resource.TypeInstance && r.client.Running(res)
	if err := r.runSetupForExistingInstance(res, resourceID, upsertSetupOnly, wasRunning); err != nil {
		return err
	}
	r.result.updated++
	return nil
}

// create handles creating a new resource.
func (r *runner) create(res *config.Resource, resourceID string) error {
	result := r.client.Create(res)
	if result.Error != nil {
		return r.result.recordError(r.opts.FailFast, resourceID, "create failed", result.Error)
	}
	printColored(r.opts.Quiet, colorGreen, "+ %s created", resourceID)
	r.result.created++
	return r.finishCreatedInstance(res, resourceID, upsertCreate)
}

func (r *runner) finishCreatedInstance(res *config.Resource, resourceID string, action upsertAction) error {
	if resource.Type(res.Type) != resource.TypeInstance {
		return nil
	}

	needsSetup := hasSetupForAction(res, action)
	if !r.opts.Launch && !needsSetup {
		return nil
	}

	startResult := r.client.Start(res)
	if startResult.Error != nil {
		return r.result.recordError(r.opts.FailFast, resourceID, "start failed", startResult.Error)
	}
	if r.opts.Launch {
		printInfo(r.opts.Quiet, "  └─ started")
	}

	if err := r.runSetupActions(res, resourceID, action); err != nil {
		if !r.opts.Launch {
			_ = r.client.Stop(res)
		}
		return err
	}

	if !r.opts.Launch {
		stopResult := r.client.Stop(res)
		if stopResult.Error != nil {
			return r.result.recordError(r.opts.FailFast, resourceID, "stop after setup failed", stopResult.Error)
		}
	}
	return nil
}

func (r *runner) runSetupForExistingInstance(res *config.Resource, resourceID string, action upsertAction, wasRunning bool) error {
	if resource.Type(res.Type) != resource.TypeInstance || !hasSetupForAction(res, action) {
		return nil
	}
	if wasRunning {
		return r.runSetupActions(res, resourceID, action)
	}

	startResult := r.client.Start(res)
	if startResult.Error != nil {
		return r.result.recordError(r.opts.FailFast, resourceID, "start for setup failed", startResult.Error)
	}
	if err := r.runSetupActions(res, resourceID, action); err != nil {
		_ = r.client.Stop(res)
		return err
	}
	stopResult := r.client.Stop(res)
	if stopResult.Error != nil {
		return r.result.recordError(r.opts.FailFast, resourceID, "stop after setup failed", stopResult.Error)
	}
	return nil
}

func (r *runner) runSetupActions(res *config.Resource, resourceID string, action upsertAction) error {
	total := 0
	for _, setup := range res.Setup {
		if shouldRunSetupAction(setup, action) {
			total++
		}
	}
	if total == 0 {
		return nil
	}
	if res.VM {
		result := r.client.WaitInstanceAgent(res)
		if result.Error != nil {
			return r.result.recordError(r.opts.FailFast, resourceID, "waiting for VM agent failed", result.Error)
		}
	}

	current := 0
	for index, setup := range res.Setup {
		if !shouldRunSetupAction(setup, action) {
			continue
		}
		current++
		result := r.client.RunSetupAction(res, setup, current, total)
		if result.Error != nil {
			if !setup.IsRequired() {
				printWarning(r.opts.Quiet, "Warning: %s setup[%d] %s failed but required=false; continuing.", resourceID, index, setup.Action)
				continue
			}
			return r.result.recordError(r.opts.FailFast, resourceID, "setup["+strconv.Itoa(index)+"] "+string(setup.Action)+" failed", result.Error)
		}
	}
	return nil
}

func shouldRunSetupAction(setup config.SetupAction, action upsertAction) bool {
	if setup.Skip {
		return false
	}

	switch action {
	case upsertCreate, upsertReplace:
		return setup.When == config.SetupWhenCreate || setup.When == config.SetupWhenUpdate || setup.When == config.SetupWhenAlways
	case upsertUpdate:
		return setup.When == config.SetupWhenUpdate || setup.When == config.SetupWhenAlways
	case upsertSetupOnly:
		return setup.When == config.SetupWhenAlways
	default:
		return false
	}
}

// delete handles deletion of a single resource based on the pre-computed plan.
func (r *runner) delete(p deletePlan) error {
	resourceID := formatResourceID(p.res)

	if p.skip {
		printColored(r.opts.Quiet, colorDim, "= %s (not found)", resourceID)
		r.result.skipped++
		return nil
	}

	result := r.client.Delete(p.res)
	if result.Error != nil {
		return r.result.recordError(r.opts.FailFast, resourceID, "delete failed", result.Error)
	}
	printColored(r.opts.Quiet, colorRed, "- %s deleted", resourceID)
	r.result.deleted++
	return nil
}
