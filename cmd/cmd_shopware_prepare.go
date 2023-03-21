package cmd

import (
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var esInitCmd = &cobra.Command{
	Use:   "prepare",
	Short: "Starts Elasticsearch infrastructure",
	Long:  `Provisions VMs`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ansible := exec.CommandContext(cmd.Context(), "ansible-playbook", "-i", "inventory.yml", "site.yml", "-l", "app", "--tags", "init")
		ansible.Stderr = os.Stderr
		ansible.Stdout = os.Stdout
		ansible.Stdin = os.Stdin

		return ansible.Run()
	},
}

func init() {
	shopwareCmd.AddCommand(esInitCmd)
}
