package internal

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"go.yaml.in/yaml/v4"
)

type HttpConfig struct {
	Enabled          bool   `yaml:"enabled"`
	AllowedAddress   string `yaml:"allowed_address"`
	Port             int    `yaml:"port"`
	BindAddress      string `yaml:"bind_address"`
	CsrfSecretBase64 string `yaml:"csrf_secret"`
	CsrfSecret       []byte `yaml:"-"`
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

type DaemonConfig struct {
	Cwd           string `yaml:"cwd"`
	EnvPath       string `yaml:"env_path"`
	LogFilePath   string `yaml:"log_file_path"`
	ErrorFilePath string `yaml:"error_file_path"`
	MacID         string `yaml:"mac_id"`
}

type Config struct {
	ConfigPath   string
	ServicesList ServicesList
	Services     ServicesMap    `yaml:"services"`
	Mqtt         MqttConfig     `yaml:"mqtt"`
	Executor     ExecutorConfig `yaml:"executor"`
	Http         HttpConfig     `yaml:"http"`
	Daemon       DaemonConfig   `yaml:"daemon"`
}

func openFile(file string) (string, error) {
	content, err := os.ReadFile(file) // the file is inside the local directory
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func LoadConfigFile(file string) (*Config, error) {
	data, err := openFile(file)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	file, err = filepath.Abs(file)
	if err != nil {
		log.Fatalf(
			"error getting the path of the config file %s: %v",
			file,
			err,
		)
	}
	cfg := Config{
		ConfigPath: file,
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
		Daemon: DaemonConfig{
			Cwd:           "",
			EnvPath:       "",
			LogFilePath:   file,
			ErrorFilePath: file,
			MacID:         "com.jtorr.mqttrooper",
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
	bytes, err := base64.StdEncoding.DecodeString(cfg.Http.CsrfSecretBase64)
	if err != nil {
		return nil, fmt.Errorf("error reading the csrf secret: %v", err)
	}
	cfg.Http.CsrfSecret = bytes
	sort.Slice(cfg.ServicesList, func(i, j int) bool {
		return cfg.ServicesList[i].Name < cfg.ServicesList[j].Name
	})
	return &cfg, nil
}
