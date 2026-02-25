{ config, lib, pkgs, ... }:

let
  cfg = config.programs.xytz;
in {
  options.programs.xytz = {
    enable = lib.mkEnableOption "xytz tool";
    package = lib.mkOption {
      type = lib.types.package;
      default = pkgs.callPackage ../. {};
      description = "Package xytz yang akan diinstal.";
    };
  };

  config = lib.mkIf cfg.enable {
    home.packages = [ cfg.package ];
  };
}
