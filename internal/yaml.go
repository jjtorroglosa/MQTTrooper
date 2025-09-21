package internal

import (
	"log"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

type HttpConfig struct {
	Enabled        bool   `yaml:"enabled"`
	AllowedAddress string `yaml:"allowed_address"`
	Port           int    `yaml:"port"`
	BindAddress    string `yaml:"bind_address"`
}
type MqttConfig struct {
	Enabled                  bool   `yaml:"enabled"`
	Address                  string `yaml:"address"`
	ClientID                 string `yaml:"client_id"`
	User                     string `yaml:"user"`
	Pass                     string `yaml:"pass"`
	Topic                    string `yaml:"topic"`
	ConnectionTimeoutSeconds int    `yaml:"connection_timeout_seconds"`
}

type ExecutorConfig struct {
	Shell  string `yaml:"shell"`
	DryRun bool   `yaml:"dry_run"`
}
type Service struct {
	Name    string `yaml:"name"`
	Command string `yaml:"command"`
}
type ServicesList []Service
type ServicesMap map[string]string

type Config struct {
	ServicesList ServicesList
	Services     ServicesMap    `yaml:"services"`
	Mqtt         MqttConfig     `yaml:"mqtt"`
	Executor     ExecutorConfig `yaml:"executor"`
	Http         HttpConfig     `yaml:"http"`
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
	cfg := Config{
		Mqtt: MqttConfig{
			Enabled:                  false,
			Address:                  "",
			ClientID:                 "",
			User:                     "",
			Pass:                     "",
			Topic:                    "",
			ConnectionTimeoutSeconds: 5,
		},
		Executor: ExecutorConfig{
			Shell:  "/usr/bin/env bash",
			DryRun: false,
		},
		Http: HttpConfig{
			Enabled:        false,
			AllowedAddress: "127.0.0.1",
			Port:           8080,
			BindAddress:    "127.0.0.1",
		},
	}
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
	if cfg.Mqtt.ConnectionTimeoutSeconds <= 0 || cfg.Mqtt.ConnectionTimeoutSeconds > 10 {
		cfg.Mqtt.ConnectionTimeoutSeconds = 3
	}
	sort.Slice(cfg.ServicesList, func(i, j int) bool {
		return cfg.ServicesList[i].Name < cfg.ServicesList[j].Name
	})
	log.Printf("Config file %s loaded: %#v\n", file, cfg)
	return cfg
}
