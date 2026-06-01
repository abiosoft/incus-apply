# Workflows

## Common inputs

`incus-apply` accepts local files, directories, standard input, and remote URLs.

```bash
# Apply all configs in the current directory
incus-apply .

# Apply specific files
incus-apply instance.yaml network.yaml

# Apply recursively from a directory
incus-apply ./configs/ -r

# Apply from stdin
cat instance.yaml | incus-apply -

# Apply from URL
incus-apply https://example.com/instance.yaml
```

## Preview and confirm changes

By default, `incus-apply` previews the changes and asks for confirmation.

```bash
# Show diff only without applying
incus-apply . --diff

# Show machine-readable diff output
incus-apply . --diff=json
```

For instance resources, `config.environment.*` values are redacted in preview output by default. Use `--show-env` when you need to inspect the actual values in the preview.

If planning detects errors, the preview is still shown, but apply or delete stops before making changes.

Preview output identifies resources by effective scope:

- Most resources use `type/name`, for example `instance/web`.
- Pool-scoped storage resources use `project:type/pool/name`, for example `default:storage-volume/pool1/data`.
- Global resources omit the project prefix and use `type/name`.

## Non-interactive runs

Use these options in automation or CI:

```bash
# Auto-accept changes
incus-apply . -y

# Non-interactive with quiet output
incus-apply . -yq
```

## Remote and project targeting

Apply everything to a specific project:

```bash
incus-apply . --project myproject
```

Apply everything to a remote server by appending the remote name with a trailing colon:

```bash
incus-apply instance.yaml server-a:
```

For per-resource remote overrides, prefix the resource `name`:

```yaml
kind: instance
name: server-a:ubuntu
image: images:ubuntu/24.04
```

## Delete, replace, and reset

```bash
# Delete resources described in the configuration
incus-apply . -d -y

# Recreate resources when create-only fields change
incus-apply . --replace

# Delete then recreate everything from scratch
incus-apply . --reset
```

`--reset` is mutually exclusive with `--delete` and `--diff`.

Some fields are create-only, such as an instance image, storage pool driver, or network type. When one of those fields changes on a managed resource, `incus-apply` prints a warning, silently ignores the create-only fields, and applies any other changes on the resource normally. Use `--replace` to delete and recreate the resource so the create-only change is also applied.

## Interactive resource selection

Use `--select` to choose a subset of resources before the diff is computed. Navigate with `↑` and `↓` or `j` and `k`, toggle items with `space`, toggle all with `a`, confirm with `enter`, and cancel with `q`.

## Dependency ordering

Resources are created in dependency order. The default order is:

1. Projects
2. Storage pools
3. Networks
4. Network forwards
5. Network ACLs
6. Network zones
7. Storage volumes
8. Storage buckets
9. Storage bucket keys
10. Cluster groups
11. Profiles
12. Instances

For deletion, the order is reversed.
