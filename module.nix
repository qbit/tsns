{ lib
, config
, pkgs
, ...
}:
with lib;
let
  cfg = config.services.tsns;
in
{
  options = {
    services.tsns = {
      package = mkPackageOption pkgs "tsns" { };

      enable = lib.mkEnableOption "Enable tsns";

      user = mkOption {
        type = with types; oneOf [ str int ];
        default = "tsns";
        description = ''
          The user the service will use.
        '';
      };

      group = mkOption {
        type = with types; oneOf [ str int ];
        default = "tsns";
        description = ''
          The group the service will use.
        '';
      };

      dataDir = mkOption {
        type = types.path;
        default = "/var/lib/tsns";
        description = "Path tsns home directory";
      };
    };
  };

  config = mkIf cfg.enable {
    environment.systemPackages = [ cfg.package ];

    users.groups."${cfg.group}" = {};
    users.users."${cfg.user}" = {
        description = "System user for tsns instance ${cfg.user}";
        isSystemUser = true;
        group = cfg.group;
        home = "${cfg.dataDir}";
        createHome = true;
    };

    systemd.services.tsns = {
      description = "tsns";
      enable = true;
      after = [ "network-online.target" ];
      wants = [ "network-online.target" ];
      wantedBy = [ "multi-user.target" ];
      
      environment = { HOME = "${cfg.dataDir}"; };
      
      serviceConfig = {
        User = cfg.user;
        Group = cfg.group;
        ExecStart = "${cfg.package}/bin/tsns -data ${cfg.dataDir}";
      };
    };
  };
}
