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
	tmpl, err := template.ParseFiles("templates/index.html.tmpl")
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

func restart(
	dryRun bool,
	allowAdress string,
	services map[string]string,
	shell string,
	w http.ResponseWriter,
	r *http.Request,
) error {
	log.Printf("[GET] %s %s\n", r.RemoteAddr, r.URL)
	log.Printf("[GET] %s %s\n", r.Header, r.URL)
	if strings.Split(r.RemoteAddr, ":")[0] != allowAdress {
		unauthorized := "Unauthorized"
		log.Println(unauthorized)
		return errors.New(unauthorized)
	}
	service := html.EscapeString(r.URL.Query().Get("s"))

	out, err := execute(dryRun, service, "", services, shell)
	if err != nil {
		log.Println(err)
		return err
	}
	if out != "" {
		log.Print(out)
	}
	fmt.Fprintf(w, "{\"result\": \"ok\"}\n")

	return nil
}

func CreateRestartHandler(
	dryRun bool,
	allowAddress string,
	services map[string]string,
	shell string,
) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		restart(dryRun, allowAddress, services, shell, w, r)
	}
}
