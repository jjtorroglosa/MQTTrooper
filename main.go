package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)


func main() {
	var dryRun = flag.Bool("d", false, "Don't run the commands. For testing purposes")
	var port = flag.Int("p", 8080, "Port to listen for HTTP requests")
	var address = flag.String("b", "127.0.0.1", "Address to bind to")
	var allow = flag.String("allow", "127.0.0.1", "Address to allow requests from")

	flag.Parse()

	fmt.Println("-------------------------------------------------")
	fmt.Println("                 systemd-api                     ")
	fmt.Println("-------------------------------------------------")

	if *dryRun {
		fmt.Println("** Dry run mode **")
	}

	http.HandleFunc("/", Home)
	http.HandleFunc("/r", CreateRestartHandler(*dryRun, *allow))

	bind := fmt.Sprintf("%s:%d", *address, *port)
	log.Printf("Listening on http://%s\n", bind)
	log.Fatal(http.ListenAndServe(bind, nil))
}
