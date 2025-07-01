package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func listenHttp(bindAddress string, port int, allowedAddress string, execute Executor) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", HomeHandler)
	mux.HandleFunc("/r", func(w http.ResponseWriter, r *http.Request) {
		ExecuteHandler(execute, allowedAddress, w, r)
	})

	addressPort := fmt.Sprintf("%s:%d", bindAddress, port)
	srv := http.Server{
		Addr:    addressPort,
		Handler: mux,
	}
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-quit
		log.Println("HTTP: Shutting down server...")

		// Context with timeout for graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Println("HTTP: Server forced to shutdown:", err)
		} else {
			log.Println("HTTP: Server exited gracefully")
		}
	}()
	log.Printf("Listening on http://%s\n", addressPort)
	log.Fatal(srv.ListenAndServe())
}

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

func getCfg() Config {
	var dryRun = flag.Bool("d", false, "Don't run the commands. For testing purposes")
	var port = flag.Int("p", 8080, "Port to listen for HTTP requests")
	var address = flag.String("b", "127.0.0.1", "Address to bind to")
	var allowedAddress = flag.String("allow", "127.0.0.1", "Address to allow requests from")
	var pub = flag.Bool("publish", false, "Use this flag to publish messages to the topic instead of subscribing to it")
	var payload = flag.String("message", "", "The message to publish")
	var user = flag.String("user", "", "Mqtt user")
	var password = flag.String("password", "", "Mqtt password")
	var configFile = flag.String("c", "config.yaml", "The path to the config.yaml file")

	flag.Parse()

	if *dryRun {
		fmt.Println("** Dry run mode **")
	}

	cfg := load(*configFile)
	cfg.Executor.DryRun = *dryRun
	if *user != "" {
		cfg.Mqtt.User = *user
	}
	if *password != "" {
		cfg.Mqtt.Pass = *password
	}

	cfg.Http.Port = *port
	cfg.Http.BindAddress = *address
	cfg.Http.AllowedAddress = *allowedAddress
	cfg.Mqtt.Payload = *payload
	cfg.Mqtt.Publish = *pub
	return cfg
}

func main() {

	fmt.Println("-------------------------------------------------")
	fmt.Println("                    MQTTrooper                   ")
	fmt.Println("-------------------------------------------------")
	cfg := getCfg()

	execute := CreateExecutor(cfg.Executor.DryRun, cfg.Executor.Shell, cfg.Services)

	var client mqtt.Client
	if cfg.Mqtt.Enabled {
		client = connect(cfg.Mqtt.Address, cfg.Mqtt.User, cfg.Mqtt.Pass, cfg.Mqtt.Topic, execute)
	}

	if cfg.Mqtt.Publish == true {
		if cfg.Mqtt.Payload == "" {
			log.Fatalln("Empty payload. You need to provide a valid payload to publish")
		}

		publish(client, cfg.Mqtt.Payload, cfg.Mqtt.Topic)
		os.Exit(0)
	}

	if cfg.Http.Enabled {
		go listenHttp(cfg.Http.BindAddress, cfg.Http.Port, cfg.Http.AllowedAddress, execute)
	}

	handleSigterm(client)
}
