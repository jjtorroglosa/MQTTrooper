package main

import (
	"flag"
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

	var payload = flag.String("message", "", "The message to publish")
	var configFile = flag.String("c", "config.yaml", "The path to the config.yaml file")
	var clientId = flag.String("clientid", "publisher", "The client id")
	flag.Parse()

	var client mqtt.Client
	cfg := internal.LoadConfigFile(*configFile)
	cfg.Mqtt.ClientID = *clientId
	client = internal.Connect(cfg.Mqtt.Address, cfg.Mqtt.ClientID, cfg.Mqtt.User, cfg.Mqtt.Pass, cfg.Mqtt.Topic, nil)

	if *payload == "" {
		log.Fatalln("Empty payload. You need to provide a valid payload to publish")
	}

	internal.Publish(client, *payload, cfg.Mqtt.Topic)
	client.Disconnect(2000)
	os.Exit(0)
}
