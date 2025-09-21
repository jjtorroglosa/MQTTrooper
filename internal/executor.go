package internal

import (
	"bytes"
	"errors"
	"log"
	"os/exec"
)

type Executor func(service string) error

func CreateExecutor(dryRun bool, shell string, services ServicesMap) Executor {
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
		cmd := exec.Command(shell, "-c", commandToExecute)
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
