# Configuration reference

This page contains the field reference for `incus-apply` resource documents.

## Common fields

| Field | Type | Description |
| --- | --- | --- |
| `kind` | string | Required resource kind. |
| `name` | string | Required resource name. May include a remote prefix. |
| `config` | map | Resource configuration options. |
| `devices` | map | Device configurations. |
| `description` | string | Resource description. |

### Devices

Devices follow the Incus device schema (`type: disk`, `type: nic`, etc.).

For `disk` devices without a `pool` key, a `source` path containing a path separator (e.g. `./volumes/volume1`) is resolved against the directory of the config file. Plain names such as `vol1` or `agent:config` reference a storage volume or a special Incus value and are left untouched. Use a `pool` key to reference a storage volume instead.

When the resource targets a remote server, a relative `source` cannot be resolved against the local config file and is rejected with an error; use an absolute path valid on the target server.

```yaml
kind: instance
name: test3
image: images:fedora/44
profiles:
  - default
devices:
  volume1:
    type: disk
    source: ./volumes/volume1
    path: /mnt/volume1
```

## Remotes

Incus supports named remote servers. By default, `incus-apply` targets whichever remote is configured as the default in the `incus` CLI.

Apply an entire run to a remote by passing the remote name with a trailing colon as the last positional argument:

```sh
incus-apply instance.yaml server-a:
```

Override the remote per resource by prefixing the resource name:

```yaml
kind: instance
name: server-a:ubuntu
image: images:ubuntu/24.04
```

The per-resource prefix takes precedence over a CLI-level remote target.

## Project

All resources are applied to an Incus project. By default, `incus-apply` uses the project configured for the `incus` CLI. Override it for a run with:

```sh
incus-apply . --project myproject
```

## Instance fields

| Field | Type | Description |
| --- | --- | --- |
| `image` | string | Image to use, for example `images:debian/12`. |
| `vm` | bool | Create a VM instead of a container. |
| `empty` | bool | Create an empty instance. |
| `ephemeral` | bool | Create an ephemeral instance deleted when it stops. |
| `profiles` | list | Profiles to apply. |
| `storage` | string | Storage pool for the root disk. |
| `network` | string | Network to attach. |
| `target` | string | Cluster member target. |
| `apply.after` | list | Same-project instance names that must be applied first. |

### Cloud-init

When an instance defines `config."cloud-init.vendor-data"` or `config."cloud-init.user-data"`, `incus-apply` waits for cloud-init to finish after creating the instance. For VMs, it first waits for the Incus agent.

Cloud-init values can be written either as a block scalar string or as an inline YAML mapping.

#### Block scalar form

```yaml
config:
  cloud-init.user-data: |
    #cloud-config
    packages:
      - caddy
```

#### Inline YAML mapping form

```yaml
config:
  cloud-init.user-data:
    #cloud-config
    packages:
      - caddy
    runcmd:
      - systemctl enable caddy
```

#### Complete instance example

```yaml
kind: instance
name: web
image: images:debian/12
config:
  cloud-init.user-data:
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

## Storage pool fields

| Field | Type | Description |
| --- | --- | --- |
| `driver` | string | Storage driver such as `dir`, `zfs`, `btrfs`, `lvm`, or `ceph`. |
| `source` | string | Source path or device. |

## Storage volume and bucket fields

| Field | Type | Description |
| --- | --- | --- |
| `pool` | string | Required storage pool name. |
| `type` | string | Storage content type passed as `--type`, such as `block` or `filesystem`. |

## Storage bucket key fields

Storage bucket keys are S3 credentials that grant access to a storage bucket.

| Field | Type | Description |
| --- | --- | --- |
| `pool` | string | Required storage pool name. |
| `bucket` | string | Required parent bucket name. |
| `role` | string | `admin` for full access or `read-only` for list and get access. |

```yaml
---
kind: storage-bucket
name: assets
pool: default
config:
  size: 10GiB
description: S3-compatible object storage bucket
---
kind: storage-bucket-key
name: app-key
bucket: assets
pool: default
role: read-only
description: Read-only S3 credentials for the application
```

To use storage buckets on local storage pools such as `dir`, `btrfs`, `lvm`, or `zfs`, configure an S3 listen address first:

```sh
incus config set core.storage_buckets_address :8555
```

## Network fields

| Field | Type | Description |
| --- | --- | --- |
| `networkType` | string | Network type, such as `bridge`, `ovn`, `macvlan`, `sriov`, or `physical`. |

## Network forward fields

For `kind: network-forward`, `listen_address` is the external address and `network` selects the parent network.

| Field | Type | Description |
| --- | --- | --- |
| `listen_address` | string | Required external listen address. |
| `network` | string | Required parent network name. |
| `ports` | list | Optional port forwarding rules in the same shape as `incus network forward edit`. |

Use `config.target_address` to define the default target for unmatched traffic.

```yaml
kind: network-forward
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

## Network ACL fields

| Field | Type | Description |
| --- | --- | --- |
| `ingress` | list | Ingress firewall rules. |
| `egress` | list | Egress firewall rules. |

## Variables

Variables are declared with a `kind: vars` document and referenced from resource documents with `$VAR` or `${VAR}`.

```yaml
---
kind: vars
vars:
  DB_NAME: myapp
  DB_USER: appuser
  DB_PASS: ${MYSQL_PASSWORD}
---
kind: instance
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

### Shell environment

- Shell environment variables can be referenced only inside the `vars` document.
- Resource documents expand only variables declared through `kind: vars`.
