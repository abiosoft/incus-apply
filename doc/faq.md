# FAQ

Frequently asked questions.

## What is incus-apply for?

`incus-apply` lets you declare Incus resources in files and apply them repeatedly. It is useful when you want repeatable infrastructure, reviewable configuration files, and a safer apply flow than typing `incus` commands manually.

## What resources can it manage?

It supports instances, profiles, networks, network ACLs, network zones, storage pools, storage volumes, storage buckets, storage bucket keys, projects, and cluster groups.

## Why use incus-apply instead of plain incus commands?

Use `incus-apply` when you want your Incus setup to be described in files instead of being driven entirely by manual CLI commands. That is especially useful when you want to keep resource definitions in version control, review changes before they are applied, rerun the same configuration reliably, and share a reproducible setup.

## What happens when I run it?

It loads your config files, computes what would change, shows a preview, and asks for confirmation before applying unless you use `--yes`.

## Can I preview without changing anything?

Yes. Use `--diff` for text output or `--diff=json` for machine-readable output.

For instances, `config.environment.*` values are redacted in preview output by default. Use `--show-env` if you need the preview to show the actual values.

## Can I use it in CI or scripts?

Yes. Use `--yes` to skip confirmation and combine it with `--quiet` or `--diff=json` depending on how much output you want.

## What if a resource already exists?

If the resource already exists, `incus-apply` compares the current state with your config and updates it when needed.

## What does unmanaged mean?

It means the resource exists in Incus but does not have valid `incus-apply` tracking state yet. In that case, `incus-apply` warns and falls back to live-state diff and update behavior.

## What if a change requires recreating the resource?

Some fields are create-only and cannot be changed on an existing resource. When such a change is detected, `incus-apply` prints a warning and silently ignores the create-only fields, applying any other changes on the resource normally.

To apply the create-only change itself, rerun with `--replace`. This deletes and recreates the resource in one run.

## What does `--stop` do?

Some configuration updates can only be applied while an instance is stopped. With `--stop`, a running instance is stopped before the update, updated, and then started again afterwards.

## What does `--reset` do?

`--reset` deletes every resource described in your config files and then recreates them all from scratch.

## Can I apply only a subset of resources?

Yes. Use `--select` to open an interactive multi-select dialog before the diff is computed.

## How do variables work?

Declare variables in a `kind: vars` document and reference them in resource documents with `$VAR` or `${VAR}`. See the [configuration reference](reference/configuration/) for syntax and scoping rules.

## Why are some preview values shown as `[redacted]`?

Instance `config.environment.*` values are hidden in preview output by default so secrets do not leak in diffs. Use `--show-env` to reveal them.

## How are resources named in preview output?

Preview output uses the resource's effective scope in the identifier.

- Pool-scoped storage resources use `project:type/pool/name`.
- Global resources use `type/name`.
- Most other resources also use `type/name`.

## Can I define multiple resources in one file?

Yes. Use multi-document YAML with `---` separators.

## Can I load configs from a directory, stdin, or a URL?

Yes. `incus-apply` supports local files, recursive directory discovery, `-` for stdin, and remote URLs.

## Can I control the order instances are applied?

Yes. Use `apply.after` to list instance names that should be applied before the current one. Cyclic dependencies are detected and cause an error.

## What happens if planning finds errors?

The preview is still shown, but apply or delete stops before making changes.

## Where can I find example configs?\n\nSee the [examples section](examples/index) for guided examples and the repository `examples/` directory for the source files.

## Can I apply configs to a remote Incus server?

Yes. Apply everything to a remote by appending the remote name with a trailing colon as the last positional argument, or override the remote per resource with a prefixed resource name.

## Is there editor schema support?

Yes. A generated schema file is available for editor validation and autocomplete. See the [editor schema page](reference/editor-schema/).
