{ lib, modulesPath, ... }:
{
  imports = [ (modulesPath + "/profiles/qemu-guest.nix") ];
  boot.tmp.cleanOnBoot = true;
  boot.growPartition = true;
  boot.loader.grub.device = "/dev/sda";

  fileSystems."/" = lib.mkDefault { device = "/dev/sda1"; fsType = "ext4"; };
}
