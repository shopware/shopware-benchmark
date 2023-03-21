{ config, pkgs, vars, ... }: {
  imports = [
    ./grafana.nix
    ./loki.nix
    ./prometheus.nix
    ./postgresql.nix
    ./minio.nix
    ./gitlab.nix
    ./blackfire.nix
  ];

  nix.extraOptions = ''
    experimental-features = nix-command flakes
  '';
  environment.systemPackages = [
    pkgs.openssh
    pkgs.git
  ];
}
