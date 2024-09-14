package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

func listenHttp(configFile string, address string, allow string, dryRun bool, port int) {
	cfg := load(configFile)
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
	fmt.Println("                 systemd-api                     ")
	fmt.Println("-------------------------------------------------")

	if *dryRun {
		fmt.Println("** Dry run mode **")
	}

	listenHttp(*configFile, *address, *allow, *dryRun, *port)
}
