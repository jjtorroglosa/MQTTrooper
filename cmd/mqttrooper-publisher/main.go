// mqttrooper-publisher is a simple mqtt publisher for testing purposes.
package main

import (
	"flag"
	"log"
	"mqttrooper/internal"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func main() {

	log.Println("-------------------------------------------------")
	log.Println("                    MQTTrooper                   ")
	log.Println("-------------------------------------------------")

	var payload = flag.String("message", "", "The message to publish")
	var configFile = flag.String("c", "config.yaml", "The path to the config.yaml file")
	var clientID = flag.String("clientid", "publisher", "The client id")
	flag.Parse()

	var client mqtt.Client
	cfg, err := internal.LoadConfigFile(*configFile)
	if err != nil {
		log.Fatalf("Error loading config file: %v", err)
	}

	cfg.Mqtt.ClientID = *clientID
	client = internal.Connect(cfg.Mqtt.Address, cfg.Mqtt.ClientID, cfg.Mqtt.User, cfg.Mqtt.Pass, cfg.Mqtt.Topic, nil)

	if *payload == "" {
		log.Fatalln("Empty payload. You need to provide a valid payload to publish")
	}

	internal.Publish(client, *payload, cfg.Mqtt.Topic)
	client.Disconnect(2000)
	os.Exit(0)
}
