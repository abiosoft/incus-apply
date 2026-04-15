# Resources

Individual Incus resource definition examples. Each file demonstrates a single resource type and can be applied independently.

## Usage

```sh
incus-apply <yaml file>
```

## Examples

| File                   | Description                                                                      |
| ---------------------- | -------------------------------------------------------------------------------- |
| `instance.yaml`        | Basic Debian 13 system container with CPU and memory limits                      |
| `cloud-init.yaml`      | Alpine container with cloud-init: installs packages and writes a sentinel file   |
| `vm.yaml`              | Alpine Edge virtual machine with resource limits                                 |
| `oci.yaml`             | Alpine Linux OCI container from ghcr.io                                          |
| `network.yaml`         | Bridge network with IPv4 DHCP and NAT                                            |
| `network-forward.yaml` | Network forward mapping an external IP to internal services with port forwarding |
| `profile.yaml`         | Shared profile with CPU and memory limits for web servers                        |
| `project.yaml`         | Isolated project with selective feature flags                                    |
| `storage-pool.yaml`    | Directory-backed storage pool                                                    |
| `storage-volume.yaml`  | Custom 20 GiB persistent storage volume                                          |
