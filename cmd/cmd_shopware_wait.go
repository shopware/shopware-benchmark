package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"github.com/rgzr/sshtun"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var shopwareCmdWait = &cobra.Command{
	Use:   "wait",
	Short: "Wait for message queue completion",
	RunE: func(cmd *cobra.Command, _ []string) error {
		client, err := getHetznerCloudClient()

		if err != nil {
			return err
		}

		servers, err := getServers(client, cmd.Context())

		if err != nil {
			return err
		}

		if err := waitForQueue(servers, cmd.Context()); err != nil {
			return err
		}

		if err := waitForElasticsearch(servers, cmd.Context()); err != nil {
			return err
		}

		return nil
	},
}

func waitForQueue(servers map[string][]*hcloud.Server, ctx context.Context) error {
	redisServers, ok := servers["redissession"]

	if !ok {
		log.Fatalf("cannot find redis server in infra structure")
	}

	sshTun := sshtun.New(6790, redisServers[0].PublicNet.IPv4.IP.String(), 15672)
	sshTun.SetRemoteHost(redisServers[0].PrivateNet[0].IP.String())
	sshTun.SetDebug(true)

	go func() {
		for {
			if err := sshTun.Start(); err != nil {
				log.Printf("SSH tunnel stopped: %s", err.Error())
				time.Sleep(time.Second)
			}
		}
	}()

	for {
		queueInfo, err := getCurrentQueue(ctx)

		if err != nil {
			return err
		}

		if queueInfo == 0 {
			break
		}

		log.Printf("Messages in the queue: %d", queueInfo)

		time.Sleep(time.Second * 2)
	}

	return nil
}

func waitForElasticsearch(servers map[string][]*hcloud.Server, ctx context.Context) error {
	elasticServers, ok := servers["elastic"]

	if !ok {
		log.Fatalf("cannot find elastic server in infra structure")
	}

	sshTun := sshtun.New(6780, elasticServers[0].PublicNet.IPv4.IP.String(), 9200)
	sshTun.SetDebug(true)

	go func() {
		for {
			if err := sshTun.Start(); err != nil {
				log.Printf("SSH tunnel stopped: %s", err.Error())
				time.Sleep(time.Second)
			}
		}
	}()

	for {
		tasks, err := getCurrentOpenSearchTasks(ctx)

		if err != nil {
			return err
		}

		remaingTasks := 0

		for _, node := range tasks.Nodes {
			for _, task := range node.Tasks {
				if strings.Contains(task.Action, "indices:data/write/bulk") {
					remaingTasks++
				}
			}
		}

		if remaingTasks == 0 {
			log.Infof("All OpenSearch tasks completed")
			break
		}

		log.Infof("Remaining OpenSearch tasks: %d", remaingTasks)

		time.Sleep(time.Second * 2)
	}

	return nil
}

func getCurrentOpenSearchTasks(ctx context.Context) (*OpenSearchTaskResponse, error) {
	r, err := http.NewRequestWithContext(ctx, "POST", "http://localhost:6780/_tasks", nil)

	if err != nil {
		return nil, err
	}

	client := retryablehttp.NewClient()

	resp, err := client.StandardClient().Do(r)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var tasks OpenSearchTaskResponse

	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return nil, err
	}

	return &tasks, nil
}

func getCurrentQueue(ctx context.Context) (int, error) {
	r, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:6790/api/queues", nil)
	r.SetBasicAuth("guest", "guest")

	if err != nil {
		return 0, err
	}

	client := retryablehttp.NewClient()

	resp, err := client.StandardClient().Do(r)

	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	var queues []struct {
		Name     string `json:"name"`
		Messages int    `json:"messages"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&queues); err != nil {
		return 0, err
	}

	for _, queue := range queues {
		if queue.Name == "messages" {
			return queue.Messages, nil
		}
	}

	return 0, fmt.Errorf("cannot find messages queue")
}

func init() {
	shopwareCmd.AddCommand(shopwareCmdWait)
}

type AMQPQueues []struct {
	Name     string `json:"name"`
	Messages int    `json:"messages"`
}

type OpenSearchTaskResponse struct {
	Nodes map[string]OpenSearchNode `json:"nodes"`
}
type Attributes struct {
	ShardIndexingPressureEnabled string `json:"shard_indexing_pressure_enabled"`
}

type OpenSearchTask struct {
	Node               string `json:"node"`
	ID                 int    `json:"id"`
	Type               string `json:"type"`
	Action             string `json:"action"`
	StartTimeInMillis  int64  `json:"start_time_in_millis"`
	RunningTimeInNanos int    `json:"running_time_in_nanos"`
	Cancellable        bool   `json:"cancellable"`
	ParentTaskID       string `json:"parent_task_id"`
}

type OpenSearchNode struct {
	Name             string                    `json:"name"`
	TransportAddress string                    `json:"transport_address"`
	Host             string                    `json:"host"`
	IP               string                    `json:"ip"`
	Roles            []string                  `json:"roles"`
	Attributes       Attributes                `json:"attributes"`
	Tasks            map[string]OpenSearchTask `json:"tasks"`
}
