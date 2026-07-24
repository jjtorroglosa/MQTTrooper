package internal

import "fmt"

type Executor func(service string) error

func CreateExecutor(dryRun bool, shell string, services map[string]string) Executor {
	return func(service string) error {
		commandToExecute, ok := services[service]
		if !ok {
			return fmt.Errorf("unknown service %q", service)
		}
		_, err := runShellCommand(shell, commandToExecute, dryRun)
		return err
	}
}
