# Windows

Example for creating a Windows 11 x86_64 virtual machine in Incus.

## Using existing ISOs

This variant expects you to provide existing Windows 11 and VirtIO ISOs on the host.

**File:** `windows.yaml`

Edit the `WINDOWS_ISO` and `VIRTIO_ISO` paths in the YAML file, then run:

```sh
incus-apply windows.yaml
```

## Downloading ISOs

This variant automatically downloads both the Windows 11 ISO and the VirtIO drivers ISO to the host `/tmp` directory. The Windows VM is then created with both ISOs attached as CD-ROM devices.

**File:** `windows-download.yaml`

An ephemeral Alpine container downloads the ISOs and creates a Windows 11 x86_64 VM with both ISOs attached, ready for a standard Windows installation.

**Prerequisites:**
- Sufficient disk space in `/tmp` for the ISOs (~6 GB for Windows + ~600 MB for VirtIO)

```sh
incus-apply windows-download.yaml
```
