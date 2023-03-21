{ lib, config, pkgs, vars, ... }:

{
  nixpkgs.config.allowUnfree = true;

  sops.secrets = {
    blackfireEnv = {
      sopsFile = ./secrets/blackfire.yaml;
      restartUnits = [ "blackfire-agent.service" ];
    };
  };

  networking.firewall.interfaces.enp7s0.allowedTCPPorts = [ 8307 ];

  systemd.services.blackfire-agent.serviceConfig.EnvironmentFile = config.sops.secrets.blackfireEnv.path;
  systemd.services.blackfire-agent.wantedBy = [ "multi-user.target" ];

  services.blackfire-agent = {
    enable = true;

    settings = {
      server-id = "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX";
      server-token = "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX";
      socket = lib.mkForce "tcp://${vars.private_ip}:8307";
    };
  };
}
