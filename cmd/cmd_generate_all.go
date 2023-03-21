package cmd

import (
	"github.com/spf13/cobra"
)

var generateAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Generate all files",
	RunE: func(cmd *cobra.Command, _ []string) error {
		client, err := getHetznerCloudClient()

		if err != nil {
			return err
		}

		infraCfg, err := loadInfraConfig()

		if err != nil {
			return err
		}

		if err := generateAnsibleInventory(infraCfg, client, cmd.Context()); err != nil {
			return err
		}

		return updateSshConfig(client, cmd.Context(), infraCfg)
	},
}

func init() {
	generateCmd.AddCommand(generateAllCmd)
}
