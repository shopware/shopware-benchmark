{ lib, pkgs, vars, serverCfg, ... }: {
  imports = [
    ./hardware.nix
  ];

  boot.cleanTmpDir = true;
  zramSwap.enable = true;
  networking.hostName = vars.server_name;
  networking.domain = "sw-bench.de";

  systemd.network.networks."10-uplink".networkConfig.Address = vars.public_ipv6;

  system.stateVersion = "23.05";

  environment.systemPackages = [
    pkgs.htop
    pkgs.tmux
  ];

  nix.gc.automatic = true;

  networking.firewall.interfaces.enp7s0.allowedTCPPorts = [ 9100 ];
  services.prometheus.exporters.node = {
    enable = true;
    listenAddress = vars.private_ip;
  };
  systemd.services.prometheus-node-exporter.after = [ "network-online.target" ];

  networking.hosts = serverCfg.hosts;

  services.openssh.enable = true;
  services.openssh.settings.PasswordAuthentication = false;
  users.users.root.openssh.authorizedKeys.keys = [
    # Soner
    "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAINlPUypv+9YSeE7HvERkY3toxodjGTckL2vnWdLJQAda"

    # CI User
    "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCUbuTkfz/R1tJRIzrxhLBgC18Kn2tmxwRGGxkuEKHEYAvVToYImid03SEhdRCXO+QJUalWqqtcqR9Pswrj/5EgXQkVJeaxjT70z5dadQViQGw5Ybx1xJ2Cl9RKWaQ9LfI5644e6yHkRy7MW8lDcamNnuyibxHNy+pobrgGFj/2a5RZB1GhYeT+kgQnWqu9HoYxotz/QBptXLW9FN9dDqCrb7tR6pLyhQ+BXbImyskTgcA1qRoUpdFF2QrvNEa4Fy14H8AJe+xyx6xrubEyfJvRRRLCwV7LBKPfc5kkjxxnvjDa2MWubYEfdnVK7mbmla5fEEKVuxi5LrXbvrtmkTIQKQA836ZaJeZAoPgx7VA8pJfsyNnEhmF1gMmGd3nIJjhLajPGbAMHRBz9zReMbNu4sKWB4AZ/rOpGKJuGoe3nWgXsXWrpPIAwCP+Ez2sK8c1tsutXz8WCzoTiWDCHtyNbZmjcGqqU+QGaa5x1OfAU7RLimov7N1rbCslXb7X6MuE="
  ];
}
