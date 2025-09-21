package internal

import (
	"bytes"
	"errors"
	"log"
	"os/exec"
	"strings"
)

type Executor func(service string) error

func CreateExecutor(dryRun bool, shell string, services map[string]string) Executor {
	return func(service string) error {
		commandToExecute, ok := services[service]
		if !ok {
			return errors.New("Unknown service")
		}
		var output bytes.Buffer
		log.Printf("$ %s %s %s", shell, "-c", commandToExecute)
		if dryRun {
			return nil
		}
		parts := strings.Fields(shell) // → []string{"/usr/bin/env", "bash"}
		parts = append(parts, "-c")
		parts = append(parts, commandToExecute)
		log.Printf("%v\n", parts)
		cmd := exec.Command(parts[0], parts[1:]...)
		cmd.Stdout = &output
		cmd.Stderr = &output
		err := cmd.Run()
		log.Println("-------- output --------")
		log.Println(output.String())
		log.Println("------------------------")
		if err != nil {
			log.Println(err)
		}
		return err
	}
}
