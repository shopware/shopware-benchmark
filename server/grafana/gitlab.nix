{ config, pkgs, vars, ... }: {
  sops.secrets = {
    gitlabEnv = {
      sopsFile = ./secrets/gitlab.yaml;
      restartUnits = [ "gitlab-runner.service" ];
    };
  };
  services.gitlab-runner.enable = true;
  services.gitlab-runner.services.default = {
    registrationConfigFile = config.sops.secrets.gitlabEnv.path;
    executor = "shell";
    limit = 1;
    environmentVariables = {
      TMP = "/tmp";
      TMPDIR = "/tmp";
      ANSIBLE_LOCAL_TEMP = "/tmp";
    };
  };
}
