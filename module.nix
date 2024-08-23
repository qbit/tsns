{ lib
, config
, pkgs
, ...
}:
with lib;
let
  cfg = config.services.tsns;
  enabledServers = filterAttrs (_: conf: conf.enable) cfg.servers;
in
{
  options = {
    services.tsns = {
      package = mkPackageOption pkgs "tsns" { };

      enable = lib.mkEnableOption "Enable tsns for ${name}";

      user = mkOption {
        type = with types; oneOf [ str int ];
        default = name;
        description = ''
          The user the service will use.
        '';
      };

      group = mkOption {
        type = with types; oneOf [ str int ];
        default = name;
        description = ''
          The group the service will use.
        '';
      };

      dataDir = mkOption {
        type = types.path;
        default = "/var/lib/${name}";
        description = "Path tsns home directory";
      };
    };
  };

  config = mkIf (enabledServers != { }) {
    environment.systemPackages = [ cfg.package ];

    users.groups = mapAttrs'
      (name: _: nameValuePair name { })
      enabledServers;
    users.users = mapAttrs'
      (name: conf: nameValuePair name {
        description = "System user for tsns instance ${name}";
        isSystemUser = true;
        group = name;
        home = "${conf.dataDir}";
        createHome = true;
      })
      enabledServers;

    systemd.services = mapAttrs'
      (name: conf: nameValuePair name {
        description = "tsns instance ${name}";
        enable = true;
        after = [ "network-online.target" ];
        wants = [ "network-online.target" ];
        wantedBy = [ "multi-user.target" ];

        environment = { HOME = "${conf.dataDir}"; };

        serviceConfig = {
          User = conf.user;
          Group = conf.group;
          ExecStart = "${cfg.package}/bin/tsns -d ${conf.dataDir}";
        };
      })
      enabledServers;
  };
}
