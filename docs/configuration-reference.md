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

| Field         | Type   | Description                                                        |
| ------------- | ------ | ------------------------------------------------------------------ |
| `image`       | string | Image to use (for example `images:debian/12`)                      |
| `vm`          | bool   | Create a VM instead of a container                                 |
| `empty`       | bool   | Create an empty instance                                           |
| `ephemeral`   | bool   | Create an ephemeral instance (deleted when it stops)               |
| `profiles`    | list   | Profiles to apply                                                  |
| `storage`     | string | Storage pool for the root disk                                     |
| `network`     | string | Network to attach                                                  |
| `target`      | string | Cluster member target                                              |
| `apply.after` | list   | Instance names (same project) that must be applied before this one |

### Cloud-Init

When an instance has `config."cloud-init.vendor-data"` or `config."cloud-init.user-data"`,
`incus-apply` automatically waits for cloud-init to finish after creating the instance by
running `cloud-init status --wait` inside it. For VMs, the incus agent is waited for first.

Use cloud-init's native `#cloud-config` format to install packages, write files, and run
commands at instance creation time. See the [cloud-init documentation](https://cloudinit.readthedocs.io/)
for the full set of available modules.

### Example

```yaml
type: instance
name: web
image: images:debian/12
config:
  cloud-init.user-data: |
    #cloud-config
    package_update: true
    packages:
      - caddy
    write_files:
      - path: /etc/caddy/Caddyfile
        content: |
          :80 {
            root * /var/www/html
            file_server
          }
    runcmd:
      - systemctl enable caddy
      - systemctl restart caddy
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
---
type: instance
name: db
image: docker:mysql
config:
  environment.MYSQL_DATABASE: $DB_NAME
  environment.MYSQL_USER: $DB_USER
  environment.MYSQL_PASSWORD: $DB_PASS
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

### Built-in Incus Variables

Variables with the `incus.` prefix are resolved automatically by `incus-apply` without
declaring them in `type: vars`. They provide values from the current Incus environment.

| Variable                                        | Description                                            |
| ----------------------------------------------- | ------------------------------------------------------ |
| `${incus.remote.get-client-certificate}`        | PEM client certificate from the current remote         |
| `${incus.remote.get-client-certificate-base64}` | Base64-encoded PEM client certificate (no line breaks) |

These variables are available in all resource documents and can use the `:-` default syntax:
`${incus.remote.get-client-certificate:-none}`.

Use the `-base64` variant with cloud-init's `write_files` `encoding: b64` to safely embed the PEM certificate, which is a multi-line value that would otherwise break YAML structure.

### Preview Redaction

- Instance preview output redacts `config.environment.*` values by default.
- This affects preview rendering only. Interpolation and apply behavior still use the actual values.
- Use `--show-env` when you want preview output to show the real environment variable values.

### Syntax

- `$VAR` uses the value of `VAR`.
- `${VAR}` uses the value of `VAR`.
- `${VAR:-default}` uses `default` when `VAR` is unset or empty.
- `${incus.<name>}` resolves a built-in Incus variable.
- `$$` produces a literal `$`.

## Notes

- Configuration files may contain multiple YAML documents separated by `---`.
- Variables are declared with `type: vars` and are not resource documents.
- See [../README.md](../README.md) for quick start and common usage.
