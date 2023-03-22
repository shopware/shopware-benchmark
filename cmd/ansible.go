package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/hetznercloud/hcloud-go/hcloud"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type InventoryConfig map[string]*InventoryGroup

type InventoryGroup struct {
	Hosts     map[string]interface{} `yaml:"hosts"`
	Variables map[string]interface{} `yaml:"vars,omitempty"`
}

type NixConfig struct {
	Servers NixServerConfig     `json:"servers"`
	Hosts   map[string][]string `json:"hosts"`
}
type NixServerConfig map[string]map[string]map[string]string

func generateAnsibleInventory(infraCfg *InfraConfig, client *hcloud.Client, ctx context.Context) error {
	servers, err := getServers(client, ctx)

	if err != nil {
		return err
	}

	log.Debugf("Found %d servers", len(servers))

	inventoryFile := InventoryConfig{}
	nixConfig := NixConfig{}
	nixConfig.Servers = make(NixServerConfig, 0)
	nixConfig.Hosts = make(map[string][]string, 0)

	for group, servers := range servers {
		invGroup, ok := inventoryFile[group]

		if !ok {
			inventoryFile[group] = &InventoryGroup{Hosts: map[string]interface{}{}}
			invGroup, _ = inventoryFile[group]

			infraServers, infraOk := infraCfg.Servers[group]

			if infraOk {
				invGroup.Variables = infraServers.Variables
			}
		}

		for _, server := range servers {
			invGroup.Hosts[server.Name] = map[string]string{
				"ansible_host":      server.PublicNet.IPv4.IP.String(),
				"private_server_ip": server.PrivateNet[0].IP.String(),
				"server_name":       fmt.Sprintf("%s.%s", server.Name, infraCfg.Domain),
			}
			nixConfig.Servers[server.Name] = map[string]map[string]string{
				"vars": {
					"public_ip":   server.PublicNet.IPv4.IP.String(),
					"public_ipv6": server.PublicNet.IPv6.Network.String(),
					"private_ip":  server.PrivateNet[0].IP.String(),
					"server_name": server.Name,
				},
			}

			nixConfig.Hosts[server.PrivateNet[0].IP.String()] = []string{fmt.Sprintf("%s.%s", server.Name, infraCfg.Domain)}
		}
	}

	loadBalancer, err := client.LoadBalancer.All(ctx)

	if err != nil {
		return err
	}

	lbGroup := &InventoryGroup{Hosts: map[string]interface{}{}}
	inventoryFile["loadbalancer"] = lbGroup

	for _, lb := range loadBalancer {
		lbGroup.Hosts[lb.Name] = map[string]string{
			"private_server_ip": lb.PrivateNet[0].IP.String(),
			"server_name":       infraCfg.Domain,
		}
	}

	inventortContent, err := yaml.Marshal(inventoryFile)

	if err != nil {
		return err
	}

	if err := ioutil.WriteFile("inventory.yml", inventortContent, os.ModePerm); err != nil {
		return err
	}

	nixContent, err := json.MarshalIndent(nixConfig, "", "    ")

	if err != nil {
		return err
	}

	if err := ioutil.WriteFile("nix-config.json", nixContent, os.ModePerm); err != nil {
		return err
	}

	log.Infof("Generated ansible inventory file: inventory.yml")

	gitCmd := exec.Command("git", "add", ".")
	gitCmd.Stdout = os.Stdout
	gitCmd.Stderr = os.Stderr

	return gitCmd.Run()
}
