{ config, pkgs, vars, ... }: {
  imports = [
    ./grafana.nix
    ./loki.nix
    ./prometheus.nix
    ./postgresql.nix
    ./minio.nix
    ./github.nix
    ./blackfire.nix
  ];

  nix.extraOptions = ''
    experimental-features = nix-command flakes
  '';
}
