package internal

import (
	"fmt"
	"strings"
)

type EntityOperation string

const (
	EntityOpGet EntityOperation = "get"
	EntityOpSet EntityOperation = "set"
	EntityOpOn  EntityOperation = "on"
	EntityOpOff EntityOperation = "off"
)

type EntityExecutor interface {
	Execute(entityName string, op EntityOperation, value *string) (string, error)
}

type shellEntityExecutor struct {
	dryRun   bool
	shell    string
	entities map[string]EntityConfig
}

func CreateEntityExecutor(dryRun bool, shell string, entities map[string]EntityConfig) EntityExecutor {
	if entities == nil {
		entities = map[string]EntityConfig{}
	}
	return shellEntityExecutor{dryRun: dryRun, shell: shell, entities: entities}
}

func (e shellEntityExecutor) Execute(entityName string, op EntityOperation, value *string) (string, error) {
	entity, ok := e.entities[entityName]
	if !ok {
		return "", fmt.Errorf("unknown entity %q", entityName)
	}

	command, err := entityCommand(entity, op, value)
	if err != nil {
		return "", err
	}

	return runShellCommand(e.shell, command, e.dryRun)
}

func entityCommand(entity EntityConfig, op EntityOperation, value *string) (string, error) {
	switch op {
	case EntityOpGet:
		if entity.Get == "" {
			return "", fmt.Errorf("entity get command is empty")
		}
		return entity.Get, nil
	case EntityOpSet:
		if entity.Set == "" {
			return "", fmt.Errorf("entity set command is empty")
		}
		if value == nil {
			return "", fmt.Errorf("entity set operation requires a value")
		}
		return replaceValue(entity.Set, *value), nil
	case EntityOpOn:
		if entity.On == "" {
			return "", fmt.Errorf("entity on command is empty")
		}
		return entity.On, nil
	case EntityOpOff:
		if entity.Off == "" {
			return "", fmt.Errorf("entity off command is empty")
		}
		return entity.Off, nil
	default:
		return "", fmt.Errorf("unknown entity operation %q", op)
	}
}

func replaceValue(command string, value string) string {
	return strings.ReplaceAll(command, "{value}", value)
}
