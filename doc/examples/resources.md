# Resources

The `examples/resources` directory contains standalone resource definitions you can apply individually or combine into larger stacks.

## Virtual machines

Alpine virtual machine:

```yaml
# Alpine virtual machine
kind: instance
name: gateway
image: images:alpine/edge
vm: true
config:
  limits.cpu: "1"
  limits.memory: 512MiB
  security.secureboot: "false"
description: Alpine virtual machine
```

## System containers

Basic Debian container:

```yaml
# Basic Debian container
kind: instance
name: app
image: images:debian/13
config:
  limits.cpu: "2"
  limits.memory: 1GiB
description: Application backend container
```

## OCI containers

The OCI registry must be added as a remote as a prior step.

For docker:

```bash
incus remote add docker https://docker.io --protocol=oci
```

Alpine Linux as an OCI container from ghcr.io:

```yaml
# Alpine Linux as an OCI container
kind: instance
name: alpine-oci
image: docker:alpine
description: Alpine Linux OCI container
```

## Profiles

Shared profile with resource limits:

```yaml
# Shared profile with resource limits
kind: profile
name: web
config:
  limits.cpu: "2"
  limits.memory: 512MiB
description: Web server resource limits profile
```

## Projects

Project for isolating example resources:

```yaml
# Project for isolating example resources
kind: project
name: example
config:
  features.images: "true"
  features.networks: "false"
  features.profiles: "true"
  features.storage.volumes: "true"
description: Example project
```

## Networks

Bridge network:

```yaml
# Bridge network
kind: network
name: webnet
networkType: bridge
config:
  ipv4.address: 10.10.10.1/24
  ipv4.dhcp: "true"
  ipv4.nat: "true"
  ipv6.address: none
description: Internal bridge network for web services
```

Network forward with external-to-internal address mapping:

```yaml
kind: network-forward
listen_address: 198.51.100.10
network: public
description: Shared external IP for web services
config:
  target_address: 10.42.0.10
ports:
  - protocol: tcp
    listen_port: "80"
    target_address: 10.42.0.11
    target_port: "8080"
    description: Web app HTTP
  - protocol: tcp
    listen_port: "443"
    target_address: 10.42.0.12
    target_port: "8443"
    description: Web app HTTPS
```

## Storage

Storage pool:

```yaml
# Storage pool
# Note: storage pools are global (not project-scoped)
kind: storage-pool
name: data
driver: dir
description: Data storage pool
```

Custom storage volume:

```yaml
# Custom storage volume
kind: storage-volume
name: data
pool: default
config:
  size: 20GiB
description: Persistent custom storage volume
```

S3-compatible storage bucket:

```yaml
# S3-compatible storage bucket
kind: storage-bucket
name: my-bucket
pool: default
config:
  size: 5GiB
description: S3-compatible object storage bucket
```

S3 credentials for a storage bucket:

```yaml
# S3 credentials for a storage bucket
kind: storage-bucket-key
name: app-key
bucket: my-bucket
pool: default
role: read-only
description: Read-only S3 credentials for application access
```

## Cloud-init

Alpine container with cloud-init:

```yaml
# Alpine container with cloud-init
# Writes files and runs a command to confirm cloud-init ran.
kind: instance
name: cloud-init-test
image: images:alpine/edge/cloud
config:
  limits.cpu: "1"
  limits.memory: 512MiB
  cloud-init.user-data:
    #cloud-config
    write_files:
      - path: /etc/motd
        content: "Provisioned by incus-apply\n"
    runcmd:
      - echo "cloud-init completed" > /run/cloud-init-done
description: Alpine container with cloud-init validation
```

