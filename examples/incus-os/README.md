# Incus OS

Example for installing [Incus OS](https://linuxcontainers.org/incus-os).

## Using an existing ISO

This variant expects you to provide an existing Incus OS ISO file on the host.

**File:** `incus-os.yaml`

Edit the `ISO_FILE` path in the YAML file, then run:

```sh
incus-apply incus-os.yaml
```

## Generating the ISO

This variant generates a customized installation ISO inside an ephemeral Alpine virtual machine.

**File:** `incus-os-download.yaml`

An ephemeral Alpine VM runs [`flasher-tool`](https://github.com/lxc/incus-os/tree/main/incus-osd/cmd/flasher-tool) to produce a customised Incus OS ISO with seed data baked in. The seed data includes the host's client certificate so the resulting Incus OS installation automatically trusts the host. The generated ISO is written to the host's `/tmp` directory and is then used to perform the installation.

```sh
incus-apply incus-os-download.yaml
```
