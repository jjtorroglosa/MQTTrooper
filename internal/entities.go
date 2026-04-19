package internal

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// PublishEntityStates runs the get command for each stateful entity at startup
// and publishes the result as the initial HA state (retained).
func PublishEntityStates(client mqtt.Client, cfg *Config, shell string, dryRun bool) error {
	for name, e := range cfg.Entities {
		switch e.Type {
		case EntityTypeNumber:
			state, err := runGet(e.Get, shell, dryRun)
			if err != nil {
				log.Printf("entities: get failed for %s: %v", name, err)
				continue
			}
			topic := fmt.Sprintf("%s/number/%s/state", cfg.Mqtt.Topic, name)
			if err := publishRetained(client, topic, []byte(state), true); err != nil {
				log.Printf("entities: state publish failed for %s: %v", name, err)
			}
		}
	}
	return nil
}

// SubscribeEntities subscribes to command topics for stateful entities (number,
// boolean). On each incoming value it executes the set command, then runs get
// and publishes the result as the new state.
func SubscribeEntities(client mqtt.Client, cfg *Config, shell string, dryRun bool) error {
	for name, e := range cfg.Entities {
		name, e := name, e
		switch e.Type {
		case EntityTypeNumber:
			cmdTopic := fmt.Sprintf("%s/number/%s/set", cfg.Mqtt.Topic, name)
			stateTopic := fmt.Sprintf("%s/number/%s/state", cfg.Mqtt.Topic, name)
			tok := client.Subscribe(cmdTopic, byte(qos), func(_ mqtt.Client, m mqtt.Message) {
				value := strings.TrimSpace(string(m.Payload()))
				if _, err := strconv.ParseFloat(value, 64); err != nil {
					log.Printf("entities: invalid number payload for %s: %q", name, value)
					return
				}
				cmd := strings.ReplaceAll(e.Set, "{value}", value)
				if err := runShell(cmd, shell, dryRun); err != nil {
					log.Printf("entities: set failed for %s: %v", name, err)
					return
				}
				state, err := runGet(e.Get, shell, dryRun)
				if err != nil {
					log.Printf("entities: get failed after set for %s: %v", name, err)
					return
				}
				if err := publishRetained(client, stateTopic, []byte(state), true); err != nil {
					log.Printf("entities: state publish failed for %s: %v", name, err)
				}
			})
			tok.Wait()
			if err := tok.Error(); err != nil {
				return fmt.Errorf("subscribe %s: %w", cmdTopic, err)
			}
			log.Printf("entities: subscribed %s", cmdTopic)
		}
	}
	return nil
}

func runGet(cmd string, shell string, dryRun bool) (string, error) {
	if dryRun {
		return "", nil
	}
	var out bytes.Buffer
	parts := strings.Fields(shell)
	parts = append(parts, "-c", cmd)
	c := exec.Command(parts[0], parts[1:]...)
	c.Stdout = &out
	c.Stderr = &out
	if err := c.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

func runShell(cmd string, shell string, dryRun bool) error {
	if dryRun {
		return nil
	}
	parts := strings.Fields(shell)
	parts = append(parts, "-c", cmd)
	return exec.Command(parts[0], parts[1:]...).Run()
}
