package main

import (
	"flag"
	"log"
	"os"
	"strconv"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const qos = 0
const cleansess = false

func subscribe(
	client mqtt.Client,
	topic string,
) chan [2]string {
	choke := make(chan [2]string)
	token := client.Subscribe(topic, byte(qos), func(client mqtt.Client, msg mqtt.Message) {
		choke <- [2]string{msg.Topic(), string(msg.Payload())}
	})
	if token.Wait() && token.Error() != nil {
		log.Println(token.Error())
		os.Exit(1)
	}
	return choke
}

func connect(address string, user string, password string) mqtt.Client {
	flag.Parse()

	opts := mqtt.NewClientOptions()
	opts.AddBroker(address)
	opts.SetClientID(user)
	opts.SetUsername(user)
	opts.SetPassword(password)
	opts.SetCleanSession(cleansess)

	client := mqtt.NewClient(opts)

	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	log.Printf("Connected to broker %s\n", address)

	return client
}

func publish(client mqtt.Client, payload string, topic string) {
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	token := client.Publish(topic, byte(qos), false, payload)

	token.Wait()
	if token.Error() != nil {
		log.Printf("Error publishing payload %s on topic %s. \nError: %s\n", payload, topic, token.Error())
	}
	log.Printf("Message published successfully on [%s] Message: [%s]\n", topic, payload)

	client.Disconnect(250)
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

func Subscribe(client mqtt.Client, topic string, dryRun bool, handler func(string) error) mqtt.Client {
	choke := subscribe(client, topic)

	go func() {
		for {
			incoming := <-choke
			topic, payload := incoming[0], incoming[1]
			log.Printf("Received in %s -> %s\n", topic, payload)
			handler(payload)
		}
	}()

	log.Printf("Listening messages on topic '%s'\n", topic)

	return client
}
