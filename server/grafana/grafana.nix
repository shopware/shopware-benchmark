{ config, pkgs, vars, ... }: {
  users.users.grafana.extraGroups = [ "caddy" ];

  services.caddy = {
    enable = true;

    virtualHosts."grafana.sw-bench.de" = {
      extraConfig = ''
        reverse_proxy unix//run/grafana/sock

        encode zstd gzip
      '';
    };
  };

  networking.firewall.allowedTCPPorts = [ 80 443 ];
  networking.firewall.allowedUDPPorts = [ 443 ];

  sops.secrets = {
    grafanaEnv = {
      sopsFile = ./secrets/grafana.yaml;
      owner = "grafana";
      restartUnits = [ "grafana.service" ];
    };
  };

  systemd.services.grafana.serviceConfig.EnvironmentFile = config.sops.secrets.grafanaEnv.path;

  services.grafana = {
    enable = true;

    settings = {
      analytics.reporting_enabled = false;

      server = {
        protocol = "socket";
        socket = "/run/grafana/sock";
        socket_gid = config.users.groups.caddy.gid;
        root_url = "https://grafana.sw-bench.de";
        domain = "grafana.sw-bench.de";
      };

      users = {
        allow_sign_up = false;
        allow_org_create = false;
      };

      "auth.basic" = {
        enabled = false;
      };

      "auth.azuread" = {
        allowed_domains = "shopware.com";
        auto_login = true;
        allow_assign_grafana_admin = true;
        skip_org_role_sync = true;
        enabled = true;
      };
    };
  };
}
