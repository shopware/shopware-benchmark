package cmd

import (
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

type InfraConfig struct {
	Name     string                       `yaml:"name"`
	Domain   string                       `yaml:"domain"`
	Location []string                     `yaml:"location"`
	Image    int                          `yaml:"image"`
	Servers  map[string]InfraConfigServer `yaml:"servers"`
	Networks map[string]map[string]string `yaml:"networks"`
	SSHKey   map[string]map[string]string `yaml:"keys"`
}

type InfraConfigServer struct {
	Amount     int                    `yaml:"amount"`
	Type       string                 `yaml:"type"`
	Variables  map[string]interface{} `yaml:"ansible_vars"`
	Persistent bool                   `yaml:"persistent"`
	Image      int                    `yaml:"image"`
}

func loadInfraConfig() (*InfraConfig, error) {
	content, err := ioutil.ReadFile("servers.yml")

	if err != nil {
		return nil, err
	}

	cfg := InfraConfig{}

	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
