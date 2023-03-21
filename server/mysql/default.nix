{ config, pkgs, vars, lib, serverCfg, ... }:

let
  serverId = lib.strings.toInt (builtins.replaceStrings [ "mysql-" ] [ "" ] vars.server_name);
  isMariaDB = lib.getName cfg.package == lib.getName pkgs.mariadb;
  cfg = config.services.mysql;
in
{
  services.mysql = {
    enable = true;
    package = pkgs.mysql80;

    settings = {
      mysqld = {
        "bind-address" = vars.private_ip;
        innodb_buffer_pool_size = "20G";
        innodb_log_file_size = "5G";
        sql-mode = "";
        max_connections = 2000;
        group_concat_max_len = 320000;
        tmp_table_size = "16M";
        max_heap_table_size = "16M";
        join_buffer_size = "262144";
      };
    };

    replication = {
      serverId = serverId;
      slaveHost = "10.0.0.0/255.255.0.0";
      masterHost = serverCfg.servers."mysql-1".vars.private_ip;
      masterPort = 3306;
      masterUser = "sync";
      masterPassword = "sync";
      role = if serverId == 1 then "master" else "slave";
    };
  };

  networking.firewall.interfaces.enp7s0.allowedTCPPorts = [ 3306 ];

  # Force the connection stuff
  systemd.services.mysql.preStart = ''
    touch /var/lib/mysql/mysql_init
  '';

  systemd.services.mysql.postStart = lib.mkForce (
    let
      # The super user account to use on *first* run of MySQL server
      superUser = if isMariaDB then cfg.user else "root";
    in
    ''
      ${lib.optionalString (!isMariaDB) ''
        # Wait until the MySQL server is available for use
        count=0
        while [ ! -e /run/mysqld/mysqld.sock ]
        do
            if [ $count -eq 30 ]
            then
                echo "Tried 30 times, giving up..."
                exit 1
            fi
            echo "MySQL daemon not yet started. Waiting for 1 second..."
            count=$((count++))
            sleep 1
        done
      ''}
      if [ -f ${cfg.dataDir}/mysql_init ]
      then
          # While MariaDB comes with a 'mysql' super user account since 10.4.x, MySQL does not
          # Since we don't want to run this service as 'root' we need to ensure the account exists on first run
          ( echo "CREATE USER IF NOT EXISTS '${cfg.user}'@'localhost' IDENTIFIED WITH ${if isMariaDB then "unix_socket" else "auth_socket"};"
            echo "GRANT ALL PRIVILEGES ON *.* TO '${cfg.user}'@'localhost' WITH GRANT OPTION;"
          ) | ${cfg.package}/bin/mysql -u ${superUser} -N
          ${lib.concatMapStrings (database: ''
            # Create initial databases
            if ! test -e "${cfg.dataDir}/${database.name}"; then
                echo "Creating initial database: ${database.name}"
                ( echo 'create database `${database.name}`;'
                  ${lib.optionalString (database.schema != null) ''
                  echo 'use `${database.name}`;'
                  # TODO: this silently falls through if database.schema does not exist,
                  # we should catch this somehow and exit, but can't do it here because we're in a subshell.
                  if [ -f "${database.schema}" ]
                  then
                      cat ${database.schema}
                  elif [ -d "${database.schema}" ]
                  then
                      cat ${database.schema}/mysql-databases/*.sql
                  fi
                  ''}
                ) | ${cfg.package}/bin/mysql -u ${superUser} -N
            fi
          '') cfg.initialDatabases}
          ${lib.optionalString (cfg.replication.role == "master")
            ''
              # Set up the replication master
              ( echo "use mysql;"
                ${if isMariaDB then ''
                  echo "CREATE USER '${cfg.replication.masterUser}'@'${cfg.replication.slaveHost}' IDENTIFIED WITH mysql_native_password;"
                  echo "SET PASSWORD FOR '${cfg.replication.masterUser}'@'${cfg.replication.slaveHost}' = PASSWORD('${cfg.replication.masterPassword}');"
                '' else ''
                  echo "CREATE USER IF NOT EXISTS '${cfg.replication.masterUser}'@'${cfg.replication.slaveHost}' IDENTIFIED WITH mysql_native_password BY '${cfg.replication.masterPassword}';"
                ''}
                echo "GRANT REPLICATION SLAVE ON *.* TO '${cfg.replication.masterUser}'@'${cfg.replication.slaveHost}';"
              ) | ${cfg.package}/bin/mysql -u ${superUser} -N
            ''}
          ${lib.optionalString (cfg.replication.role == "slave")
            ''
              # Set up the replication slave
              ( echo "stop slave;"
                echo "change master to master_host='${cfg.replication.masterHost}', master_user='${cfg.replication.masterUser}', master_password='${cfg.replication.masterPassword}';"
                echo "start slave;"
              ) | ${cfg.package}/bin/mysql -u ${superUser} -N
            ''}
          ${lib.optionalString (cfg.initialScript != null)
            ''
              # Execute initial script
              # using toString to avoid copying the file to nix store if given as path instead of string,
              # as it might contain credentials
              cat ${toString cfg.initialScript} | ${cfg.package}/bin/mysql -u ${superUser} -N
            ''}
          rm ${cfg.dataDir}/mysql_init
      fi
      ${lib.optionalString (cfg.ensureDatabases != []) ''
        (
        ${lib.concatMapStrings (database: ''
          echo "CREATE DATABASE IF NOT EXISTS \`${database}\`;"
        '') cfg.ensureDatabases}
        ) | ${cfg.package}/bin/mysql -N
      ''}
      ${lib.concatMapStrings (user:
        ''
          ( echo "CREATE USER IF NOT EXISTS '${user.name}'@'localhost' IDENTIFIED WITH ${if isMariaDB then "unix_socket" else "auth_socket"};"
            ${lib.concatStringsSep "\n" (lib.mapAttrsToList (database: permission: ''
              echo "GRANT ${permission} ON ${database} TO '${user.name}'@'localhost';"
            '') user.ensurePermissions)}
          ) | ${cfg.package}/bin/mysql -N
        '') cfg.ensureUsers}
    ''
  );
}
