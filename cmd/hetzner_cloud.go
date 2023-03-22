package cmd

import (
	"context"

	"github.com/hetznercloud/hcloud-go/hcloud"
	log "github.com/sirupsen/logrus"
)

func getServers(client *hcloud.Client, ctx context.Context) (map[string][]*hcloud.Server, error) {
	filtered := make(map[string][]*hcloud.Server)

	log.Debugf("Fetching servers form hetzner api")
	servers, _, err := client.Server.List(ctx, hcloud.ServerListOpts{})

	if err != nil {
		return filtered, err
	}

	for _, server := range servers {
		if len(server.Labels) == 0 {
			log.Warningf("Skipping server %s (id: %d) as it has no labels set", server.Name, server.ID)
			continue
		}

		group, ok := server.Labels["group"]
		if !ok {
			log.Warningf("Skipping server %s (id: %d) as it has no group label", server.Name, server.ID)
			continue
		}

		_, ok = filtered[group]

		if !ok {
			filtered[group] = make([]*hcloud.Server, 0)
		}

		filtered[group] = append(filtered[group], server)
	}

	return filtered, nil
}
