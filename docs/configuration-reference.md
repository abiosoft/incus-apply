# Configuration Reference

This page contains the full field reference for `incus-apply` resource documents.

## Common Fields

| Field         | Type   | Description                                    |
| ------------- | ------ | ---------------------------------------------- |
| `type`        | string | **Required.** Resource type                    |
| `name`        | string | **Required.** Resource name                    |
| `project`     | string | Incus project (overridden by `--project` flag) |
| `config`      | map    | Resource configuration options                 |
| `devices`     | map    | Device configurations                          |
| `description` | string | Resource description                           |

## Instance Fields

| Field       | Type   | Description                                                        |
| ----------- | ------ | ------------------------------------------------------------------ |
| `image`     | string | Image to use (for example `images:debian/12`)                      |
| `vm`        | bool   | Create a VM instead of a container                                 |
| `empty`     | bool   | Create an empty instance                                           |
| `ephemeral` | bool   | Create an ephemeral instance (deleted when it stops)               |
| `profiles`  | list   | Profiles to apply                                                  |
| `storage`   | string | Storage pool for the root disk                                     |
| `network`   | string | Network to attach                                                  |
| `target`    | string | Cluster member target                                              |
| `after`     | list   | Instance names (same project) that must be applied before this one |
| `setup`     | list   | Post-create and post-update actions to run inside the instance     |

### Instance Setup Actions

Use `setup` to run imperative instance actions after Incus resource changes are applied.

- `when: create` runs only when the instance is created or recreated.
- `when: update` runs on create and on later applies when the instance changes.
- `when: always` runs on every apply, even when the instance config itself is unchanged.
- `when` defaults to `create` if omitted.
- `required` defaults to `true`; set `required: false` to continue with later setup actions when one fails.
- `skip: true` keeps the action in config but prevents execution.

Supported actions:

| Field       | Type    | Description                                                                                |
| ----------- | ------- | ------------------------------------------------------------------------------------------ |
| `action`    | string  | **Required.** `exec` or `file_push`                                                        |
| `when`      | string  | `create`, `update`, or `always` (defaults to `create`)                                     |
| `required`  | boolean | Optional; defaults to `true`. Set to `false` to continue when this setup action fails      |
| `skip`      | boolean | Skip the action without removing it from config                                            |
| `script`    | string  | Required for `action: exec`; executed as root using `sh -c <script>`                       |
| `cwd`       | string  | Optional working directory for `action: exec`; passed to `incus exec --cwd`                |
| `path`      | string  | Required for `action: file_push`; absolute path inside the instance                        |
| `content`   | string  | Inline file content for `file_push`                                                        |
| `source`    | string  | Local source path for `file_push`; relative paths are resolved from the owning config file |
| `recursive` | boolean | Optional for `file_push`; passes `--recursive` to `incus file push` when `source` is used  |
| `uid`       | integer | Optional file owner uid for `file_push`                                                    |
| `gid`       | integer | Optional file owner gid for `file_push`                                                    |
| `mode`      | string  | Optional file mode for `file_push`                                                         |

Notes:

- `file_push` accepts `content`, `source`, or neither. When both are omitted, an empty file is created at `path`.
- `file_push.source` is passed to `incus file push` as-is after relative-path resolution. `incus-apply` validates that it exists when the action executes, but does not read it.
- Use `recursive: true` with `file_push.source` when pushing directories or when you want `incus file push --recursive`.
- `exec` actions run as the root user inside the instance unless Incus defaults are changed elsewhere.
- `exec` actions always run non-interactively and use `sh -c`, so multi-line shell scripts work as expected.
- `required: false` allows apply to continue after a setup failure; failed optional actions are reported as warnings.
- VM setup waits for `incus wait <instance> agent` before executing `exec` or `file_push` actions.
- Relative `source` paths are resolved from the configuration file location. Absolute paths are also supported.
- Relative `source` paths are not supported when applying config from stdin or a URL.
- Changes to `when: create` actions are treated as recreate-required for managed instances, because those actions cannot be replayed on a normal update. The resource is skipped until you rerun with `--replace`.
- For a full multi-service example, see [../examples/wordpress.yaml](../examples/wordpress.yaml), which provisions WordPress on Debian 13 with MariaDB and Caddy using setup actions.

### Example

```yaml
type: instance
name: web
image: images:debian/12
setup:
  - action: exec
    when: create
    script: apt-get update && apt-get install -y caddy
  - action: file_push
    when: update
    path: /etc/caddy/Caddyfile
    source: ./Caddyfile
  - action: exec
    when: always
    script: systemctl restart caddy
```

## Storage Pool Fields

| Field    | Type   | Description                                 |
| -------- | ------ | ------------------------------------------- |
| `driver` | string | Storage driver (dir, zfs, btrfs, lvm, ceph) |
| `source` | string | Source path or device                       |

## Storage Volume And Bucket Fields

| Field  | Type   | Description                     |
| ------ | ------ | ------------------------------- |
| `pool` | string | **Required.** Storage pool name |

## Network Fields

| Field         | Type   | Description                                          |
| ------------- | ------ | ---------------------------------------------------- |
| `networkType` | string | Network type (bridge, ovn, macvlan, sriov, physical) |

## Network Forward Fields

For `type: network-forward`, `listen_address` is the external address and `network` selects the parent network.

| Field            | Type   | Description                                                                      |
| ---------------- | ------ | -------------------------------------------------------------------------------- |
| `listen_address` | string | **Required.** External listen address                                            |
| `network`        | string | **Required.** Parent network name                                                |
| `ports`          | list   | Optional port forwarding rules in the same shape as `incus network forward edit` |

Use `config.target_address` to set the default target address for unmatched traffic.

### Example

```yaml
type: network-forward
listen_address: 198.51.100.10
network: public
description: Shared external IP for web services
config:
  target_address: 10.42.0.10
ports:
  - protocol: tcp
    listen_port: "80"
    target_address: 10.42.0.11
    target_port: "8080"
  - protocol: tcp
    listen_port: "443"
    target_address: 10.42.0.12
    target_port: "8443"
```

## Network ACL Fields

| Field     | Type | Description            |
| --------- | ---- | ---------------------- |
| `ingress` | list | Ingress firewall rules |
| `egress`  | list | Egress firewall rules  |

## Variables

Variables are declared with a `type: vars` document and referenced from resource documents with `$VAR` or `${VAR}`.

### Example

```yaml
---
type: vars
vars:
  DB_NAME: myapp
  DB_USER: appuser
  DB_PASS: ${MYSQL_PASSWORD}
files:
  - secrets.env
commands:
  DB_VERSION: "mysql --version | awk '{print $3}'"
---
type: instance
name: db
image: docker:mysql
config:
  environment.MYSQL_DATABASE: $DB_NAME
  environment.MYSQL_USER: $DB_USER
  environment.MYSQL_PASSWORD: $DB_PASS
  environment.MYSQL_VERSION: $DB_VERSION
```

### Scoping

- Variables are file-scoped by default.
- Use `global: true` in a `vars` document to share variables across files.
- File-scoped variables override global variables with the same name.

### Shell Environment

- Shell environment variables can be referenced only inside the `vars` document.
- Resource documents expand only variables declared through `type: vars`.
- If a referenced variable is not declared in `type: vars`, it is left unchanged in the resource document.

Example:

```yaml
---
type: vars
vars:
  APP_NAME: myapp
---
type: instance
name: web
image: images:debian/12
config:
  environment.APP_NAME: $APP_NAME
  environment.HOME_DIR: $HOME
```

In this example, `environment.APP_NAME` becomes `myapp`, while `environment.HOME_DIR` remains `$HOME` because `HOME` was not declared in `type: vars`.

### Commands

The `commands` key maps variable names to shell command strings. Each command is passed as a single argument to `sh -c`. The trimmed stdout of the command becomes the variable value.

```yaml
type: vars
commands:
  GIT_SHA: "git rev-parse --short HEAD"
  HOSTNAME: "hostname -f"
```

Resolution order (later sources win):
1. `files` — env files, in listed order
2. `vars` — inline values
3. `commands` — shell command output

A non-zero exit code from any command is a fatal error.

### Preview Redaction

- Instance preview output redacts `config.environment.*` values by default.
- This affects preview rendering only. Interpolation and apply behavior still use the actual values.
- Use `--show-env` when you want preview output to show the real environment variable values.

### Syntax

- `$VAR` uses the value of `VAR`.
- `${VAR}` uses the value of `VAR`.
- `${VAR:-default}` uses `default` when `VAR` is unset or empty.
- `$$` produces a literal `$`.

## Notes

- Configuration files may contain multiple YAML documents separated by `---`.
- Variables are declared with `type: vars` and are not resource documents.
- See [../README.md](../README.md) for quick start and common usage.
