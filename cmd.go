package main

import (
	"bytes"
	"errors"
	"log"
	"os/exec"
)

var services = map[string]string{
	"snapserver": "sudo systemctl restart snapserver.service",
	"snapclient": "systemctl --user restart snapclient.service",
	"spotifyd":   "systemctl --user restart spotifyd.service",
}

const shell = "/usr/bin/bash"

func execute(dryRun bool, service string) (string, string, error) {
	cmd, ok := services[service]
	if !ok {
		return "","", errors.New("Unknown service")
	}
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
