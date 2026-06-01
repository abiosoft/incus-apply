# Incus OS

This example installs [Incus OS](https://linuxcontainers.org/incus-os) by first generating a customized installation ISO inside an ephemeral Alpine virtual machine.

## How it works

1. An ephemeral Alpine VM runs `flasher-tool`.
2. The tool generates an Incus OS ISO with seed data baked in.
3. The seed data includes the host client certificate, so the installed system trusts the host automatically.
4. The generated ISO is written to the host `/tmp` directory and then used to launch the installation.

## Run it

Source file: [`incus-os.yaml`](https://github.com/abiosoft/incus-apply/blob/main/examples/incus-os/incus-os.yaml)

```sh
incus-apply incus-os.yaml
```

Canonical source file:

- <https://github.com/abiosoft/incus-apply/tree/main/examples/incus-os>
