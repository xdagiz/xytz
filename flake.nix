{
  description = "xytz - YouTube from your terminal";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "xytz";
          version = "0.0.0-dev";
          src = ./.;
          vendorHash = "sha256-loLssmeKd6paM2cMXzGE+xRozgOzlpOLARfP9+ZruGI=";

          # xytz shells out to these tools for media operations.
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
            platforms = platforms.unix;
          };
        };

        apps.default = flake-utils.lib.mkApp {
          drv = self.packages.${system}.default;
        };

        devShells.default = pkgs.mkShell {
          packages = [
            pkgs.go_1_25
            pkgs.gopls
            pkgs.gotools
            pkgs.yt-dlp
            pkgs.ffmpeg
            pkgs.mpv
          ];
        };
      }
    );
}
