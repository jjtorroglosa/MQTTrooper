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

func HomeHandler(w http.ResponseWriter, r *http.Request) {
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

func ExecuteHandler(
	execute Executor,
	allowedAddress string,
	w http.ResponseWriter,
	r *http.Request,
) error {
	log.Printf("[GET] %s %s\n", r.RemoteAddr, r.URL)
	log.Printf("[GET] %s %s\n", r.Header, r.URL)
	if strings.Split(r.RemoteAddr, ":")[0] != allowedAddress {
		unauthorized := "Unauthorized"
		log.Println(unauthorized)
		return errors.New(unauthorized)
	}
	service := html.EscapeString(r.URL.Query().Get("s"))

	err := execute(service)
	if err != nil {
		log.Println(err)
		return err
	}
	fmt.Fprintf(w, "{\"result\": \"ok\"}\n")

	return nil
}
