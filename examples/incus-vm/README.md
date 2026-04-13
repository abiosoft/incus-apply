# Incus VM

Examples for running Incus inside a virtual machine. Two variants are provided depending on your preferred distro and kernel setup.

In both cases the host's client certificate is trusted automatically, so the nested Incus instance can be managed from the host immediately after provisioning.

## Usage

```sh
incus-apply <yaml file>
```

## Examples

### Ubuntu (`incus-vm.yaml`)

Creates an Ubuntu 24.04 VM with Incus installed from the Zabbly repository and ZFS as the default storage pool.

- Ubuntu 24.04 cloud VM
- Incus installed via `zabbly-incus.sh`
- ZFS storage pool configured
- Host client certificate trusted automatically

```sh
incus-apply incus-vm.yaml
```

### Debian + Zabbly Kernel (`incus-zabbly.yaml`)

Creates a Debian 13 VM with the [Zabbly](https://github.com/zabbly/linux) kernel and Incus installed, with ZFS as the default storage pool.

- Debian 13 cloud VM
- Zabbly kernel installed via `zabbly-kernel.sh`
- Incus installed via `zabbly-incus.sh`
- ZFS storage pool configured
- Host client certificate trusted automatically

```sh
incus-apply incus-zabbly.yaml
```

## Files

| File                | Description                                                                              |
| ------------------- | ---------------------------------------------------------------------------------------- |
| `incus-vm.yaml`     | Ubuntu 24.04 VM with Incus and ZFS storage                                               |
| `incus-zabbly.yaml` | Debian 13 VM with Incus, Zabbly kernel, and ZFS storage                                  |
| `zabbly-kernel.sh`  | Script to install the Zabbly kernel (embedded into the VM via cloud-init)                |
| `zabbly-incus.sh`   | Script to install Incus from the Zabbly repository (embedded into the VM via cloud-init) |
