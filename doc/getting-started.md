# Getting started

Use this section to install `incus-apply`, apply your first resource, and understand the minimum configuration model.

## Installation

### Linux / MacOS

#### Install a release binary

```bash
curl -LO https://github.com/abiosoft/incus-apply/releases/latest/download/incus-apply-$(uname)-$(uname -m)
sudo install incus-apply-$(uname)-$(uname -m) /usr/local/bin/incus-apply
```

#### Or build from source

```bash
git clone https://github.com/abiosoft/incus-apply
cd incus-apply
make
sudo make install
```

### Nix / NixOS

[Flake-enabled systems](https://wiki.nixos.org/wiki/Flakes#Enabling_flakes_permanently) can try incus-apply directly with:
```bash
nix run github:abiosoft/incus-apply
```

To pass arguments, separate them with `--`:
```bash
nix run github:abiosoft/incus-apply -- debian.yaml
```

To install it, use the NixOS module. First add incus-apply to your flake inputs:
```nix
inputs.incus-apply.url = "github:abiosoft/incus-apply";
inputs.incus-apply.inputs.nixpkgs.follows = "nixpkgs";
```

Then import the module in your `nixosConfigurations` and enable it:
```nix
{
  outputs = { self, nixpkgs, incus-apply, ... }: {
    nixosConfigurations.yourhostname = nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      modules = [
        ./configuration.nix
        incus-apply.nixosModules.default
        { programs.incus-apply.enable = true; }
      ];
    };
  };
}
```

This installs the `incus-apply` binary system-wide.


## Quick start

### Prerequisites

- A working Incus installation.
- Access to the target local or remote Incus server through the `incus` CLI.

### Create a resource file

Create a file named `debian.yaml`:

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

### Apply it

```bash
incus-apply debian.yaml
```

`incus-apply` loads the file, computes the diff against the current Incus state, shows a preview, and asks for confirmation before applying changes.

## Configuration format

Configuration files can be `.yaml`, `.yml`, or `.json`. When you point `incus-apply` at a directory, it scans all YAML and JSON files and processes only the documents whose `kind` matches a supported Incus resource.

## Multi-document YAML

You can define multiple resources in a single file by separating documents with `---`:

```yaml
---
kind: profile
name: app-profile
config:
  limits.memory: 512MiB
---
kind: instance
name: app-1
image: images:alpine/3.19
profiles:
  - default
  - app-profile
---
kind: instance
name: app-2
image: images:alpine/3.19
profiles:
  - default
  - app-profile
```

## Variables

Declare variables with `kind: vars` and reference them with `$VAR` or `${VAR}` in resource documents.

```yaml
---
kind: vars
vars:
  NODE_ENV: production
  MYSQL_DATABASE: app
---
kind: instance
name: api
image: docker:node:20
config:
  environment.NODE_ENV: $NODE_ENV
  environment.MYSQL_DATABASE: $MYSQL_DATABASE
```

For more detail, continue with the [workflows](../tutorials/workflows/) section or jump to the [configuration reference](reference/configuration/).
