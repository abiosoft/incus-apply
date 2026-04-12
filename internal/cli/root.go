package cli

import (
	"fmt"
	"os/exec"

	"github.com/abiosoft/incus-apply/internal/apply"
	"github.com/abiosoft/incus-apply/internal/config"
	"github.com/abiosoft/incus-apply/internal/incus"
	"github.com/abiosoft/incus-apply/internal/renderer"
	"github.com/spf13/cobra"
)

// NewRootCommand creates the root cobra command with all flags configured.
func NewRootCommand(version, commit, date string) *cobra.Command {
	opts := &apply.Options{}

	rootCmd := &cobra.Command{
		Use:   "incus-apply [flags] [file...]",
		Short: "Declarative configuration management for Incus",
		Long: `incus-apply is a declarative configuration management tool for Incus.

It reads .yaml or .json configuration files and creates,
updates, or deletes Incus resources accordingly.

By default, a diff is shown and you are prompted before changes are applied.

Examples:
  # Apply configs in the current directory
  incus-apply .

  # Apply from specific files or a URL
  incus-apply instance.yaml network.yaml
  incus-apply https://example.com/config.yaml

  # Show diff only (no apply)
  incus-apply . --diff

  # Auto-accept changes without prompting
  incus-apply . -y

  # Silent mode for CI (no prompt, no progress output)
  incus-apply . -yq

  # Delete resources
  incus-apply . -d -y

  # Apply to a specific project
  incus-apply . --project myproject`,
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOptions(opts); err != nil {
				return err
			}
			return checkIncusBinary()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			opts.Files = args
			return runApply(opts)
		},
		SilenceUsage: true,
	}

	rootCmd.SetVersionTemplate(fmt.Sprintf("incus-apply version %s\ngit commit: %s\nbuild date: %s\n", version, commit, date))

	// Input flags
	rootCmd.Flags().BoolVarP(&opts.Recursive, "recursive", "r", false,
		"Recursively find .yaml/.json files in directories")
	rootCmd.Flags().DurationVar(&opts.FetchTimeout, "fetch-timeout", config.DefaultFetchTimeout,
		"Timeout for fetching remote config URLs (0 disables the timeout)")

	// Operation mode flags
	rootCmd.Flags().BoolVarP(&opts.Delete, "delete", "d", false,
		"Delete resources instead of creating/updating")
	rootCmd.Flags().BoolVar(&opts.Reset, "reset", false,
		"Delete all resources then recreate them from configs")
	rootCmd.Flags().BoolVar(&opts.Select, "select", false,
		"Interactively select which resources to include before applying")
	rootCmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false,
		"Auto-accept and apply changes without prompting")
	rootCmd.Flags().StringVar(&opts.Diff, "diff", "",
		"Show preview only without applying (values: text, json)")
	rootCmd.Flags().Lookup("diff").NoOptDefVal = "text"
	rootCmd.Flags().BoolVar(&opts.Replace, "replace", false,
		"Delete and recreate managed resources when create-only fields change. Without this flag, resources with create-only field changes are skipped with a warning.")
	rootCmd.Flags().BoolVar(&opts.ShowEnv, "show-env", false,
		"Show actual environment config values in preview output instead of redacting them")
	rootCmd.Flags().BoolVar(&opts.Stop, "stop", false,
		"Force-stop running instances before applying updates")
	rootCmd.Flags().BoolVar(&opts.Launch, "launch", true,
		"Start newly created instances after creation")
	rootCmd.Flags().BoolVar(&opts.FailFast, "fail-fast", false,
		"Stop on first error instead of continuing")
	rootCmd.Flags().BoolVar(&opts.NoWaitCloudInit, "no-wait-cloud-init", false,
		"Skip waiting for cloud-init to complete after instance creation")

	// Incus global flags (passthrough to incus commands)
	rootCmd.PersistentFlags().DurationVar(&opts.CommandTimeout, "command-timeout", incus.DefaultCommandTimeout,
		"Timeout for individual incus commands (0 disables the timeout)")
	rootCmd.PersistentFlags().StringVar(&opts.Project, "project", "",
		"Incus project to use")
	rootCmd.PersistentFlags().BoolVar(&opts.ForceLocal, "force-local", false,
		"Force using local unix socket")
	rootCmd.PersistentFlags().BoolVarP(&opts.Verbose, "verbose", "v", false,
		"Show verbose output: print all setup command output and log each incus command")
	rootCmd.PersistentFlags().BoolVarP(&opts.Quiet, "quiet", "q", false,
		"Suppress progress output")

	return rootCmd
}

// checkIncusBinary verifies that the incus binary is available in PATH.
func checkIncusBinary() error {
	_, err := exec.LookPath("incus")
	if err != nil {
		return fmt.Errorf("incus binary not found in PATH: %w", err)
	}
	return nil
}

func validateOptions(opts *apply.Options) error {
	switch opts.Diff {
	case "", "text", "json":
	default:
		return fmt.Errorf("invalid --diff value %q (allowed: text, json)", opts.Diff)
	}
	if opts.Reset && opts.Delete {
		return fmt.Errorf("--reset and --delete are mutually exclusive")
	}
	if opts.Reset && opts.Diff != "" {
		return fmt.Errorf("--reset and --diff are mutually exclusive")
	}
	if opts.Select && opts.Yes {
		return fmt.Errorf("--select and --yes are mutually exclusive")
	}
	if opts.FetchTimeout < 0 {
		return fmt.Errorf("--fetch-timeout must be >= 0")
	}
	if opts.CommandTimeout < 0 {
		return fmt.Errorf("--command-timeout must be >= 0")
	}
	return nil
}

// buildGlobalFlags converts Options into incus global flags.
func buildGlobalFlags(opts *apply.Options) []string {
	var flags []string
	if opts.Verbose {
		flags = append(flags, "--verbose")
	}
	if opts.Quiet {
		flags = append(flags, "--quiet")
	}
	if opts.ForceLocal {
		flags = append(flags, "--force-local")
	}
	return flags
}

// newRenderer returns the appropriate output renderer based on options.
func newRenderer(opts *apply.Options) apply.Renderer {
	if opts.IsJSONDiff() {
		return renderer.NewJSONRenderer()
	}
	return renderer.NewTextRenderer(opts.Quiet)
}
