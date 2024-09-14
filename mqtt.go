package main

import (
	"flag"
	"log"
	"os"
	"strconv"
	"strings"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

func connect(address string, user string, password string, topic string) (MQTT.Client, chan [2]string) {
	cleansess := flag.Bool("clean", false, "Set Clean Session (default false)")
	qos := flag.Int("qos", 0, "The Quality of Service 0,1,2 (default 0)")
	flag.Parse()

	opts := MQTT.NewClientOptions()
	opts.AddBroker(address)
	opts.SetClientID(user)
	opts.SetUsername(user)
	opts.SetPassword(password)
	opts.SetCleanSession(*cleansess)

	choke := make(chan [2]string)

	opts.SetDefaultPublishHandler(func(client MQTT.Client, msg MQTT.Message) {
		choke <- [2]string{msg.Topic(), string(msg.Payload())}
	})

	client := MQTT.NewClient(opts)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	if token := client.Subscribe(topic, byte(*qos), nil); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
		os.Exit(1)
	}
	log.Printf("Connected to broker %s\n", address)
	log.Printf("Listening topic: %s\n", topic)
	return client, choke
}

func parseAndValidateFloat(input string) (bool, float32) {
	f64, err := strconv.ParseFloat(input, 32)
	if err != nil {
		log.Println("   That wasn't a float ¬_¬\n> " + input)
		return false, .0
	}
	f32 := float32(f64)

	if f32 > 0.5 || f32 < 0 {
		log.Println("  That float wasn't great :)\n> " + input)
		return false, .0
	}

	return true, f32
}

func mqttLoop(dryRun bool, cfg Config, choke chan[2] string) {
	for {
		incoming := <-choke
		topic, payload := incoming[0], strings.Split(incoming[1], ":")
		log.Printf("Received in %s -> %s\n", topic, payload)
		var cmd, args string
		if len(payload) == 1 {
			cmd, args = payload[0], ""
		} else if len(payload) == 2 {
			// TODO right now it only accepts floats as argument
			cmd, args = payload[0], payload[1]
			if isFloat, _ := parseAndValidateFloat(args); !isFloat {
				log.Printf("Ignoring cmd as the argument is not a float")
				continue
			}
		} else {
			cmd, args = "", ""
			cmd, args = "", ""
		}
		log.Printf("Cmd: %s Args: %s\n", cmd, args)
		go execute(dryRun, cmd, args, cfg.Services, cfg.Executor.Shell)
	}
}

func listenMqtt(cfg Config, dryRun bool) MQTT.Client {
	// dryRun := flag.Bool("d", false, "Dry run mode")

	// cfg := load(*configFile)
	log.Printf("Available services:")
	for k, v := range cfg.Services {
		log.Printf("  %s => %s\n", k, v)
	}

	client, choke := connect(cfg.Mqtt.Address, cfg.Mqtt.User, cfg.Mqtt.Pass, cfg.Mqtt.Topic)

	log.Printf("Listening messages in topic %s", cfg.Mqtt.Topic)
	go mqttLoop(dryRun, cfg, choke)
	return client
}
