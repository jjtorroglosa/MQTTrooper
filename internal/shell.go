package internal

import (
	"bytes"
	"log"
	"os/exec"
	"strings"
)

func runShellCommand(shell string, command string, dryRun bool) (string, error) {
	var output bytes.Buffer
	log.Printf("$ %s %s %s", shell, "-c", command)
	if dryRun {
		return "", nil
	}
	parts := strings.Fields(shell) // → []string{"/usr/bin/env", "bash"}
	parts = append(parts, "-c")
	parts = append(parts, command)
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
	return strings.TrimSpace(output.String()), err
}
