package internal

import (
	"log"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const qos = 0
const cleansess = false

// Connects to the MQTT broker.
func Connect(address string, clientId string, user string, password string, topic string, execute Executor, onConnect func(mqtt.Client) error) mqtt.Client {
	mqtt.ERROR = log.New(os.Stdout, "[ERROR] ", 0)
	mqtt.CRITICAL = log.New(os.Stdout, "[CRIT] ", 0)
	mqtt.WARN = log.New(os.Stdout, "[WARN]  ", 0)
	// mqtt.DEBUG = log.New(os.Stdout, "[DEBUG] ", 0)

	opts := mqtt.NewClientOptions()
	opts.AddBroker(address)
	opts.SetClientID(clientId)
	opts.SetUsername(user)
	opts.SetPassword(password)
	opts.SetCleanSession(cleansess)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.KeepAlive = 10

	if execute != nil || onConnect != nil {
		opts.SetOnConnectHandler(buildOnConnectHandler(topic, execute, onConnect))

		opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
			log.Println("Connection lost:", err)
		})

		opts.OnReconnecting = func(mqtt.Client, *mqtt.ClientOptions) {
			log.Println("attempting to reconnect")
		}
	}

	client := mqtt.NewClient(opts)

	log.Printf("...Connecting to broker %s\n", address)
	token := client.Connect()
	ok := token.WaitTimeout(3 * time.Second)
	if !ok {
		panic("❌ Connection timeout")
	}
	if token.Error() != nil {
		panic(token.Error())
	}

	log.Printf("✅ Connected to broker %s\n", address)

	return client
}

// buildOnConnectHandler wires the MQTT OnConnect callback to resubscribe and
// run any extra post-connect work.
func buildOnConnectHandler(topic string, execute Executor, onConnect func(mqtt.Client) error) func(mqtt.Client) {
	return func(c mqtt.Client) {
		log.Println("Connected to broker.")
		if execute != nil {
			subscribe(c, topic, execute)
		}
		if onConnect != nil {
			if err := onConnect(c); err != nil {
				log.Printf("OnConnect hook error: %v", err)
			}
		}
	}
}

func Publish(client mqtt.Client, payload string, topic string) {
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

func subscribe(client mqtt.Client, topic string, handler func(string) error) mqtt.Client {
	token := client.Subscribe(topic, qos, func(client mqtt.Client, msg mqtt.Message) {
		message := string(msg.Payload())
		if err := handler(message); err != nil {
			log.Printf("Error handling message: %v", err)
		}
	})
	token.Wait()
	if err := token.Error(); err != nil {
		log.Println("Subscribe error:", err)
	}

	log.Printf("Listening messages on topic '%s'\n", topic)

	return client
}
