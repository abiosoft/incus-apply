# FAQ

## What is incus-apply for?

`incus-apply` lets you declare Incus resources in files and apply them repeatedly. It is useful when you want repeatable infrastructure, reviewable config files, and a safer apply flow than typing `incus` commands manually.

## What resources can it manage?

It supports instances, profiles, networks, network ACLs, network zones, storage pools, storage volumes, storage buckets, projects, and cluster groups.

## Why use incus-apply instead of plain incus commands?

Use `incus-apply` when you want your Incus setup to be described in files instead of being driven entirely by manual CLI commands.

That is especially useful when you want to:

- keep resource definitions in version control
- review changes before they are applied
- rerun the same configuration reliably
- share a reproducible setup with other people or environments

Plain `incus` commands are still useful for ad hoc work. `incus-apply` is better when you want repeatability, review, and a file-based workflow.

## What happens when I run it?

It loads your config files, computes what would change, shows a preview, and asks for confirmation before applying unless you use `--yes`.

## Can I preview without changing anything?

Yes. Use `--diff` for text output or `--diff=json` for machine-readable output.

For instances, `config.environment.*` values are redacted in preview output by default. Use `--show-env` if you need the preview to show the actual values.

## Can I use it in CI or scripts?

Yes. Use `--yes` to skip confirmation, and combine it with `--quiet` or `--diff=json` depending on how much output you want.

## What if a resource already exists?

If the resource already exists, `incus-apply` compares the current state with your config and updates it when needed.

## What does unmanaged mean?

It means the resource exists in Incus but does not have valid `incus-apply` tracking state yet. In that case, `incus-apply` warns and falls back to live-state diff/update behavior.

## What if a change requires recreating the resource?

Some fields are create-only and cannot be changed on an existing resource. When such a change is detected, the resource is shown in the diff with a `recreate required` note and then **skipped** — the rest of the apply continues normally.

To apply the change, rerun with `--replace`. This deletes and recreates the resource in one run.

## What does --stop do?

Some configuration updates can only be applied while an instance is stopped.

When you run `incus-apply` with `--stop`, a running instance is stopped before the update, updated, and then started again afterwards.

If that restart path will be used, the preview shows a `restart` note in the diff output.

## What does --reset do?

`--reset` deletes every resource described in your config files and then recreates them all from scratch.

It computes a combined diff showing what will be deleted and what will be recreated, then asks for confirmation once before executing both phases. Delete and create execution summaries are printed separately after each phase.

`--reset` is useful when you want a clean slate — for example to test a full provisioning run from zero, or to recover from state drift that simpler updates cannot fix.

It is mutually exclusive with `--delete` and `--diff`. Use `--yes` to skip the confirmation prompt.

## How do variables work?

Declare variables in a `type: vars` document and reference them in resource documents with `$VAR` or `${VAR}`. See [configuration-reference.md](./configuration-reference.md) for syntax and scoping rules.

## Why are some preview values shown as [redacted]?

Instance `config.environment.*` values are hidden in preview output by default so secrets do not leak in diffs. This only affects preview rendering. Use `--show-env` to reveal them.

## How are resources named in preview output?

Preview output uses the resource's effective scope in the identifier.

- Project-scoped resources use `project:type/name`.
- Pool-scoped storage resources use `project:type/pool/name`.
- Global resources use `type/name`.

Examples: `default:instance/web`, `default:storage-volume/pool1/data`, and `storage-pool/fast`.

## Can I define multiple resources in one file?

Yes. Use multi-document YAML with `---` separators.

## Can I load configs from a directory, stdin, or a URL?

Yes. `incus-apply` supports local files, recursive directory discovery, `-` for stdin, and remote URLs.

## Can I control the order instances are applied?

Yes. Use `after` to list instance names that should be applied before the current one:

```yaml
type: instance
name: app
after:
  - database
```

The `after` field is scoped to the same project. Cyclic dependencies are detected and cause an error.

## Where can I find example configs?

See [../examples/](../examples/) for sample configurations.

## Is there editor schema support?

Yes. A generated schema file is available for editor validation and autocomplete. See [editor-schema.md](./editor-schema.md).
