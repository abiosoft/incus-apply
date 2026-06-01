# WordPress

The WordPress examples show three ways to deploy the same application stack with `incus-apply`: OCI containers, a Debian system container, or a virtual machine.

All variants provision a working WordPress site with a database backend and a Caddy frontend exposed on port 80.

## OCI containers

[`wordpress-oci.yaml`](https://github.com/abiosoft/incus-apply/blob/main/examples/wordpress/wordpress-oci.yaml) runs WordPress and MySQL as OCI containers by using Incus OCI support.

Prerequisite:

```sh
incus remote add docker https://docker.io --protocol=oci
```

Environment variables:

- `MYSQL_ROOT_PASSWORD` with default `rootpassword`
- `MYSQL_PASSWORD` with default `wordpresspassword`

```sh
incus-apply wordpress-oci.yaml
```

## Debian system container

[`wordpress.yaml`](https://github.com/abiosoft/incus-apply/blob/main/examples/wordpress/wordpress.yaml) provisions MariaDB, Caddy, and WordPress inside a single Debian 13 system container using cloud-init.

Environment variables:

- `MARIADB_PASSWORD` with default `wordpresspassword`

```sh
incus-apply wordpress.yaml
```

## Virtual machine

[`wordpress-vm.yaml`](https://github.com/abiosoft/incus-apply/blob/main/examples/wordpress/wordpress-vm.yaml) provisions the same stack inside a Debian 13 virtual machine and adds a block storage volume for persistent database storage.

Environment variables:

- `MARIADB_PASSWORD` with default `wordpresspassword`

```sh
incus-apply wordpress-vm.yaml
```

Canonical source files:

- <https://github.com/abiosoft/incus-apply/tree/main/examples/wordpress>
