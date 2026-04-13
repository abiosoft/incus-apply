# Incus VM

Example for running Incus inside a Debian virtual machine, using the [Zabbly](https://github.com/zabbly/linux) kernel and ZFS as the storage backend.

The host's client certificate is trusted automatically so the nested Incus instance can be managed from the host immediately after provisioning.

## How it works

- A Debian 13 cloud VM is created with 4 CPUs and 4 GiB of memory.
- cloud-init installs the Zabbly kernel (`zabbly-kernel.sh`) and Zabbly Incus (`zabbly-incus.sh`)
- ZFS is configured as the default storage pool
- The host's client certificate (resolved via `incus remote get-client-certificate`) is added as a trusted client

The two shell scripts (`zabbly-kernel.sh` and `zabbly-incus.sh`) are embedded into the VM via cloud-init `write_files` and executed during first boot.

## Usage

```sh
incus-apply incus-vm.yaml
```

## Examples

| File               | Description                                                                              |
| ------------------ | ---------------------------------------------------------------------------------------- |
| `incus-vm.yaml`    | Debian 13 VM with Incus installed, Zabbly kernel, and ZFS storage                        |
| `zabbly-kernel.sh` | Script to install the Zabbly kernel (embedded into the VM via cloud-init)                |
| `zabbly-incus.sh`  | Script to install Incus from the Zabbly repository (embedded into the VM via cloud-init) |
