package apply

import (
	"sort"
	"strings"

	"github.com/abiosoft/incus-apply/internal/config"
	"github.com/abiosoft/incus-apply/internal/incus"
	"github.com/abiosoft/incus-apply/internal/resource"
)

// computeUpsertDiff computes what would change for each resource without applying.
// Returns an Output ready for rendering, the stats, and the execution plan.
func computeUpsertDiff(opts *Options, client incus.Client, resources []*config.Resource) (Output, *result, []upsertPlan) {
	preview := &result{}
	var creates, updates, replaces, unchanged []OutputItem
	var plans []upsertPlan
	appendNote := func(note, extra string) string {
		if note == "" {
			return extra
		}
		return note + ", " + extra
	}
	buildOutput := func() Output {
		var output Output
		output.FileCount = opts.FileCount
		output.ResourceCount = len(resources)
		output.AddGroup(ActionCreate, creates)
		output.AddGroup(ActionUpdate, updates)
		output.AddGroup(ActionReplace, replaces)
		output.AddGroup(ActionUnchange, unchanged)
		output.Summary = preview.upsertSummary()
		return output
	}

	for _, res := range resources {
		resourceID := formatResourceID(res)

		exists, err := client.Exists(res)
		if err != nil {
			if preview.recordError(opts.FailFast, resourceID, "checking existence", err) != nil {
				return buildOutput(), preview, plans
			}
			continue
		}

		if !exists {
			item := OutputItem{ResourceID: resourceID}
			if resource.Type(res.Type) == resource.TypeInstance && opts.Launch {
				item.Note = "launch"
			}
			if hasSetupForCreate(res) {
				item.Note = appendNote(item.Note, "setup")
			}
			creates = append(creates, item)
			preview.created++
			plans = append(plans, upsertPlan{res: res, action: upsertCreate})
			continue
		}

		current, err := client.CurrentConfig(res)
		if err != nil {
			if preview.recordError(opts.FailFast, resourceID, "getting current config", err) != nil {
				return buildOutput(), preview, plans
			}
			continue
		}

		diff, status, err := incus.DiffResource(current, res)
		if err != nil {
			if preview.recordError(opts.FailFast, resourceID, "computing diff", err) != nil {
				return buildOutput(), preview, plans
			}
			continue
		}
		redactPreviewDiff(diff, res, opts.ShowEnv)

		if len(diff) > 0 {
			item := OutputItem{ResourceID: resourceID, Changes: diff}
			if opts.Stop && resource.Type(res.Type) == resource.TypeInstance && client.Running(res) {
				item.Note = "restart"
			}
			if hasSetupForUpdate(res) {
				item.Note = appendNote(item.Note, "setup")
			}
			if !status.Managed && status.Warning != "" {
				printWarning(opts.Quiet, "Warning: %s was not created by incus-apply; falling back to live-state diff and update behavior.", resourceID)
				item.Note = appendNote(item.Note, status.Warning)
			}
			if len(status.UnsupportedChanges) > 0 {
				item.Note = appendNote(item.Note, status.Warning)
				if opts.Replace {
					replaces = append(replaces, item)
					preview.replaced++
					plans = append(plans, upsertPlan{res: res, action: upsertReplace})
					continue
				}

				// Without --replace, skip the resource and warn.
				printWarning(opts.Quiet, "Warning: %s has create-only field changes (%s); skipping (rerun with --replace to recreate).",
					resourceID, unsupportedChangePaths(status.UnsupportedChanges))
				unchanged = append(unchanged, item)
				preview.unchanged++
				plans = append(plans, upsertPlan{res: res, action: upsertSkip})
				continue
			}
			updates = append(updates, item)
			preview.updated++
			plans = append(plans, upsertPlan{res: res, action: upsertUpdate})
		} else if hasSetupForAlways(res) {
			updates = append(updates, OutputItem{ResourceID: resourceID, Note: "setup"})
			preview.updated++
			plans = append(plans, upsertPlan{res: res, action: upsertSetupOnly})
		} else {
			unchanged = append(unchanged, OutputItem{ResourceID: resourceID})
			preview.unchanged++
			plans = append(plans, upsertPlan{res: res, action: upsertSkip})
		}
	}

	return buildOutput(), preview, plans
}

func unsupportedChangePaths(changes []incus.DiffChange) string {
	seen := make(map[string]any, len(changes))
	paths := make([]string, 0, len(changes))
	for _, change := range changes {
		if _, ok := seen[change.Path]; ok {
			continue
		}
		seen[change.Path] = true
		paths = append(paths, change.Path)
	}
	sort.Strings(paths)
	return strings.Join(paths, ", ")
}

// computeDeleteDiff computes which resources exist and which don't.
// Returns an Output ready for rendering, the stats, and the execution plan.
func computeDeleteDiff(opts *Options, client incus.Client, resources []*config.Resource) (Output, *result, []deletePlan) {
	preview := &result{}
	var deletes, notFound []OutputItem
	var plans []deletePlan
	buildOutput := func() Output {
		var output Output
		output.FileCount = opts.FileCount
		output.ResourceCount = len(resources)
		output.AddGroup(ActionDelete, deletes)
		output.AddGroup(ActionNotFound, notFound)
		output.Summary = preview.deleteSummary()
		return output
	}

	for _, res := range resources {
		resourceID := formatResourceID(res)

		exists, err := client.Exists(res)
		if err != nil {
			if preview.recordError(opts.FailFast, resourceID, "checking existence", err) != nil {
				return buildOutput(), preview, plans
			}
			continue
		}

		if !exists {
			notFound = append(notFound, OutputItem{ResourceID: resourceID})
			preview.skipped++
			plans = append(plans, deletePlan{res: res, skip: true})
		} else {
			deletes = append(deletes, OutputItem{ResourceID: resourceID})
			preview.deleted++
			plans = append(plans, deletePlan{res: res, skip: false})
		}
	}

	return buildOutput(), preview, plans
}

func redactPreviewDiff(diff []incus.DiffChange, res *config.Resource, showEnv bool) {
	if len(diff) == 0 || showEnv {
		return
	}

	for i := range diff {
		if !shouldRedactPreviewPath(diff[i].Path, res.PreviewRedactPrefixes) {
			continue
		}
		diff[i].Old = "[redacted]"
		diff[i].New = "[redacted]"
	}
}

func shouldRedactPreviewPath(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func hasSetupForCreate(res *config.Resource) bool {
	return hasSetupForAction(res, upsertCreate)
}

func hasSetupForUpdate(res *config.Resource) bool {
	return hasSetupForAction(res, upsertUpdate)
}

func hasSetupForAlways(res *config.Resource) bool {
	return hasSetupForAction(res, upsertSetupOnly)
}

func hasSetupForAction(res *config.Resource, action upsertAction) bool {
	if resource.Type(res.Type) != resource.TypeInstance {
		return false
	}
	for _, setup := range res.Setup {
		if setup.Skip {
			continue
		}
		switch action {
		case upsertCreate, upsertReplace:
			if setup.When == config.SetupWhenCreate || setup.When == config.SetupWhenUpdate || setup.When == config.SetupWhenAlways {
				return true
			}
		case upsertUpdate:
			if setup.When == config.SetupWhenUpdate || setup.When == config.SetupWhenAlways {
				return true
			}
		case upsertSetupOnly:
			if setup.When == config.SetupWhenAlways {
				return true
			}
		}
	}
	return false
}
