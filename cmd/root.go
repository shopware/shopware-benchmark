package cmd

import (
	"context"
	"fmt"
	hetzner_dns "github.com/panta/go-hetzner-dns"
	"github.com/spf13/cobra"
	"os"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hetznercloud/hcloud-go/hcloud"
	log "github.com/sirupsen/logrus"
)

var rootCmd = &cobra.Command{
	Use:   `benchmark-setup`,
	Short: `Setup server at hetzner`,
	Long:  `This setups our nightly infrastructure at Hetzner to perform performance benchmarks`,
}

func init() {
	rootCmd.PersistentFlags().Bool("verbose", false, "show debug output")
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	if verbose, _ := rootCmd.PersistentFlags().GetBool("verbose"); verbose {
		log.SetLevel(log.TraceLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	log.SetFormatter(&log.TextFormatter{DisableTimestamp: true})
}

func Execute(ctx context.Context) {
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		log.Fatalln(err)
	}
}

func getHetznerCloudClient() (*hcloud.Client, error) {
	if os.Getenv("HETZNER_CLOUD_TOKEN") == "" {
		return nil, fmt.Errorf("environment variable HETZNER_CLOUD_TOKEN is not filled")
	}

	return hcloud.NewClient(hcloud.WithToken(os.Getenv("HETZNER_CLOUD_TOKEN")), hcloud.WithHTTPClient(retryablehttp.NewClient().StandardClient())), nil
}

func getHetznerDnsClient() (*hetzner_dns.Client, error) {
	if os.Getenv("HETZNER_DNS_TOKEN") == "" {
		return nil, fmt.Errorf("environment variable HETZNER_DNS_TOKEN is not filled")
	}

	client := hetzner_dns.Client{ApiKey: os.Getenv("HETZNER_DNS_TOKEN"), HttpClient: retryablehttp.NewClient().StandardClient()}

	return &client, nil
}
