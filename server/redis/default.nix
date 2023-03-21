{ config, lib, pkgs, vars, ... }: {
  boot.kernel.sysctl = {
    "vm.swappiness" = 1;
  };

  services.redis = {
    servers.default = {
      enable = true;
      port = 6379;
      appendOnly = false;
      bind = "${vars.private_ip} 127.0.0.1";
      unixSocket = null;
      save = [ ];
      settings = {
        maxmemory = "7g";
        protected-mode = "no";
        maxmemory-policy = "volatile-lru";
        tcp-backlog = 65536;
        maxclients = 10000;
      };
    };
  };

  systemd.services.redis-default.after = [ "network-online.target" ];
  systemd.services.redis-default.serviceConfig.Type = lib.mkForce "simple";

  services.prometheus.exporters.redis = {
    enable = true;
    listenAddress = vars.private_ip;
  };

  networking.firewall.interfaces.enp7s0.allowedTCPPorts = [
    config.services.prometheus.exporters.redis.port
    config.services.redis.servers."default".port
  ];

  systemd.services.prometheus-redis-exporter.after = [ "network-online.target" ];

  systemd.services.disable-transparent-huge-pages = {
      description = "Disable Transparent Huge Pages (required by Redis)";
      before = [ "redis-default.service" ];
      wantedBy = [ "redis-default.service" "multi-user.target" ];
      script = "echo never > /sys/kernel/mm/transparent_hugepage/enabled";
      serviceConfig.Type = "oneshot";
  };
}
