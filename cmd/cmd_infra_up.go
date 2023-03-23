package cmd

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/hetznercloud/hcloud-go/hcloud"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var infraUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Starts our infrastructure at Hetzner Cloud",
	Long:  `Provisions VMs`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := prepareProject(); err != nil {
			return err
		}

		client, err := getHetznerCloudClient()

		if err != nil {
			return err
		}

		infraCfg, err := loadInfraConfig()

		if err != nil {
			return err
		}

		networks, err := getNetworks(client, cmd.Context(), infraCfg)
		if err != nil {
			return err
		}

		sshKeys, err := getSSHKeys(client, cmd.Context(), infraCfg)
		if err != nil {
			return err
		}

		groups, err := getServers(client, cmd.Context())

		if err != nil {
			return err
		}

		images, _, err := client.Image.List(cmd.Context(), hcloud.ImageListOpts{IncludeDeprecated: false, Type: []hcloud.ImageType{hcloud.ImageTypeSnapshot}})
		if err != nil {
			return err
		}

		mysqlSnapshot, _ := cmd.PersistentFlags().GetString("run-env")
		mysqlSnapshotId := -1

		for _, image := range images {
			if image.Description == mysqlSnapshot {
				mysqlSnapshotId = image.ID
			}
		}

		if mysqlSnapshotId == -1 {
			log.Fatalf("cannot find MySQL snapshot by name %s", mysqlSnapshot)
		}

		createdServer := false

		for configGroup, serverCfg := range infraCfg.Servers {
			group, ok := groups[configGroup]
			image := serverCfg.Image

			if image == 0 {
				image = infraCfg.Image
			}

			if configGroup == "mysql" {
				image = mysqlSnapshotId
			}

			opts := CreateServerOpts{
				infraCfg:   infraCfg,
				name:       configGroup,
				serverType: serverCfg.Type,
				networks:   networks,
				keys:       sshKeys,
				image:      image,
			}

			if !ok {
				createdServer = true
				err := createServer(client, cmd.Context(), opts, 1, serverCfg.Amount)
				if err != nil {
					return err
				}
				continue
			}

			groupLength := len(group)

			for _, externalServer := range group {
				if externalServer.ServerType.Name != serverCfg.Type {
					log.Warningf("Server %s (id: %d) is currently: %s, but expected %s.", externalServer.Name, externalServer.ID, externalServer.ServerType.Name, serverCfg.Type)
				}
			}

			if groupLength != serverCfg.Amount {
				if groupLength > serverCfg.Amount {
					for i, server := range group {
						if i > serverCfg.Amount-1 {
							if _, err := client.Server.Delete(cmd.Context(), server); err != nil {
								return err
							}

							log.Infof("Server %s has been deleted", server.Name)
						}
					}
				} else {
					createdServer = true
					err := createServer(client, cmd.Context(), opts, groupLength+1, serverCfg.Amount-groupLength)
					if err != nil {
						return err
					}
				}
			}
		}

		log.Infof("Created all servers. Waiting now they come online")

		for {
			groups, err = getServers(client, cmd.Context())

			if err != nil {
				break
			}

			allServersOnline := true

			for _, servers := range groups {
				for _, server := range servers {
					if server.Status != hcloud.ServerStatusRunning {
						allServersOnline = false
						log.Infof("Server %s is still offline", server.Name)
					}
				}
			}

			if allServersOnline {
				break
			}

			log.Infof("Checking again in 20 seconds")
			time.Sleep(20 * time.Second)
		}

		if err := generateAnsibleInventory(infraCfg, client, cmd.Context()); err != nil {
			return err
		}

		if err := updateSshConfig(client, cmd.Context(), infraCfg); err != nil {
			return err
		}

		if createdServer {
			log.Infof("Waiting 1 minutes to give Servers the chance to get up")
			time.Sleep(time.Minute)
		}

		cmdCtx := cmd.Context()
		if err := runColmena(cmdCtx); err != nil {
			return err
		}

		appPlayBook := exec.CommandContext(cmdCtx, "ansible-playbook", "-i", "inventory.yml", "site.yml", "-l", "app")
		appPlayBook.Stderr = os.Stderr
		appPlayBook.Stdout = os.Stdout
		appPlayBook.Stdin = os.Stdin

		if err := appPlayBook.Run(); err != nil {
			return err
		}

		if appPlayBook.ProcessState.ExitCode() != 0 {
			return fmt.Errorf("app playbook failed")
		}

		locustPlayBook := exec.CommandContext(cmdCtx, "ansible-playbook", "-i", "inventory.yml", "site.yml", "-l", "locust")
		locustPlayBook.Stderr = os.Stderr
		locustPlayBook.Stdout = os.Stdout
		locustPlayBook.Stdin = os.Stdin

		if err := locustPlayBook.Run(); err != nil {
			return err
		}

		if locustPlayBook.ProcessState.ExitCode() != 0 {
			return fmt.Errorf("locust playbook failed")
		}

		return nil
	},
}

func runColmena(ctx context.Context) error {
	target := []string{"apply"}

	if runtime.GOOS == "darwin" {
		target = append(target, "--build-on-target")
	}

	cmd := exec.CommandContext(ctx, "colmena", target...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		log.Errorf("Coleman failed %s, error: %v", err)
		return err
	}

	if cmd.ProcessState.ExitCode() != 0 {
		log.Errorf("Colmena failed exited with %d", cmd.ProcessState.ExitCode())
		return fmt.Errorf("colmena failed")
	}

	return nil
}

func getSSHKeys(client *hcloud.Client, ctx context.Context, cfg *InfraConfig) ([]*hcloud.SSHKey, error) {
	keys, _, err := client.SSHKey.List(ctx, hcloud.SSHKeyListOpts{})

	if err != nil {
		return nil, err
	}

	attachedKeys := make([]*hcloud.SSHKey, 0)

	for keyName, _ := range cfg.SSHKey {
		var foundKey *hcloud.SSHKey

		for _, key := range keys {
			if key.Name == keyName {
				foundKey = key
				break
			}
		}

		if foundKey == nil {
			return nil, fmt.Errorf("cannot find requested key %s", keyName)
		}

		attachedKeys = append(attachedKeys, foundKey)
	}

	return attachedKeys, nil
}

func getNetworks(client *hcloud.Client, ctx context.Context, config *InfraConfig) ([]*hcloud.Network, error) {
	networks, _, err := client.Network.List(ctx, hcloud.NetworkListOpts{})

	if err != nil {
		return nil, err
	}

	attachedNetworks := make([]*hcloud.Network, 0)

	for networkName, _ := range config.Networks {
		var foundNetwork *hcloud.Network

		for _, network := range networks {
			if network.Name == networkName {
				foundNetwork = network
				break
			}
		}

		if foundNetwork == nil {
			return nil, fmt.Errorf("cannot find requested network %s", networkName)
		}

		attachedNetworks = append(attachedNetworks, foundNetwork)
	}

	return attachedNetworks, nil
}

type CreateServerOpts struct {
	infraCfg   *InfraConfig
	name       string
	serverType string
	networks   []*hcloud.Network
	keys       []*hcloud.SSHKey
	image      int
}

func createServer(client *hcloud.Client, ctx context.Context, opts CreateServerOpts, startIndex int, amount int) error {
	start := true
	for i := 0; i < amount; i++ {
		var err error

		// try to spawn at multiple locations
		for _, location := range opts.infraCfg.Location {
			_, _, err = client.Server.Create(ctx, hcloud.ServerCreateOpts{
				Name:             fmt.Sprintf("%s-%d", opts.name, startIndex),
				ServerType:       &hcloud.ServerType{Name: opts.serverType},
				Datacenter:       &hcloud.Datacenter{Name: location},
				StartAfterCreate: &start,
				Labels:           map[string]string{"group": opts.name},
				Image:            &hcloud.Image{ID: opts.image},
				Networks:         opts.networks,
				SSHKeys:          opts.keys,
			})

			if err == nil {
				break
			}
		}

		if err != nil {
			return err
		}

		log.Infof("Created server with name %s and type %s", fmt.Sprintf("%s-%d", opts.name, startIndex), opts.serverType)

		startIndex++
	}

	return nil
}

func prepareProject() error {
	curDir, _ := os.Getwd()
	jwtFolder := filepath.Join(curDir, "roles", "app", "files", "jwt")

	if _, err := os.Stat(jwtFolder); os.IsNotExist(err) {
		os.MkdirAll(jwtFolder, os.ModePerm)

		publicKey, privateKey, err := generatePrivatePublicKey(2048)
		if err != nil {
			return err
		}

		if err := os.WriteFile(filepath.Join(jwtFolder, "private.pem"), []byte(privateKey), 0644); err != nil {
			return err
		}

		if err := os.WriteFile(filepath.Join(jwtFolder, "public.pem"), []byte(publicKey), 0644); err != nil {
			return err
		}
	}

	return nil
}

func generatePrivatePublicKey(keyLength int) (string, string, error) {
	rsaKey, err := generatePrivateKey(keyLength)
	if err != nil {
		return "", "", err
	}

	rsaPrivKey := encodePrivateKeyToPEM(rsaKey)
	rsaPubKey, err := generatePublicKey(rsaKey)
	if err != nil {
		return "", "", fmt.Errorf("unable to generate public key: %v", err)
	}

	return rsaPubKey, rsaPrivKey, nil
}

// generatePrivateKey creates a RSA Private Key of specified byte size.
func generatePrivateKey(bitSize int) (*rsa.PrivateKey, error) {
	// Private Key generation
	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return nil, err
	}

	// Validate Private Key
	err = privateKey.Validate()
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

func encodePrivateKeyToPEM(privateKey *rsa.PrivateKey) string {
	key := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	return string(pem.EncodeToMemory(key))
}

func generatePublicKey(key *rsa.PrivateKey) (string, error) {
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 180),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return "", err
	}

	pubPem := &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}

	return string(pem.EncodeToMemory(pubPem)), nil
}

func init() {
	infraUpCmd.PersistentFlags().String("run-env", "demo-data", "Hetzner Cloud Snapshot Name")
	infraCmd.AddCommand(infraUpCmd)
}
