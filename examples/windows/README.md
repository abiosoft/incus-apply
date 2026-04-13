# Windows

Example for creating a Windows 11 AMD64 virtual machine in Incus.

An ephemeral Alpine container is used to download the Windows 11 ISO and the VirtIO drivers ISO to the host's `/tmp` directory. The Windows VM is then created with both ISOs attached as CD-ROM devices, ready for a standard Windows installation.

## Prerequisites

- Sufficient disk space in `/tmp` for the ISOs (~6 GB for Windows + ~600 MB for VirtIO)
- A TPM device and UEFI firmware configured for the VM (handled automatically)

## Usage

```sh
incus-apply windows.yaml
```

## Examples

| File           | Description                                                                                                                 |
| -------------- | --------------------------------------------------------------------------------------------------------------------------- |
| `windows.yaml` | Downloads Windows 11 and VirtIO ISOs via an ephemeral Alpine VM, then creates a Windows 11 AMD64 VM with both ISOs attached |
