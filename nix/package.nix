{
  lib,
  buildGo126Module,
  makeWrapper,
  yt-dlp,
  ffmpeg,
  mpv,
}:

buildGo126Module {
  pname = "xytz";
  version = "unstable";
  src = lib.cleanSource ./..;
  vendorHash = lib.strings.trim (builtins.readFile ./vendor-hash);
  doCheck = false;
  nativeBuildInputs = [ makeWrapper ];
  postInstall = ''
    wrapProgram "$out/bin/xytz" \
      --prefix PATH : ${
        lib.makeBinPath [
          yt-dlp
          ffmpeg
          mpv
        ]
      }
  '';

  meta = with lib; {
    description = "a beautiful TUI YouTube Downloader";
    homepage = "https://github.com/xdagiz/xytz";
    license = licenses.mit;
    mainProgram = "xytz";
  };
}
