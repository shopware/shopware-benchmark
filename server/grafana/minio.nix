{ lib, config, pkgs, vars, ... }:

{
  sops.secrets = {
    minioEnv = {
      sopsFile = ./secrets/minio.yaml;
      owner = "minio";
      restartUnits = [ "minio.service" ];
    };
  };

  services.caddy = {
    virtualHosts."assets.sw-bench.de" = {
      extraConfig = ''
        reverse_proxy 127.0.0.1:9000

        encode zstd gzip
      '';
    };
  };

  services.minio = {
    enable = true;
    rootCredentialsFile = config.sops.secrets.minioEnv.path;
    listenAddress = "127.0.0.1:9000";
  };
}
