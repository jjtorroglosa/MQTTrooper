package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type MqttConfig struct {
	Address string
	User    string
	Pass    string
	Topic   string
}
type ExecutorConfig struct {
	Shell string
}
type Config struct {
	Services map[string]string
	Mqtt     MqttConfig
	Executor ExecutorConfig
}

func GetFlag() *string {
	return flag.String("c", "config.yaml", "The path to the config.yaml file")
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
	fmt.Println("-- Result --")
	return cfg
}
