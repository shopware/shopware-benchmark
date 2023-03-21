package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var generatePhpStormCmd = &cobra.Command{
	Use:   "phpstorm",
	Short: "Generate ansible inventory from existing servers",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getHetznerCloudClient()

		if err != nil {
			return err
		}

		groups, err := getServers(client, cmd.Context())

		if err != nil {
			return err
		}

		sshConfig := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<project version=\"4\">\n  <component name=\"SshConfigs\">\n    <configs>"
		endSsh := "</configs>\n  </component></project>"

		webServerConfig := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<project version=\"4\">\n  <component name=\"WebServers\">\n<option name=\"servers\">\n"
		endWebServer := "  </component>\n</project>"

		for groupName, servers := range groups {
			root := "/root"

			if groupName == "app" {
				root = "/var/www/html"
			}

			for _, server := range servers {
				sshConfigId := uuid.New().String()
				sshConfig += fmt.Sprintf(
					"<sshConfig authType=\"OPEN_SSH\" host=\"%s\" id=\"%s\" port=\"22\" nameFormat=\"DESCRIPTIVE\" username=\"root\" useOpenSSHConfig=\"true\" />\n",
					fmt.Sprintf("%s.sw-bench.de", server.Name),
					sshConfigId,
				)

				webServerConfig += fmt.Sprintf(
					"<webServer id=\"%s\" name=\"%s\">\n        <fileTransfer rootFolder=\"%s\" accessType=\"SFTP\" host=\"%s\" port=\"22\" sshConfigId=\"%s\" sshConfig=\"root@benchmark-%s:22 agent\" authAgent=\"true\">\n          <advancedOptions>\n            <advancedOptions dataProtectionLevel=\"Private\" keepAliveTimeout=\"0\" passiveMode=\"true\" shareSSLContext=\"true\" />\n          </advancedOptions>\n        </fileTransfer>\n      </webServer>\n",
					uuid.New().String(),
					server.Name,
					root,
					server.Name,
					sshConfigId,
					server.Name,
				)
			}
		}

		webServerConfig += "</option>"
		webServerConfig += "<groups>"

		for groupName, servers := range groups {
			webServerConfig += fmt.Sprintf("<group>\n        <name>%s</name>\n        <servers>", groupName)

			for _, server := range servers {
				webServerConfig += fmt.Sprintf("<name value=\"%s\" />\n", server.Name)
			}

			webServerConfig += "</servers>\n      </group>\n"
		}

		webServerConfig += "</groups>"

		ioutil.WriteFile(fmt.Sprintf("%s/.idea/sshConfigs.xml", args[0]), []byte(sshConfig+endSsh), os.ModePerm)
		ioutil.WriteFile(fmt.Sprintf("%s/.idea/webServers.xml", args[0]), []byte(webServerConfig+endWebServer), os.ModePerm)

		return nil
	},
}

func init() {
	generateCmd.AddCommand(generatePhpStormCmd)
}
