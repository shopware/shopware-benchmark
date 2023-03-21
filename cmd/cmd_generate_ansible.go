package cmd

import (
	"github.com/spf13/cobra"
)

var generateAnsibleCmd = &cobra.Command{
	Use:   "ansible",
	Short: "Generate ansible inventory from existing servers",
	RunE: func(cmd *cobra.Command, _ []string) error {
		client, err := getHetznerCloudClient()

		if err != nil {
			return err
		}

		infraCfg, err := loadInfraConfig()

		if err != nil {
			return err
		}

		return generateAnsibleInventory(infraCfg, client, cmd.Context())
	},
}

func init() {
	generateCmd.AddCommand(generateAnsibleCmd)
}
