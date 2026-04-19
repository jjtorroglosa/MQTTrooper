package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// discoveryScanWindow is how long we wait for the broker to replay retained
// discovery messages on our scan subscription before considering the
// collection complete.
var discoveryScanWindow = 500 * time.Millisecond

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

type numberDiscovery struct {
	Name         string          `json:"name"`
	UniqueID     string          `json:"unique_id"`
	ObjectID     string          `json:"object_id"`
	CommandTopic string          `json:"command_topic"`
	StateTopic   string          `json:"state_topic"`
	Min          float64         `json:"min"`
	Max          float64         `json:"max"`
	Step         float64         `json:"step"`
	Device       discoveryDevice `json:"device"`
}

// PublishDiscovery publishes Home Assistant MQTT discovery config messages for
// all entities in cfg.Entities. Messages are published with the retained flag
// so HA picks them up whenever it (re)starts. Errors publishing individual
// entities are logged but do not abort the batch.
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

	// Build per-type sets of current entity names for stale cleanup.
	currentByType := map[string]map[string]struct{}{
		"button": {},
		"number": {},
	}
	for name, e := range cfg.Entities {
		switch e.Type {
		case EntityTypeCommand:
			currentByType["button"][name] = struct{}{}
		case EntityTypeNumber:
			currentByType["number"][name] = struct{}{}
		}
	}
	for entityType, current := range currentByType {
		if err := cleanupStaleDiscovery(client, d, entityType, current); err != nil {
			log.Printf("discovery cleanup failed for %s: %v", entityType, err)
		}
	}

	for name, e := range cfg.Entities {
		entityID := fmt.Sprintf("%s_%s", d.DevicePrefix, name)
		var (
			encoded []byte
			err     error
			topic   string
		)
		switch e.Type {
		case EntityTypeCommand:
			topic = fmt.Sprintf("%s/button/%s/%s/config", d.Prefix, d.DevicePrefix, name)
			encoded, err = json.Marshal(buttonDiscovery{
				Name:         name,
				UniqueID:     entityID,
				ObjectID:     entityID,
				CommandTopic: cfg.Mqtt.Topic,
				PayloadPress: name,
				Device:       device,
			})
		case EntityTypeNumber:
			topic = fmt.Sprintf("%s/number/%s/%s/config", d.Prefix, d.DevicePrefix, name)
			encoded, err = json.Marshal(numberDiscovery{
				Name:         name,
				UniqueID:     entityID,
				ObjectID:     entityID,
				CommandTopic: fmt.Sprintf("%s/number/%s/set", cfg.Mqtt.Topic, name),
				StateTopic:   fmt.Sprintf("%s/number/%s/state", cfg.Mqtt.Topic, name),
				Min:          e.Min,
				Max:          e.Max,
				Step:         e.Step,
				Device:       device,
			})
		default:
			continue
		}
		if err != nil {
			return fmt.Errorf("marshal discovery for %s: %w", name, err)
		}
		if err := publishRetained(client, topic, encoded, true); err != nil {
			log.Printf("discovery publish failed for %s: %v", name, err)
			continue
		}
		log.Printf("discovery published: %s", topic)
	}
	return nil
}

// cleanupStaleDiscovery subscribes briefly to the discovery wildcard for the
// given entityType and device_prefix, collects whatever retained messages the
// broker replays, and clears any topics whose entity name is not in `current`.
func cleanupStaleDiscovery(client mqtt.Client, d DiscoveryConfig, entityType string, current map[string]struct{}) error {
	wildcard := fmt.Sprintf("%s/%s/%s/+/config", d.Prefix, entityType, d.DevicePrefix)

	var (
		mu    sync.Mutex
		seen  = map[string]struct{}{}
		first = make(chan struct{}, 1)
	)
	handler := func(_ mqtt.Client, m mqtt.Message) {
		if len(m.Payload()) == 0 {
			return
		}
		mu.Lock()
		seen[m.Topic()] = struct{}{}
		mu.Unlock()
		select {
		case first <- struct{}{}:
		default:
		}
	}

	token := client.Subscribe(wildcard, byte(qos), handler)
	token.Wait()
	if err := token.Error(); err != nil {
		return fmt.Errorf("subscribe %s: %w", wildcard, err)
	}
	defer func() {
		unsub := client.Unsubscribe(wildcard)
		unsub.Wait()
	}()

	// Wait at least one scan window; if retained messages start arriving,
	// wait one more window past the first one to catch the rest. Capped to
	// 3x the base window.
	select {
	case <-first:
		time.Sleep(discoveryScanWindow)
	case <-time.After(discoveryScanWindow):
	}

	mu.Lock()
	defer mu.Unlock()
	for topic := range seen {
		svc := entityFromDiscoveryTopic(topic, entityType, d)
		if svc == "" {
			continue
		}
		if _, keep := current[svc]; keep {
			continue
		}
		if err := publishRetained(client, topic, nil, true); err != nil {
			log.Printf("clear stale discovery %s: %v", topic, err)
			continue
		}
		log.Printf("cleared stale discovery: %s", topic)
	}
	return nil
}

func entityFromDiscoveryTopic(topic string, entityType string, d DiscoveryConfig) string {
	prefix := fmt.Sprintf("%s/%s/%s/", d.Prefix, entityType, d.DevicePrefix)
	suffix := "/config"
	if !strings.HasPrefix(topic, prefix) || !strings.HasSuffix(topic, suffix) {
		return ""
	}
	return topic[len(prefix) : len(topic)-len(suffix)]
}

func publishRetained(client mqtt.Client, topic string, payload []byte, retained bool) error {
	token := client.Publish(topic, byte(qos), retained, payload)
	token.Wait()
	return token.Error()
}
