{
  description = "xytz flake and modules";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }: 
    let
      supportedSystems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });
    in
    {
      # 1. Build Package Go
      packages = forAllSystems (system:
        let pkgs = nixpkgsFor.${system}; in {
          default = pkgs.buildGoModule {
            pname = "xytz";
            version = "unstable";
            src = ./.;
            vendorHash = "sha256-loLssmeKd6paM2cMXzGE+xRozgOzlpOLARfP9+ZruGI="; # Ganti dengan hash yang sesuai jika menggunakan vendor, atau biarkan null jika gomod

            doCheck = false;

            nativeBuildInputs = [ pkgs.makeWrapper ];
            postInstall = ''
              wrapProgram $out/bin/xytz --prefix PATH : ${pkgs.lib.makeBinPath [ pkgs.mpv ]}            
            '';
          };
        });

      # 2. Export NixOS Module
      nixosModules.default = { config, lib, pkgs, ... }: {
        imports = [ ./nix/nixos-module.nix ];
      };

      # 3. Export Home Manager Module
      homeManagerModules.default = { config, lib, pkgs, ... }: {
        imports = [ ./nix/home-manager-module.nix ];
      };
    };
}
