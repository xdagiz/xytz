{ config, lib, pkgs, self, ... }:

let
  cfg = config.programs.xytz;
in {
  options.programs.xytz = {
    enable = lib.mkEnableOption "xytz tool";
    package = lib.mkOption {
      type = lib.types.package;
      default = self.packages.${pkgs.system}.default;
      description = "Which Package xytz installed.";
    };
  };

  config = lib.mkIf cfg.enable {
    environment.systemPackages = [ cfg.package ];
  };
}
