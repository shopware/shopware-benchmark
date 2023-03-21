{ config, pkgs, vars, ... }: {
  sops.secrets = {
    githubRunnerToken = {
      sopsFile = ./secrets/github.yaml;
    };
  };

  services.github-runners.default = {
    enable = true;
    url = "https://github.com/shopware/shopware-benchmark";
    tokenFile = config.sops.secrets.githubRunnerToken.path;
    extraPackages = [
      pkgs.openssh
      pkgs.git
      pkgs.psmisc
    ];
    extraEnvironment = {
      TMP = "/tmp";
      TMPDIR = "/tmp";
      ANSIBLE_LOCAL_TEMP = "/tmp";
    };
  };
}
