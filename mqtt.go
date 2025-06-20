package main

import (
	"flag"
	"log"
	"os"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const qos = 0
const cleansess = false

func connect(address string, user string, password string, topic string, execute Executor) mqtt.Client {
	flag.Parse()

	mqtt.ERROR = log.New(os.Stdout, "[ERROR] ", 0)
	mqtt.CRITICAL = log.New(os.Stdout, "[CRIT] ", 0)
	mqtt.WARN = log.New(os.Stdout, "[WARN]  ", 0)
	// mqtt.DEBUG = log.New(os.Stdout, "[DEBUG] ", 0)

	opts := mqtt.NewClientOptions()
	opts.AddBroker(address)
	opts.SetClientID(user)
	opts.SetUsername(user)
	opts.SetPassword(password)
	opts.SetCleanSession(cleansess)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.KeepAlive = 10

	opts.SetOnConnectHandler(func(c mqtt.Client) {
		log.Println("Connected to broker.")
		subscribe(c, topic, execute)
	})

	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		log.Println("Connection lost:", err)
	})

	opts.OnReconnecting = func(mqtt.Client, *mqtt.ClientOptions) {
		log.Println("attempting to reconnect")
	}

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

func subscribe(client mqtt.Client, topic string, handler func(string) error) mqtt.Client {
	token := client.Subscribe(topic, qos, func(client mqtt.Client, msg mqtt.Message) {
		message := string(msg.Payload())
		handler(message)
	})
	token.Wait()
	if err := token.Error(); err != nil {
		log.Println("Subscribe error:", err)
	}

	log.Printf("Listening messages on topic '%s'\n", topic)

	return client
}
