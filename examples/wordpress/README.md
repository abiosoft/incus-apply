# WordPress

WordPress deployment examples in three variants: OCI containers, system container, and virtual machine.

All variants provision a fully functional WordPress site with a database backend and a Caddy frontend, accessible on port 80.

## Usage

```sh
incus-apply <yaml file>
```

## Examples

### OCI Containers (`wordpress-oci.yaml`)

Runs WordPress and MySQL as OCI (Docker) containers using Incus's native OCI support.

**Prerequisites:** add the Docker remote first:
```sh
incus remote add docker https://docker.io --protocol=oci
```

**Environment variables:**
- `MYSQL_ROOT_PASSWORD` (default: `rootpassword`)
- `MYSQL_PASSWORD` (default: `wordpresspassword`)

```sh
incus-apply wordpress-oci.yaml
```

---

### System Container (`wordpress.yaml`)

Provisions MariaDB, Caddy, and WordPress on a single Debian 13 system container using cloud-init.

**Environment variables:**
- `MARIADB_PASSWORD` (default: `wordpresspassword`)

```sh
incus-apply wordpress.yaml
```

---

### Virtual Machine (`wordpress-vm.yaml`)

Provisions MariaDB, Caddy, and WordPress on a single Debian 13 VM using cloud-init, with a block storage volume for persistent database storage.

**Environment variables:**
- `MARIADB_PASSWORD` (default: `wordpresspassword`)

```sh
incus-apply wordpress-vm.yaml
```
