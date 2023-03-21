package cmd

import (
	"github.com/spf13/cobra"
)

var shopwareCmd = &cobra.Command{
	Use:   "shopware",
	Short: "Shopware commands",
}

func init() {
	rootCmd.AddCommand(shopwareCmd)
}
