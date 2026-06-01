# Reusable resources

The `examples/resources` directory contains standalone resource definitions you can apply individually or combine into larger stacks.

## Included files

| File | Purpose |
| --- | --- |
| [`instance.yaml`](https://github.com/abiosoft/incus-apply/blob/main/examples/resources/instance.yaml) | Basic system container example. |
| [`vm.yaml`](https://github.com/abiosoft/incus-apply/blob/main/examples/resources/vm.yaml) | Basic virtual machine example. |
| [`oci.yaml`](https://github.com/abiosoft/incus-apply/blob/main/examples/resources/oci.yaml) | OCI-based instance example. |
| [`profile.yaml`](https://github.com/abiosoft/incus-apply/blob/main/examples/resources/profile.yaml) | Reusable profile definition. |
| [`project.yaml`](https://github.com/abiosoft/incus-apply/blob/main/examples/resources/project.yaml) | Project definition for isolating resources. |
| [`network.yaml`](https://github.com/abiosoft/incus-apply/blob/main/examples/resources/network.yaml) | Network definition. |
| [`network-forward.yaml`](https://github.com/abiosoft/incus-apply/blob/main/examples/resources/network-forward.yaml) | Network forward example with external-to-internal address mapping. |
| [`storage-pool.yaml`](https://github.com/abiosoft/incus-apply/blob/main/examples/resources/storage-pool.yaml) | Storage pool definition. |
| [`storage-volume.yaml`](https://github.com/abiosoft/incus-apply/blob/main/examples/resources/storage-volume.yaml) | Custom storage volume definition. |
| [`storage-bucket.yaml`](https://github.com/abiosoft/incus-apply/blob/main/examples/resources/storage-bucket.yaml) | Object storage bucket definition. |
| [`storage-bucket-key.yaml`](https://github.com/abiosoft/incus-apply/blob/main/examples/resources/storage-bucket-key.yaml) | S3 credentials for a bucket. |
| [`cloud-init.yaml`](https://github.com/abiosoft/incus-apply/blob/main/examples/resources/cloud-init.yaml) | Cloud-init example suitable for instance config. |

## How to use them

1. Choose a file that matches the resource you want to create.
2. Copy it into your own configuration directory or apply it directly.
3. Adjust names, pools, images, and networks to match your environment.

Canonical source files:

- <https://github.com/abiosoft/incus-apply/tree/main/examples/resources>
