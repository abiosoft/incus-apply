# Getting started

Use this section to apply your first resource and understand the minimum configuration model.

Start with the [installation](installation/) guide to install `incus-apply`.

## Quick start

### Prerequisites

- A working Incus installation.
- Access to the target local or remote Incus server through the `incus` CLI.

### Create a resource file

Create a file named `debian.yaml`:

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

### Apply it

```bash
incus-apply debian.yaml
```

`incus-apply` loads the file, computes the diff against the current Incus state, shows a preview, and asks for confirmation before applying changes.

## Configuration format

Configuration files can be `.yaml`, `.yml`, or `.json`. When you point `incus-apply` at a directory, it scans all YAML and JSON files and processes only the documents whose `kind` matches a supported Incus resource.

## Multi-document YAML

You can define multiple resources in a single file by separating documents with `---`:

```yaml
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

For more detail, continue with the [workflows](../tutorials/workflows/) section or jump to the [configuration reference](reference/configuration/).
