# Installation

Choose one of the following methods to install `incus-apply`.

## Install binary

Download and install the latest release binary for your system:

```bash
curl -LO https://github.com/abiosoft/incus-apply/releases/latest/download/incus-apply-$(uname)-$(uname -m)
sudo install incus-apply-$(uname)-$(uname -m) /usr/local/bin/incus-apply
```

## Build from source

Clone the repository and build:

```bash
git clone https://github.com/abiosoft/incus-apply
cd incus-apply
make
sudo make install
```

## Nix / NixOS

### Try it directly

[Flake-enabled systems](https://wiki.nixos.org/wiki/Flakes#Enabling_flakes_permanently) can try incus-apply directly with:

```bash
nix run github:abiosoft/incus-apply
```

To pass arguments, separate them with `--`:

```bash
nix run github:abiosoft/incus-apply -- debian.yaml
```

### Install it

To install it on NixOS, use the NixOS module. First add incus-apply to your flake inputs:

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
