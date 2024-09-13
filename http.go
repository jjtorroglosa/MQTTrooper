package main

import (
	"errors"
	"fmt"
	"html"
	"html/template"
	"log"
	"net/http"
	"strings"
)

type Link struct {
	Service string
}

var links = []Link{
	{Service: "snapclient"},
	{Service: "snapserver"},
	{Service: "spotifyd"},
}

func Home(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		fmt.Println("Error reading the template")
		fmt.Println(err)
		return
	}
	err2 := tmpl.Execute(w, links)
	if err2 != nil {
		fmt.Println(err2)
	}
}

func restart(dryRun bool, allowAdress string, w http.ResponseWriter, r *http.Request) error {
	log.Printf("[GET] %s %s\n", r.RemoteAddr, r.URL)
	log.Printf("[GET] %s %s\n", r.Header, r.URL)
	if strings.Split(r.RemoteAddr, ":")[0] != allowAdress {
		log.Println("Unauthorized")
		return errors.New("Unauthorized")
	}
	service := html.EscapeString(r.URL.Query().Get("s"))

	out, errout, err := execute(dryRun, service)
	if err != nil {
		log.Println(err)
        return err
	}
	if out != "" {
		log.Print(out)
	}
	if errout != "" {
		log.Print(errout)
	}
	fmt.Fprintf(w, "{\"result\": \"ok\"}\n")
    return nil
}

func CreateRestartHandler(dryRun bool, allowAddress string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		restart(dryRun, allowAddress, w, r)
	}
}
