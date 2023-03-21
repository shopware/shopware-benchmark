package cmd

import (
	"context"
	"fmt"

	"github.com/hetznercloud/hcloud-go/hcloud"
	hetzner_dns "github.com/panta/go-hetzner-dns"
	log "github.com/sirupsen/logrus"
)

func updateDns(client *hcloud.Client, dnsClient *hetzner_dns.Client, infraCfg *InfraConfig, ctx context.Context) error {
	groups, err := getServers(client, ctx)
	if err != nil {
		return err
	}

	zones, err := dnsClient.GetZones(ctx, "", "", 0, 20)
	if err != nil {
		return err
	}

	var foundZone *hetzner_dns.Zone

	for _, zone := range zones.Zones {
		if zone.Name == infraCfg.Domain {
			foundZone = &zone
			break
		}
	}

	if foundZone == nil {
		return fmt.Errorf("cannot find dns zone")
	}

	records, err := dnsClient.GetRecords(ctx, foundZone.ID, 1, 100)
	if err != nil {
		return err
	}

	// Add or update records
	for _, servers := range groups {
		for _, server := range servers {
			name := fmt.Sprintf("%s.%s", server.Name, infraCfg.Domain)

			var foundRecord *hetzner_dns.Record
			for _, record := range records.Records {
				if record.Name == server.Name {
					foundRecord = &record
					break
				}
			}

			dnsRequest := hetzner_dns.RecordRequest{
				ZoneID: foundZone.ID,
				Type:   "A",
				Name:   server.Name,
				Value:  server.PublicNet.IPv4.IP.String(),
				TTL:    300,
			}

			if foundRecord == nil {
				_, err := dnsClient.CreateRecord(ctx, dnsRequest)

				if err != nil {
					return err
				}

				log.Infof("ADD: %s %s %s", name, dnsRequest.Type, dnsRequest.Value)
			} else {
				if foundRecord.Value != server.PublicNet.IPv4.IP.String() {

					dnsRequest.ID = foundRecord.ID
					_, err := dnsClient.UpdateRecord(ctx, dnsRequest)
					if err != nil {
						return err
					}

					log.Infof("UPD: %s %s %s", name, dnsRequest.Type, dnsRequest.Value)
				} else {
					log.Infof("OK: %s %s %s", name, dnsRequest.Type, dnsRequest.Value)
				}
			}
		}
	}

	// Delete records
	for _, record := range records.Records {
		if record.Type == "NS" || record.Type == "SOA" || record.Name == "@" || record.Name == "paas-grid" || record.Name == "paas-dedicated" || record.Name == "md" || record.Name == "grafana" || record.Name == "assets" {
			continue
		}

		found := false

		for _, servers := range groups {
			for _, server := range servers {
				if server.Name == record.Name {
					found = true
				}
			}
		}

		if !found {
			err := dnsClient.DeleteRecord(ctx, record.ID)

			if err != nil {
				return err
			}

			log.Infof("DEL: %s %s %s", record.Name, record.Type, record.Value)
		}
	}
	return nil
}
