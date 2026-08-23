{
  description = "a beautiful TUI YouTube Downloader";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
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
        pkgs = nixpkgs.legacyPackages.${system};
        package = pkgs.callPackage ./nix/package.nix { };
      in
      {
        packages.default = package;

        apps = {
          default = flake-utils.lib.mkApp { drv = package; } // {
            meta.description = "a beautiful TUI YouTube Downloader";
          };
          update-vendor-hash = flake-utils.lib.mkApp {
            drv = pkgs.callPackage ./nix/update-vendor-hash.nix { };
          };
        };

        devShells.default = pkgs.mkShell {
          packages = [ pkgs.go_1_26 ];
        };
      }
    );
}
