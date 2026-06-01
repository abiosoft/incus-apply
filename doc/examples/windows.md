# Windows

This example creates a Windows 11 AMD64 VM in Incus.

An ephemeral Alpine container downloads both the Windows 11 ISO and the VirtIO drivers ISO to the host `/tmp` directory. The Windows VM is then created with both ISOs attached as CD-ROM devices.

## Prerequisites

- Enough free space in `/tmp` for the downloaded ISOs.
- TPM and UEFI support on the host, which the example config enables automatically.

## Run it

Source file: [`windows.yaml`](https://github.com/abiosoft/incus-apply/blob/main/examples/windows/windows.yaml)

```sh
incus-apply windows.yaml
```

Canonical source file:

- <https://github.com/abiosoft/incus-apply/tree/main/examples/windows>
