{ config, lib, pkgs, self, ... }:

let
  cfg = config.programs.xytz;
in {
  options.programs.xytz = {
    enable = lib.mkEnableOption "xytz tool";
    package = lib.mkOption {
      type = lib.types.package;
      default = self.packages.${pkgs.stdenv.hostPlatform.system}.default;
      defaultText = lib.literalExpression "self.packages.\${pkgs.stdenv.hostPlatform.system}.default";
      description = "The xytz package to install.";
    };
  };

  config = lib.mkIf cfg.enable {
    environment.systemPackages = [ cfg.package ];
  };
}
