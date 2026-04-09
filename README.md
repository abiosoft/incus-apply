# incus-apply

[![CI](https://github.com/abiosoft/incus-apply/actions/workflows/go.yml/badge.svg)](https://github.com/abiosoft/incus-apply/actions/workflows/go.yml)

Declarative configuration management for [Incus](https://linuxcontainers.org/incus/).

![incus-apply demo](./demo.gif)

## Installation

Install the latest release binary:

```bash
curl -LO https://github.com/abiosoft/incus-apply/releases/latest/download/incus-apply-$(uname)-$(uname -m)
sudo install incus-apply-$(uname)-$(uname -m) /usr/local/bin/incus-apply
```

Or build from source (requires Go):

```bash
git clone https://github.com/abiosoft/incus-apply
cd incus-apply
make && sudo make install
```

## Quick Start

1. Create a config file `debian.yaml`:

```yaml
type: instance
name: debian
image: images:debian/12
profiles:
  - default
config:
  limits.cpu: "2"
  limits.memory: 1GiB
```

2. Apply it:

```bash
incus-apply debian.yaml
```

## Usage

```bash
# Apply all configs in current directory
incus-apply .

# Apply specific files
incus-apply instance.yaml network.yaml

# Apply recursively from a directory
incus-apply ./configs/ -r

# Apply from stdin
cat instance.yaml | incus-apply -

# Apply from URL
incus-apply https://example.com/instance.yaml

# Override remote fetch and incus command timeouts
incus-apply . --fetch-timeout=10s --command-timeout=2m

# Auto-accept changes without prompting
incus-apply . -y

# In non-interactive environments, use --yes to apply changes
incus-apply . -yq

# Silent mode for CI (no diff, no prompt)
incus-apply . -yq

# Show diff only (no apply)
incus-apply . --diff

# Show diff only and reveal environment values in preview
incus-apply . --diff --show-env

# Replace resources when create-only fields change
incus-apply . --replace -y

# Show diff as JSON (for tooling)
incus-apply . --diff=json

# Delete resources defined in configs
incus-apply . -d -y

# Stop running instances before applying (for config keys that require restart)
incus-apply . --stop

# Create instances without starting them
incus-apply . --launch=false

# Apply to a specific project
incus-apply . --project myproject
```

## Configuration Format

Configuration files can be any `.yaml`, `.yml`, or `.json` file. When scanning a directory, `incus-apply` reads all YAML and JSON files and processes only the documents whose `type` matches a supported incus resource type, skipping everything else.

### Basic Example

```yaml
# web-server.yaml
type: instance
name: web-server
image: images:debian/12
profiles:
  - default
config:
  limits.cpu: "2"
  limits.memory: 1GiB
devices:
  root:
    type: disk
    pool: default
    path: /
    size: 10GiB
description: Web server container
```

### Multi-Document YAML

Multiple resources can be defined in a single file using YAML document separators (`---`):

```yaml
# stack.yaml
---
type: profile
name: app-profile
config:
  limits.memory: 512MiB
---
type: instance
name: app-1
image: images:alpine/3.19
profiles:
  - default
  - app-profile
---
type: instance
name: app-2
image: images:alpine/3.19
profiles:
  - default
  - app-profile
```

## Variables

Declare variables with `type: vars` and reference them with `$VAR` or `${VAR}` in resource documents.

```yaml
---
type: vars
vars:
  NODE_ENV: production
  MYSQL_DATABASE: app
commands:
  GIT_SHA: "git rev-parse --short HEAD"
---
type: instance
name: api
image: docker:node:20
config:
  environment.NODE_ENV: $NODE_ENV
  environment.MYSQL_DATABASE: $MYSQL_DATABASE
  environment.GIT_SHA: $GIT_SHA
```

`commands` maps variable names to shell command strings passed to `sh -c`; stdout becomes the value. Resolution order: `files` → `vars` → `commands` (later wins).

For full variable usage, scoping rules, and syntax, see [docs/configuration-reference.md](./docs/configuration-reference.md).

## Supported Resource Types

| Type              | Description                           |
| ----------------- | ------------------------------------- |
| `instance`        | Containers and virtual machines       |
| `profile`         | Configuration profiles                |
| `network`         | Networks (bridge, ovn, macvlan, etc.) |
| `network-forward` | Forward external addresses and ports  |
| `network-acl`     | Network access control lists          |
| `network-zone`    | DNS zones                             |
| `storage-pool`    | Storage pools                         |
| `storage-volume`  | Custom storage volumes                |
| `storage-bucket`  | S3-compatible storage buckets         |
| `project`         | Projects for resource isolation       |
| `cluster-group`   | Cluster member groups                 |

## Resource Dependency Ordering

Resources are automatically created in dependency order:

1. Projects
2. Storage pools
3. Networks
4. Network forwards
5. Network ACLs
6. Network zones
7. Storage volumes
8. Storage buckets
9. Cluster groups
10. Profiles
11. Instances

For deletion, the order is reversed.

## Common Configuration Fields

| Field         | Type   | Description                                    |
| ------------- | ------ | ---------------------------------------------- |
| `type`        | string | **Required.** Resource type                    |
| `name`        | string | **Required.** Resource name                    |
| `project`     | string | Incus project (overridden by `--project` flag) |
| `config`      | map    | Resource configuration options                 |
| `devices`     | map    | Device configurations                          |
| `description` | string | Resource description                           |

For the full per-resource field reference, see [docs/configuration-reference.md](./docs/configuration-reference.md).

## CLI Flags

```
Usage:
  incus-apply [flags] [file...]

Arguments:
  file...   Config files, directories, URLs, or '-' for stdin

Flags:
  -r, --recursive        Recursively find .yaml/.yml/.json files in directories
  -d, --delete           Delete resources instead of creating/updating
  -y, --yes              Auto-accept and apply changes without prompting
    --diff [text|json] Show preview only without applying
      --replace          Delete and recreate managed resources when create-only fields change.
                         Without this flag, resources with create-only field changes are skipped with a warning.
      --show-env         Show actual environment config values in preview output instead of redacting them
    --fetch-timeout duration
             Timeout for fetching remote config URLs (default: 30s, 0 disables)
      --stop             Force-stop running instances before applying updates
      --launch           Start newly created instances after creation (default: true)
      --fail-fast        Stop on first error instead of continuing
  -h, --help             Help for incus-apply
      --version          Print version information

Incus Global Flags (passed through):
    --command-timeout duration
             Timeout for individual incus commands (default: 5m, 0 disables)
      --project string   Incus project to use
      --debug            Show debug output from incus
  -v, --verbose          Show verbose output from incus
  -q, --quiet            Suppress progress output
      --force-local      Force using local unix socket
```

## Examples

See the [examples](./examples/) directory for sample configurations.

## Advanced Notes

<details>
<summary>Preview, diff, and apply behavior</summary>

By default, `incus-apply` shows a preview and asks for confirmation before making changes.

- Use `--diff` to preview only.
- Use `--diff=json` for machine-readable output.
- In non-interactive environments, use `--yes` to proceed.
- If planning hits errors, the preview is still shown but apply/delete stops before making changes.
- Instance `config.environment.*` values are redacted in preview output by default.
- Use `--show-env` to reveal those values in preview output when needed.

Preview output identifies resources by effective scope:

- Project-scoped resources use `project:type/name`, for example `default:instance/web`.
- Pool-scoped storage resources use `project:type/pool/name`, for example `default:storage-volume/pool1/data`.
- Global resources omit the project prefix and use `type/name`.

</details>

<details>
<summary>Recreate-required changes</summary>

Some fields are create-only, such as an instance image, storage pool driver, or network type.

When those fields change on a managed resource, the preview is marked `recreate required` and apply stops before making changes.

Use `--replace` to delete and recreate the resource in one run.

</details>

## Schema And Editor Setup

For schema URL and editor setup, see [docs/editor-schema.md](./docs/editor-schema.md).

## FAQ

See [docs/faq.md](./docs/faq.md) for common questions and operational notes.

## License

Apache 2.0

## Sponsoring the Project

If you (or your company) are benefiting from the project and would like to support the contributors, kindly sponsor.

- [Github Sponsors](https://github.com/sponsors/abiosoft)
- [Buy me a coffee](https://www.buymeacoffee.com/abiosoft)

