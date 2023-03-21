{ config, pkgs, vars, ... }: {
  services.opensearch = {
    enable = true;

    settings = {
      "http.host" = [ "127.0.0.1" vars.private_ip ];
      "network.host" = vars.private_ip;
      "network.publish_host" = vars.private_ip;
      "cluster.name" = "shopware";
      "node.name" = config.networking.hostName;
      "discovery.type" = "single-node";
    };
    extraJavaOptions = [ "-Xms20g" "-Xmx20g" ];
  };

  networking.firewall.interfaces.enp7s0.allowedTCPPorts = [ 9200 ];
}
