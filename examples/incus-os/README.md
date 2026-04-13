# Incus OS

Examples for installing [Incus OS](https://github.com/lxc/incus-os) using an ephemeral Alpine VM to generate a installation ISO.

## How it works

An ephemeral Alpine VM uses [`flasher-tool`](https://github.com/lxc/incus-os) to produce a customised Incus OS ISO that has seed data baked in. The seed data includes the host's client certificate so the resulting Incus OS installation automatically trusts the host. The generated ISO is written to the host's `/tmp` directory and is then used to perform the Incus OS installation.

## Prerequisites

- The Incus OS ISO path on the host (default: `/tmp/incus-iso/IncusOS.iso`); override with `ISO_FILE`
- The host's Incus client certificate is read automatically via a computed variable

## Usage

```sh
incus-apply incus-os.yaml
```

## Examples

| File            | Description                                                                                                                                           |
| --------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| `incus-os.yaml` | Spins up an ephemeral Alpine VM that generates a installation ISO with the host's client certificate baked in, ready to use for Incus OS installation |
