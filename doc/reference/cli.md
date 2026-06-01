# CLI reference

## Usage

```text
Usage:
  incus-apply [flags] [file...] [remote:]
```

## Flags

| Flag | Description |
| --- | --- |
| `--command-timeout` | Timeout for individual `incus` commands. `0` disables the timeout. |
| `--delete`, `-d` | Delete resources instead of creating or updating them. |
| `--diff[=text\|json]` | Show a preview only, without applying changes. |
| `--fail-fast` | Stop on the first error instead of continuing. |
| `--fetch-timeout` | Timeout for fetching remote config URLs. `0` disables the timeout. |
| `--force-local` | Force the local Unix socket instead of using a remote target. |
| `--help`, `-h` | Show command help. |
| `--launch` | Start newly created instances after creation. |
| `--no-wait-cloud-init` | Skip waiting for cloud-init after instance creation. |
| `--project` | Select the Incus project for the run. |
| `--quiet`, `-q` | Suppress progress output. |
| `--recursive`, `-r` | Recursively search directories for YAML and JSON files. |
| `--replace` | Delete and recreate managed resources when create-only fields change. |
| `--reset` | Delete all resources in the config set and then recreate them. |
| `--select` | Interactively select resources before diff and apply. |
| `--show-env` | Show actual environment config values in preview output instead of redacting them. |
| `--stop` | Stop running instances before applying updates that require it. |
| `--verbose`, `-v` | Print setup output and every executed `incus` command. |
| `--version` | Print the version. |
| `--yes`, `-y` | Apply changes without prompting for confirmation. |

## Notes

- `--select` is mutually exclusive with `--yes` because selection happens before confirmation.
- `--force-local` cannot be combined with a remote target.
- `--reset` is mutually exclusive with `--delete` and `--diff`.
