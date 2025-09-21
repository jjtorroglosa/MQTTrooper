package main

import (
	"fmt"
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

	fmt.Println("-------------------------------------------------")
	fmt.Println("                    MQTTrooper                   ")
	fmt.Println("-------------------------------------------------")

	cfg := internal.GetCfg()

	execute := internal.CreateExecutor(cfg.Executor.DryRun, cfg.Executor.Shell, cfg.Services)

	var client mqtt.Client
	if cfg.Mqtt.Enabled {
		client = internal.Connect(cfg.Mqtt.Address, cfg.Mqtt.ClientID, cfg.Mqtt.User, cfg.Mqtt.Pass, cfg.Mqtt.Topic, execute)
	}

	if cfg.Http.Enabled {
		h := internal.NewHttp(cfg)
		go h.ListenHttp(cfg.Http.BindAddress, cfg.Http.Port, cfg.Http.AllowedAddress, execute)
	}

	handleSigterm(client)
}
