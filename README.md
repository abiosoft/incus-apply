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
kind: instance
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

```
Usage:
  incus-apply [flags] [file...] [remote:]

Flags:
      --command-timeout duration   Timeout for individual incus commands (0 disables the timeout) (default 5m0s)
  -d, --delete                     Delete resources instead of creating/updating
      --diff string[="text"]       Show preview only without applying (values: text, json)
      --fail-fast                  Stop on first error instead of continuing
      --fetch-timeout duration     Timeout for fetching remote config URLs (0 disables the timeout) (default 30s)
      --force-local                Force using local unix socket
  -h, --help                       help for incus-apply
      --launch                     Start newly created instances after creation (default true)
      --no-wait-cloud-init         Skip waiting for cloud-init to complete after instance creation
      --project string             Incus project to use
  -q, --quiet                      Suppress progress output
  -r, --recursive                  Recursively find .yaml/.json files in directories
      --replace                    Delete and recreate managed resources when create-only fields change. Without this flag, resources with create-only field changes are skipped with a warning.
      --reset                      Delete all resources then recreate them from configs
      --select                     Interactively select which resources to include before applying
      --show-env                   Show actual environment config values in preview output instead of redacting them
      --stop                       Force-stop running instances before applying updates
  -v, --verbose                    Show verbose output: print all setup command output and log each incus command
      --version                    version for incus-apply
  -y, --yes                        Auto-accept and apply changes without prompting
```

## Configuration Format

Configuration files can be any `.yaml`, `.yml`, or `.json` file. When scanning a directory, `incus-apply` reads all YAML and JSON files and processes only the documents whose `kind` matches a supported incus resource type, skipping everything else.

### Basic Example

```yaml
# web-server.yaml
kind: instance
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
kind: profile
name: app-profile
config:
  limits.memory: 512MiB
---
kind: instance
name: app-1
image: images:alpine/3.19
profiles:
  - default
  - app-profile
---
kind: instance
name: app-2
image: images:alpine/3.19
profiles:
  - default
  - app-profile
```

## Variables

Declare variables with `kind: vars` and reference them with `$VAR` or `${VAR}` in resource documents.

```yaml
---
kind: vars
vars:
  NODE_ENV: production
  MYSQL_DATABASE: app
---
kind: instance
name: api
image: docker:node:20
config:
  environment.NODE_ENV: $NODE_ENV
  environment.MYSQL_DATABASE: $MYSQL_DATABASE
```

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


## Examples

The [examples](./examples/) directory contains ready-to-run configurations covering a range of real-world use cases:

- **[resources](./examples/resources/)** — Individual resource definitions to get started quickly: containers, VMs, OCI instances, networks, storage, profiles, projects, and more.
- **[incus-vm](./examples/incus-vm/)** — Spin up a nested Incus environment inside a VM. Choose between an Ubuntu 24.04 variant or a Debian 13 VM with the Zabbly kernel — both use ZFS storage and are trusted by the host automatically.
- **[wordpress](./examples/wordpress/)** — Deploy a full WordPress stack three ways: OCI application containers, a Debian system container, or a virtual machine — all provisioned via cloud-init in a single `incus-apply` run.
- **[incus-os](./examples/incus-os/)** — Download Incus OS iso and create an [Incus OS](https://linuxcontainers.org/incus-os) installation.
- **[windows](./examples/windows/)** — Download Windows 11 iso and create a Windows 11 AMD64 VM installation.

## CLI Examples

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

# Show diff only (no apply)
incus-apply . --diff

# Auto-accept changes without prompting
incus-apply . -y

# Silent mode for CI (no prompt, no progress output)
incus-apply . -yq

# Delete resources defined in configs
incus-apply . -d -y

# Apply to a specific project
incus-apply . --project myproject

# Apply to a specific remote server (append remote name with trailing colon)
incus-apply instance.yaml server-a:

# Apply to a specific project on a remote server
incus-apply instance.yaml --project myproject server-a:
```



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

