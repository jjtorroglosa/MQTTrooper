package main

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type HttpConfig struct {
	Enabled        bool
	AllowedAddress string
	Port           int
	BindAddress    string
}
type MqttConfig struct {
	Enabled bool
	Address string
	User    string
	Pass    string
	Topic   string
	Payload string
	Publish bool
}
type ExecutorConfig struct {
	Shell  string
	DryRun bool
}
type ServicesMap map[string]string
type Config struct {
	Services ServicesMap
	Mqtt     MqttConfig
	Executor ExecutorConfig
	Http     HttpConfig
}

func openFile(file string) (string, error) {
	content, err := os.ReadFile(file) // the file is inside the local directory
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func load(file string) Config {
	data, err := openFile(file)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	var cfg Config
	err = yaml.Unmarshal([]byte(data), &cfg)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	log.Printf("Config file loaded: %s\n", file)
	return cfg
}
