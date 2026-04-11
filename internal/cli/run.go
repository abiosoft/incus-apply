package cli

import (
	"github.com/abiosoft/incus-apply/internal/apply"
	"github.com/abiosoft/incus-apply/internal/incus"
)

// runApply is the entry point for the apply command.
// It wires CLI options into an Executor and delegates all orchestration to it.
func runApply(opts *apply.Options) error {
	if opts.IsJSONDiff() {
		opts.Quiet = true
	}

	globalFlags := buildGlobalFlags(opts)
	client := incus.New(globalFlags, opts.Stop, opts.Debug, opts.CommandTimeout)

	if err := client.Ping(); err != nil {
		return err
	}

	executor := apply.NewExecutor(*opts, client, newRenderer(opts))

	if opts.Delete {
		return executor.Delete()
	}
	if opts.Reset {
		return executor.Reset()
	}
	return executor.Upsert()
}
