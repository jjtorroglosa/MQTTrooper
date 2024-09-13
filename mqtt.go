package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

func connect() (MQTT.Client, chan [2]string) {
	broker := "tcp://***REMOVED***:1883"
	user := "systemd-api"
	password := "***REMOVED***"
	topic := "/systemd-api/b"
	cleansess := flag.Bool("clean", false, "Set Clean Session (default false)")
	qos := flag.Int("qos", 0, "The Quality of Service 0,1,2 (default 0)")
	flag.Parse()

	opts := MQTT.NewClientOptions()
	opts.AddBroker(broker)
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
		fmt.Println(token.Error())
		os.Exit(1)
	}
	fmt.Printf("Connected to %s\n", broker)
	fmt.Printf("Listening topic: %s\n", topic)
	return client, choke
}

func main() {
	client, choke := connect()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("Exiting...")
		client.Disconnect(250)
		fmt.Println("Client Disconnected")
		os.Exit(1)
	}()

	for {
		incoming := <-choke
		fmt.Printf("Received in %s -> %s\n", incoming[0], incoming[1])
		execute(true, incoming[1])
	}
}
