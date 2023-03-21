package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var infraDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Destroys all running servers",
	RunE: func(cmd *cobra.Command, _ []string) error {
		client, err := getHetznerCloudClient()

		if err != nil {
			return err
		}

		infraCfg, err := loadInfraConfig()

		if err != nil {
			return err
		}

		groups, err := getServers(client, cmd.Context())

		if err != nil {
			return err
		}

		for configGroup, serverCfg := range infraCfg.Servers {
			group, ok := groups[configGroup]

			if !ok {
				continue
			}

			if serverCfg.Persistent {
				continue
			}

			for _, server := range group {
				if _, err := client.Server.Delete(cmd.Context(), server); err != nil {
					return err
				}

				log.Infof("Deleted server %s", server.Name)
			}
		}

		dnsClient, err := getHetznerDnsClient()

		if err != nil {
			return err
		}

		return updateDns(client, dnsClient, infraCfg, cmd.Context())
	},
}

func init() {
	infraCmd.AddCommand(infraDownCmd)
}
