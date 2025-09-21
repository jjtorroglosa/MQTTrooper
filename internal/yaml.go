package internal

import (
	"log"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

type HttpConfig struct {
	Enabled        bool
	AllowedAddress string
	Port           int
	BindAddress    string
}
type MqttConfig struct {
	Enabled  bool
	Address  string
	ClientID string
	User     string
	Pass     string
	Topic    string
}

type ExecutorConfig struct {
	Shell  string
	DryRun bool
}
type Service struct {
	Name    string
	Command string
}
type ServicesList []Service
type ServicesMap map[string]string

type Config struct {
	ServicesList ServicesList
	Services     ServicesMap
	Mqtt         MqttConfig
	Executor     ExecutorConfig
	Http         HttpConfig
}

func openFile(file string) (string, error) {
	content, err := os.ReadFile(file) // the file is inside the local directory
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func LoadConfigFile(file string) Config {
	data, err := openFile(file)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	var cfg Config
	err = yaml.Unmarshal([]byte(data), &cfg)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	for k, v := range cfg.Services {
		cfg.ServicesList = append(cfg.ServicesList, Service{
			Name:    k,
			Command: v,
		})
	}
	sort.Slice(cfg.ServicesList, func(i, j int) bool {
		return cfg.ServicesList[i].Name < cfg.ServicesList[j].Name
	})
	log.Printf("Config file loaded: %s\n", file)
	log.Printf("Config file loaded: %v\n", cfg)
	return cfg
}
