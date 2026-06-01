{
  description = "xytz - YouTube from your terminal";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    let
      supportedSystems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });
    in
    {
      packages = forAllSystems (system:
        let pkgs = nixpkgsFor.${system}; in {
          default = pkgs.buildGo126Module {
            pname = "xytz";
            version = "unstable";
            src = pkgs.lib.cleanSource ./.;
            vendorHash = "sha256-j4K61ESqtlfOD8S3E0vtL18aziSFztoU3V0KSLtJEME=";
            doCheck = false;
            nativeBuildInputs = [ pkgs.makeWrapper ];
            postInstall = ''
              wrapProgram "$out/bin/xytz" \
                --prefix PATH : ${pkgs.lib.makeBinPath [
                  pkgs.yt-dlp
                  pkgs.ffmpeg
                  pkgs.mpv
                ]}
            '';

            meta = with pkgs.lib; {
              description = "A TUI app for searching and downloading YouTube videos";
              homepage = "https://github.com/xdagiz/xytz";
              license = licenses.mit;
              mainProgram = "xytz";
            };
          };
        });

      apps = forAllSystems (system: {
        default = flake-utils.lib.mkApp {
          drv = self.packages.${system}.default;
        };
      });

      devShells = forAllSystems (system:
        let pkgs = nixpkgsFor.${system}; in {
          default = pkgs.mkShell {
            packages = [ pkgs.go_1_25 ];
          };
        });
    };
}
