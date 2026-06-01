# Nested Incus VM

These examples provision a virtual machine with Incus already installed so the host can manage a nested Incus environment immediately after provisioning.

In both variants, the host client certificate is trusted automatically.

## Ubuntu variant

[`incus-vm.yaml`](https://github.com/abiosoft/incus-apply/blob/main/examples/incus-vm/incus-vm.yaml) creates an Ubuntu 24.04 VM with:

- Incus installed from the Zabbly repository
- ZFS configured as the default storage pool
- Host trust bootstrapped automatically

```sh
incus-apply incus-vm.yaml
```

## Debian and Zabbly kernel variant

[`incus-zabbly.yaml`](https://github.com/abiosoft/incus-apply/blob/main/examples/incus-vm/incus-zabbly.yaml) creates a Debian 13 VM with:

- The Zabbly kernel installed by `zabbly-kernel.sh`
- Incus installed by `zabbly-incus.sh`
- ZFS configured as the default storage pool

```sh
incus-apply incus-zabbly.yaml
```

Canonical source files:

- <https://github.com/abiosoft/incus-apply/tree/main/examples/incus-vm>
