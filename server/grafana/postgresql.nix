{ lib, config, pkgs, vars, ... }:

{
  networking.firewall.interfaces.enp7s0.allowedTCPPorts = [ 5432 ];

  services.postgresql = {
    enable = true;
    package = pkgs.postgresql_15;
    settings = {
      listen_addresses = lib.mkForce vars.private_ip;
      shared_preload_libraries = "timescaledb";
    };
    authentication = ''
      host    all         all         10.0.0.0/24    trust
    '';
    extraPlugins = [ pkgs.postgresql15Packages.timescaledb ];
  };
}
