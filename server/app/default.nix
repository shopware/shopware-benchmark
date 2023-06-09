{ config, lib, pkgs, vars, serverCfg, ... }:

let
  grafanaIp = serverCfg.servers.grafana-1.vars.private_ip;
  php = pkgs.php82.buildEnv {
    extensions = ({ enabled, all }: enabled ++ (with all; [
      amqp
      redis
      blackfire
    ]));
    extraConfig = ''
      memory_limit=512M
      post_max_size=32M
      upload_max_filesize=32M
      session.gc_probability=0
      session.save_handler = redis
      session.save_path = "tcp://${serverCfg.servers.redissession-1.vars.private_ip}:6379/0?persistent=1"
      assert.active=0
      date.timezone=UTC
      opcache.enable_file_override=1
      opcache.interned_strings_buffer=20
      opcache.preload=/var/www/html/var/cache/opcache-preload.php
      opcache.preload_user=caddy
      zend.detect_unicode=0
      realpath_cache_ttl=3600
      redis.clusters.cache_slots = 1
      blackfire.agent_socket = "tcp://${grafanaIp}:8307"
    '';
  };
in
{
  nixpkgs.config.allowUnfree = true;
  environment.systemPackages = [
    php
    php.packages.composer
    pkgs.nodejs-18_x
    pkgs.rsync
    pkgs.git
    pkgs.python311
  ];

  services.phpfpm.pools.web = {
    user = "caddy";
    group = "caddy";
    phpPackage = php;
    settings = {
      "clear_env" = "no";
      "listen.owner" = "caddy";
      "pm" = "static";
      "pm.max_children" = 100;
      "pm.start_servers" = 40;
      "pm.min_spare_servers" = 40;
      "pm.max_spare_servers" = 100;
      "rlimit_files" = 64000;
    };
  };

  services.caddy = {
    enable = true;

    virtualHosts.":80" = {
      extraConfig = ''
        php_fastcgi unix/${config.services.phpfpm.pools.web.socket}
        root * /var/www/html/public
      '';
    };
  };

  networking.firewall.allowedTCPPorts = [ 80 ];

  systemd.services.phpfpm-web.wantedBy = lib.mkForce [ ];
  systemd.services.phpfpm-web.partOf = lib.mkForce [ ];

  systemd.services."shopware-worker@" = {
    serviceConfig = {
      Type = "simple";
      User = "caddy";
      Restart = "always";
      ExecStart = "${php}/bin/php /var/www/html/bin/console messenger:consume --time-limit=60 --memory-limit=512M async";
      TimeoutStopFailureMode = "kill";
      TimeoutStopSec = "1s";
    };
  };

  systemd.services."shopware_scheduled_task" = {
    serviceConfig = {
      Type = "simple";
      User = "caddy";
      Restart = "always";
      ExecStart = "${php}/bin/php /var/www/html/bin/console scheduled-task:run --time-limit=60 --memory-limit=512M";
      TimeoutStopFailureMode = "kill";
      TimeoutStopSec = "1s";
    };
  };

  services.promtail = {
    enable = true;
    configuration = {
      server = {
        http_listen_port = 9080;
      };
      clients = [
        {
          url = "http://${grafanaIp}:3100/loki/api/v1/push";
        }
      ];
      scrape_configs = [
        {
          job_name = "shopware";
          static_configs = [
            {
              targets = [ "localhost" ];
              labels = {
                app = "shopware";
                host = vars.server_name;
                job = "varlogs";
                __path__ = "/var/www/html/var/log/*.log";
              };
            }
          ];
        }
      ];
    };
  };
}
