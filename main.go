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

func listenHttp(cfg Config, address string, allow string, dryRun bool, port int) {
	http.HandleFunc("/", Home)
	http.HandleFunc("/r", CreateRestartHandler(dryRun, allow, cfg.Services, cfg.Executor.Shell))

	bindAddress := fmt.Sprintf("%s:%d", address, port)
	log.Printf("Listening on http://%s\n", bindAddress)
	log.Fatal(http.ListenAndServe(bindAddress, nil))
}

func main() {
	var dryRun = flag.Bool("d", false, "Don't run the commands. For testing purposes")
	var port = flag.Int("p", 8080, "Port to listen for HTTP requests")
	var address = flag.String("b", "127.0.0.1", "Address to bind to")
	var allow = flag.String("allow", "127.0.0.1", "Address to allow requests from")

	var configFile = GetFlag()

	flag.Parse()

	fmt.Println("-------------------------------------------------")
	fmt.Println("                 mqtt-commander                  ")
	fmt.Println("-------------------------------------------------")

	if *dryRun {
		fmt.Println("** Dry run mode **")
	}

	cfg := load(*configFile)

	if cfg.Http.Enabled {
		go listenHttp(cfg, *address, *allow, *dryRun, *port)
	}
	var mqttClient mqtt.Client = nil
	if cfg.Mqtt.Enabled {
		mqttClient = listenMqtt(cfg, *dryRun)
	}

	sigtermChan := make(chan os.Signal, 1)
	signal.Notify(sigtermChan, os.Interrupt, syscall.SIGTERM)

	<-sigtermChan
	log.Println("SIGTERM received. Exiting...")
	if mqttClient != nil {
		mqttClient.Disconnect(250)
	}
	log.Println("Client Disconnected")
	os.Exit(1)

}
