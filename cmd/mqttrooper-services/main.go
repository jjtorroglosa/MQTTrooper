package main

import (
	"html/template"
	"log"
	"os"
)

func generatePlist() {
	tmpl, err := template.ParseFiles("templates/com.jjtorroglosa.mqttrooper.plist.tmpl")
	if err != nil {
		log.Fatalf("Error parsing plist tmpl: %v", err)
	}

	err = tmpl.Execute(os.Stdout, struct {
		ExecutablePath string
		ConfigPath     string
		Home           string
		LogfilePath    string
		ErrLogfilePath string
	}{
		ExecutablePath: "/Users/jtorr/src/mqttrooper/dist/mqttrooper.arm64.darwin",
		ConfigPath:     "/Users/jtorr/src/mqttrooper/config.yaml",
		Home:           "/Users/jtorr",
		LogfilePath:    "/tmp/mqttrooper_jtorr.info.log",
		ErrLogfilePath: "/tmp/mqttrooper_jtorr.error.log",
	})

	if err != nil {
		log.Fatalf("Error rendering plist tmpl: %v", err)
	}
}

func generateService() {
	tmpl, err := template.ParseFiles("templates/mqttrooper.service.tmpl")
	if err != nil {
		log.Fatalf("Error parsing plist tmpl: %v", err)
	}

	err = tmpl.Execute(os.Stdout, struct {
		After            string
		ExecutablePath   string
		BindIpAddress    string
		BindPort         string
		WorkingDirectory string
		ConfigPath       string
	}{
		After:            "snapclient",
		ExecutablePath:   "/Users/jtorr/src/mqttrooper/dist/mqttrooper.arm64.darwin",
		ConfigPath:       "/Users/jtorr/src/mqttrooper/config.yaml",
		WorkingDirectory: "/Users/jtorr/src/mqttrooper",
		BindIpAddress:    "127.0.0.1",
		BindPort:         "8989",
	})

	if err != nil {
		log.Fatalf("Error rendering plist tmpl: %v", err)
	}
}

func main() {
	generateService()
}
