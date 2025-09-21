package main

import (
	"log"
	"mqttrooper/internal"
	"os"
	"text/template"
)

func generatePlist(cfg internal.Config) {
	tmpl, err := template.ParseFiles("templates/com.jjtorroglosa.mqttrooper.plist.tmpl")
	if err != nil {
		log.Fatalf("Error parsing plist tmpl: %v", err)
	}

	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Error getting executable: %v", err)
	}
	err = tmpl.Execute(os.Stdout, struct {
		ExecutablePath string
		ConfigPath     string
		PathEnv        string
		LogfilePath    string
		ErrLogfilePath string
		MacId          string
	}{
		ExecutablePath: exePath,
		ConfigPath:     cfg.ConfigPath,
		PathEnv:        cfg.Daemon.EnvPath,
		LogfilePath:    cfg.Daemon.LogFilePath,
		ErrLogfilePath: cfg.Daemon.ErrorFilePath,
		MacId:          cfg.Daemon.MacId,
	})

	if err != nil {
		log.Fatalf("Error rendering plist tmpl: %v", err)
	}
}

func generateSystemdService(cfg internal.Config) {
	executablePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Error getting executable: %v", err)
	}
	tmpl, err := template.ParseFiles("templates/mqttrooper.service.tmpl")
	if err != nil {
		log.Fatalf("Error parsing plist tmpl: %v", err)
	}

	err = tmpl.Execute(os.Stdout, struct {
		After            string
		ExecutablePath   string
		WorkingDirectory string
		ConfigPath       string
		EnvPath          string
	}{
		After:            "",
		ExecutablePath:   executablePath,
		ConfigPath:       cfg.ConfigPath,
		WorkingDirectory: cfg.Daemon.Cwd,
		EnvPath:          cfg.Daemon.EnvPath,
	})

	if err != nil {
		log.Fatalf("Error rendering plist tmpl: %v", err)
	}
}
