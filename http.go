package main

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"log"
	"net/http"
	"os/exec"
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

var services = map[string]string{
	"snapserver": "sudo systemctl restart snapserver.service",
	"snapclient": "systemctl --user restart snapclient.service",
	"spotifyd":   "systemctl --user restart spotifyd.service",
}

const shell = "/usr/bin/bash"

func execute(dryRun bool, cmd string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	log.Printf("$ %s %s %s", shell, "-c", cmd)
	if !dryRun {
		cmd := exec.Command(shell, "-c", cmd)
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		return stdout.String(), stderr.String(), err
	}

	return "", "", nil
}

func restart(dryRun bool, allowAdress string, w http.ResponseWriter, r *http.Request) {
	log.Printf("[GET] %s %s\n", r.RemoteAddr, r.URL)
	log.Printf("[GET] %s %s\n", r.Header, r.URL)
	if strings.Split(r.RemoteAddr, ":")[0] != allowAdress {
		log.Println("Unauthorized")
		return
	}
	service := html.EscapeString(r.URL.Query().Get("s"))
	cmd, ok := services[service]
	if !ok {
		log.Println("Unknown service")
		return
	}

	out, errout, err := execute(dryRun, cmd)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Fprintf(w, "{\"result\": \"ok\"}\n")
	if out != "" {
		log.Print(out)
	}
	if errout != "" {
		log.Print(errout)
	}
}

func CreateRestartHandler(dryRun bool, allowAddress string) func(http.ResponseWriter, *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		restart(dryRun, allowAddress, w, r)
	}
}
