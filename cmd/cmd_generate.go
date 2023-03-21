package cmd

import (
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate configurations",
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
