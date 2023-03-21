package cmd

import (
	"github.com/spf13/cobra"
)

var infraDns = &cobra.Command{
	Use:   "dns",
	Short: "Sets DNS records for all running servers",
	RunE: func(cmd *cobra.Command, _ []string) error {
		client, err := getHetznerCloudClient()

		if err != nil {
			return err
		}

		dnsClient, err := getHetznerDnsClient()

		if err != nil {
			return err
		}

		infraCfg, err := loadInfraConfig()
		if err != nil {
			return err
		}

		if err := updateDns(client, dnsClient, infraCfg, cmd.Context()); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	infraCmd.AddCommand(infraDns)
}
