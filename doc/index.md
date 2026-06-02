# incus-apply

```{toctree}
:hidden:
:maxdepth: 2

incus-apply <self>
getting-started
tutorials/index
reference
faq
support/index
```

<div class="hero">
<p><strong>Declarative configuration management for <a href="https://linuxcontainers.org/incus">Incus</a>.</strong><p>
</div>

`incus-apply` lets you manage Incus resources from files instead of long sequences of imperative CLI commands. It is designed for repeatable environments, reviewable configuration, and safer change application through a diff-first workflow.

## Typical workflow

1. Write YAML or JSON documents that declare the desired Incus resources.
2. Run `incus-apply` against a file, a directory, standard input, or a URL.
3. Review the diff preview.
4. Confirm the changes, or automate the run with `--yes`.

## Supported resources

Supported resource kinds include:

- `instance`
- `profile`
- `network`
- `network-forward`
- `network-acl`
- `network-zone`
- `storage-pool`
- `storage-volume`
- `storage-bucket`
- `storage-bucket-key`
- `project`
- `cluster-group`

## Next steps

- Start with the [getting started](getting-started/) guide.
- Use the [workflows](tutorials/workflows/) section for common command patterns.
- Jump to the [configuration reference](reference/configuration/) for full field details.
