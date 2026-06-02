# Windows

This example creates a Windows 11 x86_64 VM in Incus.


## Using existing ISOs

This variant expects you to provide existing Windows 11 and VirtIO ISOs on the host.

Source file: [`windows.yaml`](https://github.com/abiosoft/incus-apply/blob/main/examples/windows/windows.yaml)

Edit the `WINDOWS_ISO` and `VIRTIO_ISO` paths in the YAML file, then run:

```sh
incus-apply windows.yaml
```

## Downloading ISOs

This variant automatically downloads both the Windows 11 ISO and the VirtIO drivers ISO to the host `/tmp` directory. The Windows VM is then created with both ISOs attached as CD-ROM devices.

Source file: [`windows-download.yaml`](https://github.com/abiosoft/incus-apply/blob/main/examples/windows/windows-download.yaml)

```sh
incus-apply windows-download.yaml
```

Canonical source files:

- <https://github.com/abiosoft/incus-apply/tree/main/examples/windows>
