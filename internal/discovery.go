package internal

import (
	"encoding/json"
	"fmt"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type discoveryDevice struct {
	Identifiers  []string `json:"identifiers"`
	Name         string   `json:"name"`
	Manufacturer string   `json:"manufacturer"`
}

type buttonDiscovery struct {
	Name         string          `json:"name"`
	UniqueID     string          `json:"unique_id"`
	ObjectID     string          `json:"object_id"`
	CommandTopic string          `json:"command_topic"`
	PayloadPress string          `json:"payload_press"`
	Device       discoveryDevice `json:"device"`
}

// PublishDiscovery publishes a Home Assistant MQTT discovery config message for
// each service in cfg.ServicesList, modelling each service as a stateless
// `button` entity. Messages are published with the retained flag so HA picks
// them up whenever it (re)starts. Errors publishing individual entities are
// logged but do not abort the batch.
func PublishDiscovery(client mqtt.Client, cfg *Config) error {
	d := cfg.Mqtt.Discovery
	deviceName := d.DeviceName
	if deviceName == "" {
		deviceName = d.DevicePrefix
	}
	device := discoveryDevice{
		Identifiers:  []string{d.DevicePrefix},
		Name:         deviceName,
		Manufacturer: "mqttrooper",
	}

	for _, svc := range cfg.ServicesList {
		entityID := fmt.Sprintf("%s_%s", d.DevicePrefix, svc.Name)
		payload := buttonDiscovery{
			Name:         svc.Name,
			UniqueID:     entityID,
			ObjectID:     entityID,
			CommandTopic: cfg.Mqtt.Topic,
			PayloadPress: svc.Name,
			Device:       device,
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal discovery for %s: %w", svc.Name, err)
		}
		topic := fmt.Sprintf("%s/button/%s/%s/config", d.Prefix, d.DevicePrefix, svc.Name)
		if err := publishRetained(client, topic, encoded, true); err != nil {
			log.Printf("discovery publish failed for %s: %v", svc.Name, err)
			continue
		}
		log.Printf("discovery published: %s", topic)
	}
	return nil
}

func publishRetained(client mqtt.Client, topic string, payload []byte, retained bool) error {
	token := client.Publish(topic, byte(qos), retained, payload)
	token.Wait()
	return token.Error()
}
