# Incus OS

This example launches an Incus OS virtual machine.

## Using an existing ISO

This variant expects you to provide an existing Incus OS ISO file on the host.

Source file: [`incus-os.yaml`](https://github.com/abiosoft/incus-apply/blob/main/examples/incus-os/incus-os.yaml)

Edit the `ISO_FILE` path in the YAML file, then run:

```sh
incus-apply incus-os.yaml
```

## Generating the ISO

This variant generates a customized installation ISO inside an ephemeral Alpine virtual machine.

1. An ephemeral Alpine VM runs `flasher-tool`.
2. The tool generates an Incus OS ISO with seed data baked in.
3. The seed data includes the host client certificate, so the installed system trusts the host automatically.
4. The generated ISO is written to the host `/tmp` directory and then used to launch the installation.

Source file: [`incus-os-download.yaml`](https://github.com/abiosoft/incus-apply/blob/main/examples/incus-os/incus-os-download.yaml)

```sh
incus-apply incus-os-download.yaml
```

Canonical source files:

- <https://github.com/abiosoft/incus-apply/tree/main/examples/incus-os>
