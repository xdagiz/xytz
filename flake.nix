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
            src = pkgs.lib.cleanSource ./.;

            # Replace with the appropriate hash if using vendor, or leave null if gomod
            vendorHash = "sha256-loLssmeKd6paM2cMXzGE+xRozgOzlpOLARfP9+ZruGI=";

            doCheck = false;
            nativeBuildInputs = [ pkgs.makeWrapper ];

            postInstall = ''
              wrapProgram $out/bin/xytz \
                --prefix PATH : ${pkgs.lib.makeBinPath [ pkgs.mpv pkgs.yt-dlp pkgs.ffmpeg ]}
            '';

            meta = with pkgs.lib; {
              description = "xytz – media helper utility";
              homepage = "https://github.com/xdagiz/xytz";
              license = licenses.mit;
              mainProgram = "xytz";
            };
          };
        });

      # 2. Export NixOS Module
      nixosModules.default = { config, lib, pkgs, ... }: {
        imports = [ ./nix/nixos-module.nix ];
        _module.args.self = self;
      };

      # 3. Export Home Manager Module
      homeManagerModules.default = { config, lib, pkgs, ... }: {
        imports = [ ./nix/home-manager-module.nix ];
        _module.args.self = self;
      };
    };
}
