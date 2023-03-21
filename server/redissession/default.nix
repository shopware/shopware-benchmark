{ config, lib, pkgs, vars, ... }: {
  imports = [
    ../redis/default.nix
  ];

  services.rabbitmq = {
    enable = true;
    listenAddress = vars.private_ip;
    managementPlugin.enable = true;
    config = ''
      [{rabbit, [{loopback_users, []}]}].
    '';
  };

  networking.firewall.interfaces.enp7s0.allowedTCPPorts = [
    5672
  ];

  systemd.services.rabbitmq.after = [ "network-online.target" ];
}
