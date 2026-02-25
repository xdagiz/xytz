{ config, lib, pkgs, ... }:

let
  cfg = config.programs.xytz;
in {
  options.programs.xytz = {
    enable = lib.mkEnableOption "xytz tool";
    package = lib.mkOption {
      type = lib.types.package;
      default = pkgs.callPackage ../. {}; # Akan di-override oleh flake
      description = "Package xytz yang akan diinstal.";
    };
  };

  config = lib.mkIf cfg.enable {
    environment.systemPackages = [ cfg.package ];
  };
}
