package internal

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// PublishEntityStates runs the get command for each stateful entity at startup
// and publishes the result as the initial HA state (retained).
func PublishEntityStates(client mqtt.Client, cfg *Config, executor EntityExecutor) error {
	for name, e := range cfg.Entities {
		switch e.Type {
		case EntityTypeNumber:
			state, err := executor.Execute(name, EntityOpGet, nil)
			if err != nil {
				log.Printf("entities: get failed for %s: %v", name, err)
				continue
			}
			topic := fmt.Sprintf("%s/number/%s/state", cfg.Mqtt.Topic, name)
			if err := publishRetained(client, topic, []byte(state), true); err != nil {
				log.Printf("entities: state publish failed for %s: %v", name, err)
			}
		case EntityTypeSwitch:
			raw, err := executor.Execute(name, EntityOpGet, nil)
			if err != nil {
				log.Printf("entities: get failed for %s: %v", name, err)
				continue
			}
			topic := fmt.Sprintf("%s/switch/%s/state", cfg.Mqtt.Topic, name)
			if err := publishRetained(client, topic, []byte(normalizeBool(raw)), true); err != nil {
				log.Printf("entities: state publish failed for %s: %v", name, err)
			}
		}
	}
	return nil
}

// SubscribeEntities subscribes to command topics for stateful entities (number,
// switch). On each incoming value it executes the set command, then runs get
// and publishes the result as the new state.
func SubscribeEntities(client mqtt.Client, cfg *Config, executor EntityExecutor) error {
	for name, e := range cfg.Entities {
		switch e.Type {
		case EntityTypeNumber:
			cmdTopic := fmt.Sprintf("%s/number/%s/set", cfg.Mqtt.Topic, name)
			stateTopic := fmt.Sprintf("%s/number/%s/state", cfg.Mqtt.Topic, name)
			tok := client.Subscribe(cmdTopic, byte(qos), func(_ mqtt.Client, m mqtt.Message) {
				payload := strings.TrimSpace(string(m.Payload()))
				if _, err := strconv.ParseFloat(payload, 64); err != nil {
					log.Printf("entities: invalid number payload for %s: %q", name, payload)
					return
				}
				if _, err := executor.Execute(name, EntityOpSet, &payload); err != nil {
					log.Printf("entities: set failed for %s: %v", name, err)
					return
				}
				state, err := executor.Execute(name, EntityOpGet, nil)
				if err != nil {
					log.Printf("entities: get failed after set for %s: %v", name, err)
					return
				}
				if err := publishRetained(client, stateTopic, []byte(state), true); err != nil {
					log.Printf("entities: state publish failed for %s: %v", name, err)
					return
				}
			})
			tok.Wait()
			if err := tok.Error(); err != nil {
				return fmt.Errorf("subscribe %s: %w", cmdTopic, err)
			}
			log.Printf("entities: subscribed %s", cmdTopic)
		case EntityTypeSwitch:
			cmdTopic := fmt.Sprintf("%s/switch/%s/set", cfg.Mqtt.Topic, name)
			stateTopic := fmt.Sprintf("%s/switch/%s/state", cfg.Mqtt.Topic, name)
			tok := client.Subscribe(cmdTopic, byte(qos), func(_ mqtt.Client, m mqtt.Message) {
				value := strings.TrimSpace(string(m.Payload()))
				var op EntityOperation
				switch value {
				case "ON":
					op = EntityOpOn
				case "OFF":
					op = EntityOpOff
				default:
					log.Printf("entities: invalid switch payload for %s: %q", name, value)
					return
				}
				if _, err := executor.Execute(name, op, nil); err != nil {
					log.Printf("entities: set failed for %s: %v", name, err)
					return
				}
				state, err := executor.Execute(name, EntityOpGet, nil)
				if err != nil {
					log.Printf("entities: get failed after set for %s: %v", name, err)
					return
				}
				if err := publishRetained(client, stateTopic, []byte(normalizeBool(state)), true); err != nil {
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

func normalizeBool(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return "ON"
	}
	return "OFF"
}
