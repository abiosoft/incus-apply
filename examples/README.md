# Examples

This directory contains example configurations for `incus-apply`, organized by use case.

## Usage

Run any example with:

```sh
incus-apply <yaml file>
```

## Directories

| Directory                | Description                                                                              |
| ------------------------ | ---------------------------------------------------------------------------------------- |
| [resources/](resources/) | Individual Incus resource definitions (instances, networks, storage, profiles, and more) |
| [incus-os/](incus-os/)   | Incus OS virtual-machine with an ephemeral ISO downloader instance                       |
| [incus-vm/](incus-vm/)   | Incus installed inside a Debian VM with the Zabbly kernel and ZFS                        |
| [windows/](windows/)     | Windows 11 AMD64 virtual machine with an ephemeral ISO downloader instance               |
| [wordpress/](wordpress/) | WordPress stack in three variants: OCI containers, system container, and VM              |
