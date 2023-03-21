package cmd

import (
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var infraClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clears the all caches and truncate order table",
	RunE: func(cmd *cobra.Command, _ []string) error {
		ansible := exec.CommandContext(cmd.Context(), "ansible-playbook", "site.yml", "--tags", "clearcache")
		ansible.Stderr = os.Stderr
		ansible.Stdout = os.Stdout
		ansible.Stdin = os.Stdin

		if err := ansible.Run(); err != nil {
			return err
		}

		ansible = exec.CommandContext(cmd.Context(), "ansible-playbook", "site.yml", "--tags", "deleteorder", "-l", "mysql")
		ansible.Stderr = os.Stderr
		ansible.Stdout = os.Stdout
		ansible.Stdin = os.Stdin

		if err := ansible.Run(); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	infraCmd.AddCommand(infraClearCmd)
}
