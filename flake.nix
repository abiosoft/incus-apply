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

          ldflags = [
            "-X main.version=${self.shortRev or self.dirtyShortRev or "dev"}"
            "-X main.commit=${self.rev or self.dirtyRev or "none"}"
            "-X main.date=${self.lastModifiedDate or "unknown"}"
          ];

          meta = {
            description = "Declarative configuration management for Incus";
            homepage = "https://github.com/abiosoft/incus-apply";
            mainProgram = "incus-apply";
          };
        };
      }
    )
    // {
      nixosModules.default =
        {
          config,
          lib,
          pkgs,
          ...
        }:
        let
          cfg = config.programs.incus-apply;
        in
        {
          options.programs.incus-apply.enable = lib.mkEnableOption "incus-apply, declarative configuration management for Incus";

          config = lib.mkIf cfg.enable {
            environment.systemPackages = [ self.packages.${pkgs.system}.default ];
          };
        };
    };
}
