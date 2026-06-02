# incus-apply

[![Go](https://github.com/abiosoft/incus-apply/actions/workflows/go.yml/badge.svg)](https://github.com/abiosoft/incus-apply/actions/workflows/go.yml)
[![Integration](https://github.com/abiosoft/incus-apply/actions/workflows/integration.yml/badge.svg)](https://github.com/abiosoft/incus-apply/actions/workflows/integration.yml)

Declarative configuration management for [Incus](https://linuxcontainers.org/incus/).

![incus-apply demo](./demo.gif)

## Installation

Install the latest release binary:

```bash
curl -LO https://github.com/abiosoft/incus-apply/releases/latest/download/incus-apply-$(uname)-$(uname -m)
sudo install incus-apply-$(uname)-$(uname -m) /usr/local/bin/incus-apply
```

Or explore other [installation options](https://incus-apply.abiosoft.com/installation).

## Quick Start

### 1. Create a config file `debian.yaml`:

```yaml
kind: instance
name: debian
image: images:debian/12
profiles:
  - default
config:
  limits.cpu: "2"
  limits.memory: 1GiB
```

### 2. Apply it:

```bash
incus-apply debian.yaml
```

## Documentation

Check the [project website](https://incus-apply.abiosoft.com).

## License

Apache 2.0

## Sponsoring the Project

If you (or your company) are benefiting from the project and would like to support the contributors, kindly sponsor.

- [Github Sponsors](https://github.com/sponsors/abiosoft)
- [Buy me a coffee](https://www.buymeacoffee.com/abiosoft)

