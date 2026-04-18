package main

import (
	"flag"
	"log"
	"mqttrooper/internal"
	"os"
	"os/signal"
	"syscall"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func handleSigterm(client mqtt.Client) {
	sigtermChan := make(chan os.Signal, 1)
	signal.Notify(sigtermChan, os.Interrupt, syscall.SIGTERM)

	<-sigtermChan
	log.Println("MQTT: SIGTERM received. Exiting...")
	if client != nil {
		client.Disconnect(250)
	}
	log.Println("Client Disconnected")
	os.Exit(1)
}

func main() {
	cfg, err := internal.GetCfg()
	if err != nil {
		log.Fatalf("Error reading the csrf key: %v", err)
	}
	cmd := flag.Arg(0)
	if cmd == "" || cmd == "serve" {
		serve(cfg)
		os.Exit(0)
	}
	if cmd == "dump-plist" {
		generatePlist(cfg)
		os.Exit(0)
	}
	if cmd == "dump-systemd-service" {
		generateSystemdService(cfg)
		os.Exit(0)
	}

	log.Fatalf("Unknown cmd: %s", cmd)
}

func serve(cfg *internal.Config) {
	log.Println("-------------------------------------------------")
	log.Println("                    MQTTrooper                   ")
	log.Println("-------------------------------------------------")

	log.Printf("Config loaded: %#v\n", cfg)

	if cfg.Executor.DryRun {
		log.Println("** Dry run mode **")
	}

	execute := internal.CreateExecutor(cfg.Executor.DryRun, cfg.Executor.Shell, cfg.Services)
	log.Println("Config created")

	var client mqtt.Client
	if cfg.Mqtt.Enabled {
		client = internal.Connect(cfg.Mqtt.Address, cfg.Mqtt.ClientID, cfg.Mqtt.User, cfg.Mqtt.Pass, cfg.Mqtt.Topic, execute)
		if cfg.Mqtt.Discovery.Enabled {
			if err := internal.PublishDiscovery(client, cfg); err != nil {
				log.Printf("Discovery publish error: %v", err)
			}
		}
	} else if cfg.Mqtt.Discovery.Enabled {
		log.Println("mqtt.discovery.enabled=true but mqtt.enabled=false; skipping discovery")
	}
	log.Println("Connected to mqtt")

	if cfg.Http.Enabled {
		h := internal.NewHttp(cfg)
		go h.ListenHttp(&cfg.Http, execute)
	}

	handleSigterm(client)
}
