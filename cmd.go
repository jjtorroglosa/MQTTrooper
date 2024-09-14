package main

import (
	"bytes"
	"errors"
	"log"
	"os/exec"
)

const shell = "/bin/bash"

func execute(
	dryRun bool,
	service string,
	args string,
	services map[string]string,
	shell string,
) (string, error) {
	cmd, ok := services[service]
	if !ok {
		return "", errors.New("Unknown service")
	}
	var output bytes.Buffer
	log.Printf("$ %s %s %s", shell, "-c", cmd + " " + args)
	if !dryRun {
		cmd := exec.Command(shell, "-c", cmd + " " + args)
		cmd.Stdout = &output
		cmd.Stderr = &output
		err := cmd.Run()
		log.Println("  ------ output ------")
		log.Println(output.String())
		log.Println("  --------------------")
		return "", err
	}

	return "", nil
}
