{ config, pkgs, vars, ... }: {
  networking.firewall.interfaces.enp7s0.allowedTCPPorts = [ 3100 ];

  services.loki = {
    enable = true;
    configuration = {
      auth_enabled = false;
      server = {
        http_listen_port = 3100;
        http_listen_address = vars.private_ip;
      };

      ingester = {
        lifecycler = {
          address = "127.0.0.1";
          ring = {
            kvstore = {
              store = "inmemory";
            };
            replication_factor = 1;
          };
          final_sleep = 0;
        };
        chunk_idle_period = "5m";
        chunk_retain_period = "30s";
        max_transfer_retries = 0;
        wal = {
          enabled = true;
          dir = "/var/lib/loki/wal";
        };
      };

      schema_config = {
        configs = [
          {
            from = "2018-04-15";
            store = "boltdb";
            object_store = "filesystem";
            schema = "v11";
            index = {
              prefix = "index_";
              period = "168h";
            };
          }
        ];
      };

      storage_config.boltdb.directory = "/tmp/loki/index";
      storage_config.filesystem.directory = "/tmp/loki/chunks";
    };
  };
}
