package cmd

import (
	"github.com/spf13/cobra"
)

var infraCmd = &cobra.Command{
	Use:   "infra",
	Short: "Control infrastructure",
}

func init() {
	rootCmd.AddCommand(infraCmd)
}
