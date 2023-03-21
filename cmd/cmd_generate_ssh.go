package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/hetznercloud/hcloud-go/hcloud"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var generateSshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "Generate ssh config for all hosts",
	RunE: func(cmd *cobra.Command, _ []string) error {
		infraCfg, err := loadInfraConfig()
		if err != nil {
			return err
		}

		client, err := getHetznerCloudClient()

		if err != nil {
			return err
		}

		return updateSshConfig(client, cmd.Context(), infraCfg)
	},
}

func init() {
	generateCmd.AddCommand(generateSshCmd)
}

func updateSshConfig(client *hcloud.Client, ctx context.Context, infraCfg *InfraConfig) error {
	groups, err := getServers(client, ctx)

	if err != nil {
		return err
	}

	hostfile := ""

	serverIps := make([]string, 0)

	for _, servers := range groups {
		for _, server := range servers {
			hostfile += fmt.Sprintf("Host %s-%s\n", infraCfg.Name, server.Name)
			hostfile += fmt.Sprintf("HostName %s.%s\n", server.Name, infraCfg.Domain)
			hostfile += fmt.Sprintf("User root\n")
			hostfile += fmt.Sprintf("Port 22\n\n")

			serverIps = append(serverIps, server.PublicNet.IPv4.IP.String())
		}
	}

	homeDir, err := os.UserHomeDir()

	if err != nil {
		return err
	}

	configFolder := fmt.Sprintf("%s/.ssh/groups", homeDir)

	if _, stat := os.Stat(configFolder); os.IsNotExist(stat) {
		if err := os.MkdirAll(configFolder, os.ModePerm); err != nil {
			return err
		}
	}

	configFile := fmt.Sprintf("%s/%s", configFolder, infraCfg.Name)

	if err := ioutil.WriteFile(configFile, []byte(hostfile), os.ModePerm); err != nil {
		return err
	}

	log.Infof("Generated ssh config at %s. Don't forget to include it to your ~/.ssh/config with \"Include groups/*\"\n", configFile)

	knownHostFile := fmt.Sprintf("%s/.ssh/known_hosts", homeDir)

	if _, err := os.Stat(knownHostFile); err == nil {
		content, err := ioutil.ReadFile(knownHostFile)

		if err != nil {
			return err
		}

		newLines := make([]string, 0)
		lines := strings.Split(string(content), "\n")

		for _, line := range lines {
			if strings.Contains(line, infraCfg.Domain) {
				continue
			}

			foundIp := false
			for _, ip := range serverIps {
				if strings.Contains(line, ip) {
					foundIp = true
					break
				}
			}

			if foundIp {
				continue
			}

			newLines = append(newLines, line)
		}

		if err := ioutil.WriteFile(knownHostFile, []byte(strings.Join(newLines, "\n")), os.ModePerm); err != nil {
			return err
		}

		log.Infof("Filtered out deleted hosts of your ~/.ssh/known_hosts file")
	}

	return nil
}
