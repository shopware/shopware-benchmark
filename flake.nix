{
  description = "Benchmark Setup";

  inputs = {
    colmena = {
      url = "github:zhaofengli/colmena";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    sops-nix.url = "github:Mic92/sops-nix";
    sops-nix.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs = { self, nixpkgs, colmena, sops-nix }:
    let
      lastModifiedDate =
        self.lastModifiedDate or self.lastModified or "19700101";
      forAllSystems = nixpkgs.lib.genAttrs [ "x86_64-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin" ];
      serverCfg = builtins.fromJSON (builtins.readFile ./nix-config.json);
    in
    {
      colmena = {
        meta = {
          nixpkgs = import nixpkgs {
            system = "x86_64-linux";
          };
          specialArgs = {
            inherit serverCfg;
          };
          nodeSpecialArgs = serverCfg.servers;
        };

        elastic-1 = {
          deployment.targetHost = serverCfg.servers."elastic-1".vars.public_ip;
          imports = [
            sops-nix.nixosModules.sops
            ./server/common
            ./server/elastic
          ];
        };

        grafana-1 = {
          deployment.targetHost = serverCfg.servers."grafana-1".vars.public_ip;
          imports = [
            sops-nix.nixosModules.sops
            ./server/common
            ./server/grafana
          ];
        };

        redis-1 = {
          deployment.targetHost = serverCfg.servers."redis-1".vars.public_ip;
          imports = [
            sops-nix.nixosModules.sops
            ./server/common
            ./server/redis
          ];
        };

        redissession-1 = {
          deployment.targetHost = serverCfg.servers."redissession-1".vars.public_ip;
          imports = [
            sops-nix.nixosModules.sops
            ./server/common
            ./server/redissession
          ];
        };

        mysql-1 = {
          deployment.targetHost = serverCfg.servers."mysql-1".vars.public_ip;
          imports = [
            sops-nix.nixosModules.sops
            ./server/common
            ./server/mysql
          ];
        };

        mysql-2 = {
          deployment.targetHost = serverCfg.servers."mysql-2".vars.public_ip;
          imports = [
            sops-nix.nixosModules.sops
            ./server/common
            ./server/mysql
          ];
        };

        mysql-3 = {
          deployment.targetHost = serverCfg.servers."mysql-3".vars.public_ip;
          imports = [
            sops-nix.nixosModules.sops
            ./server/common
            ./server/mysql
          ];
        };

        app-1 = {
          deployment.targetHost = serverCfg.servers."app-1".vars.public_ip;
          imports = [
            sops-nix.nixosModules.sops
            ./server/common
            ./server/app
          ];
        };

        app-2 = {
          deployment.targetHost = serverCfg.servers."app-2".vars.public_ip;
          imports = [
            sops-nix.nixosModules.sops
            ./server/common
            ./server/app
          ];
        };

        app-3 = {
          deployment.targetHost = serverCfg.servers."app-3".vars.public_ip;
          imports = [
            sops-nix.nixosModules.sops
            ./server/common
            ./server/app
          ];
        };
      };

      packages = forAllSystems (system:
        let pkgs = nixpkgs.legacyPackages.${system};
        in rec {
          benchmark-setup = pkgs.buildGoModule {
            name = "benchmark-setup";
            src = ./.;
            vendorSha256 = "sha256-xZcgfX8gL28HlOxH4KX7qZabPTR6ByAXLuk0Ng49hfM";
          };
          ansible = pkgs.ansible_2_13.overrideAttrs (old: rec {
            propagatedBuildInputs = old.propagatedBuildInputs ++ [
              pkgs.python310Packages.jmespath
              pkgs.gnutar
            ];
          });

          default = benchmark-setup;
        });

      apps = forAllSystems (system: rec {
        benchmark-setup = {
          type = "app";
          program = "${self.packages.${system}.benchmark-setup}/bin/benchmark-setup";
        };
        default = benchmark-setup;
      });

      defaultPackage = forAllSystems (system: self.packages.${system}.default);

      defaultApp = forAllSystems (system: self.apps.${system}.default);

      devShell = forAllSystems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        pkgs.mkShell {
          buildInputs = with pkgs; [
            go_1_20
            golangci-lint
            self.packages.${system}.default
            self.packages.${system}.ansible
            rsync
            gnutar
            openssh
            colmena.packages.${system}.colmena
          ];
        });

      formatter = forAllSystems (
        system:
        nixpkgs.legacyPackages.${system}.nixpkgs-fmt
      );
    };
}
