{ lib, config, pkgs, serverCfg, vars, ... }: {
  services.prometheus = {
    enable = true;
    scrapeConfigs = [
      {
        job_name = "all";
        static_configs = [
          {
            targets = map (server: "${server.vars.private_ip}:9100") (lib.attrsets.attrValues serverCfg.servers);
          }
        ] ++ [{
          targets = map (server: "${server.vars.private_ip}:9121") (builtins.filter (server: lib.strings.hasPrefix "redis" server.vars.server_name) (lib.attrsets.attrValues serverCfg.servers));
        }];
      }
    ];
  };
}
