{
  inputs = {
    nixpkgs.url = "nixpkgs";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:

    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      {
        packages.default = pkgs.buildGo126Module {
          pname = "incus-apply";
          name = "incus-apply";
          src = ./.;
          vendorHash = "sha256-u+nl3P7YNl+3DJIXo7pnDKF4PkoYLaHf3B1LqF9b+V8=";

          meta = {
            description = "Declarative configuration management for Incus";
            homepage = "https://github.com/abiosoft/incus-apply";
            mainProgram = "incus-apply";
          };
        };
      }
    );
}
