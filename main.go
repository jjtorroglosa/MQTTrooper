package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func listenHttp(bindAddress string, port int, allowedAddress string, execute Executor) {
	http.HandleFunc("/", HomeHandler)
	http.HandleFunc("/r", func(w http.ResponseWriter, r *http.Request) {
		ExecuteHandler(execute, allowedAddress, w, r)
	})

	addressPort := fmt.Sprintf("%s:%d", bindAddress, port)
	log.Printf("Listening on http://%s\n", addressPort)
	log.Fatal(http.ListenAndServe(addressPort, nil))
}

func handleSigterm(client mqtt.Client) {
	sigtermChan := make(chan os.Signal, 1)
	signal.Notify(sigtermChan, os.Interrupt, syscall.SIGTERM)

	<-sigtermChan
	log.Println("SIGTERM received. Exiting...")
	if client != nil {
		client.Disconnect(250)
	}
	log.Println("Client Disconnected")
	os.Exit(1)
}

func main() {
	var dryRun = flag.Bool("d", false, "Don't run the commands. For testing purposes")
	var port = flag.Int("p", 8080, "Port to listen for HTTP requests")
	var address = flag.String("b", "127.0.0.1", "Address to bind to")
	var allowedAddress = flag.String("allow", "127.0.0.1", "Address to allow requests from")
	var pub = flag.Bool("publish", false, "Use this flag to publish messages to the topic instead of subscribing to it")
	var payload = flag.String("message", "", "The message to publish")
	var user = flag.String("user", "", "The message to publish")

	var configFile = GetFlag()

	flag.Parse()

	fmt.Println("-------------------------------------------------")
	fmt.Println("                    MQTTrooper                   ")
	fmt.Println("-------------------------------------------------")

	if *dryRun {
		fmt.Println("** Dry run mode **")
	}

	cfg := load(*configFile)
	if *user != "" {
		cfg.Mqtt.User = *user
		cfg.Executor.DryRun = *dryRun
	}
	client := connect(cfg.Mqtt.Address, cfg.Mqtt.User, cfg.Mqtt.Pass)

	if *pub == true {
		if *payload == "" {
			log.Fatalln("Empty payload. You need to provide a valid payload to publish")
		}

		publish(client, *payload, cfg.Mqtt.Topic)
		os.Exit(0)
	}
	execute := CreateExecutor(cfg.Executor.DryRun, cfg.Executor.Shell, cfg.Services)

	if cfg.Http.Enabled {
		go listenHttp(*address, *port, *allowedAddress, execute)
	}
	if cfg.Mqtt.Enabled {
		Subscribe(client, cfg.Mqtt.Topic, *dryRun, execute)
	}

	handleSigterm(client)
}
