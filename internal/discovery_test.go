package internal

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
)

func TestPublishDiscoveryEmitsButtonConfigPerService(test *testing.T) {
	t := setupMqttTest(test, nil)
	defer t.teardown()

	host, _ := t.testContainer.Host(t.ctx)
	port, _ := t.testContainer.MappedPort(t.ctx, "1883/tcp")
	address := fmt.Sprintf("%s:%s", host, port)

	cfg := &Config{
		Mqtt: MqttConfig{
			Enabled:  true,
			Topic:    "/mqttrooper/test",
			ClientID: "test",
			Discovery: DiscoveryConfig{
				Enabled:      true,
				Prefix:       "homeassistant",
				DevicePrefix: "mqttrooper_test",
				DeviceName:   "mqttrooper test",
			},
		},
		Entities: map[string]EntityConfig{
			"alpha": {Type: EntityTypeCommand, Run: "echo a"},
			"beta":  {Type: EntityTypeCommand, Run: "echo b"},
		},
		Services: ServicesMap{},
	}

	type received struct {
		topic    string
		payload  []byte
		retained bool
	}
	var (
		mu   sync.Mutex
		msgs []received
	)
	done := make(chan struct{})

	subOpts := mqtt.NewClientOptions().
		AddBroker(address).
		SetClientID("discovery-sub").
		SetCleanSession(true).
		SetOnConnectHandler(func(c mqtt.Client) {
			c.Subscribe("homeassistant/button/#", 0, func(_ mqtt.Client, m mqtt.Message) {
				mu.Lock()
				msgs = append(msgs, received{topic: m.Topic(), payload: m.Payload(), retained: m.Retained()})
				if len(msgs) == 2 {
					select {
					case <-done:
					default:
						close(done)
					}
				}
				mu.Unlock()
			})
		})
	sub := mqtt.NewClient(subOpts)
	if token := sub.Connect(); token.WaitTimeout(3*time.Second) && token.Error() != nil {
		t.Fatal(token.Error())
	}
	defer sub.Disconnect(250)
	// Wait for subscription to register.
	time.Sleep(200 * time.Millisecond)

	pubOpts := mqtt.NewClientOptions().AddBroker(address).SetClientID("discovery-pub").SetCleanSession(true)
	pub := mqtt.NewClient(pubOpts)
	if token := pub.Connect(); token.WaitTimeout(3*time.Second) && token.Error() != nil {
		t.Fatal(token.Error())
	}
	defer pub.Disconnect(250)

	err := PublishDiscovery(pub, cfg)
	assert.NoError(t, err)

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for discovery messages")
	}

	mu.Lock()
	defer mu.Unlock()
	assert.Len(t, msgs, 2)

	byTopic := map[string]received{}
	for _, m := range msgs {
		byTopic[m.topic] = m
	}

	alpha, ok := byTopic["homeassistant/button/mqttrooper_test/alpha/config"]
	assert.True(t, ok, "alpha topic missing")

	var payload map[string]any
	assert.NoError(t, json.Unmarshal(alpha.payload, &payload))
	assert.Equal(t, "alpha", payload["name"])
	assert.Equal(t, "mqttrooper_test_alpha", payload["unique_id"])
	assert.Equal(t, "mqttrooper_test_alpha", payload["object_id"])
	assert.Equal(t, "/mqttrooper/test", payload["command_topic"])
	assert.Equal(t, "alpha", payload["payload_press"])
	device, ok := payload["device"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "mqttrooper test", device["name"])
	ids, ok := device["identifiers"].([]any)
	assert.True(t, ok)
	assert.Equal(t, []any{"mqttrooper_test"}, ids)

	_, ok = byTopic["homeassistant/button/mqttrooper_test/beta/config"]
	assert.True(t, ok, "beta topic missing")

	// Verify the messages were published with the retained flag by connecting
	// a fresh subscriber after publishing: the broker replays retained
	// messages to new subscribers with retained=true.
	retainedCh := make(chan mqtt.Message, 4)
	lateOpts := mqtt.NewClientOptions().
		AddBroker(address).
		SetClientID("discovery-late-sub").
		SetCleanSession(true).
		SetOnConnectHandler(func(c mqtt.Client) {
			c.Subscribe("homeassistant/button/#", 0, func(_ mqtt.Client, m mqtt.Message) {
				retainedCh <- m
			})
		})
	late := mqtt.NewClient(lateOpts)
	if token := late.Connect(); token.WaitTimeout(3*time.Second) && token.Error() != nil {
		t.Fatal(token.Error())
	}
	defer late.Disconnect(250)

	seen := 0
	for seen < 2 {
		select {
		case m := <-retainedCh:
			assert.True(t, m.Retained(), "expected retained=true on replayed message for %s", m.Topic())
			seen++
		case <-time.After(3 * time.Second):
			t.Fatalf("timeout waiting for retained replay; got %d/2", seen)
		}
	}
}

func TestPublishDiscoveryBackwardCompatServicesOnlyConfig(test *testing.T) {
	t := setupMqttTest(test, nil)
	defer t.teardown()

	host, _ := t.testContainer.Host(t.ctx)
	port, _ := t.testContainer.MappedPort(t.ctx, "1883/tcp")
	address := fmt.Sprintf("%s:%s", host, port)

	// Simulate a config loaded from a file that only has `services` (no `entities`).
	// LoadConfigFile folds services into Entities, so we replicate that here.
	cfg, err := loadConfigFromContent(t.T, `
mqtt:
  enabled: true
  topic: /mqttrooper/test
  discovery:
    enabled: true
    prefix: homeassistant
    device_prefix: mqttrooper_compat
    device_name: mqttrooper compat
services:
  alpha: echo a
  beta: echo b
`)
	assert.NoError(t, err)

	received := make(chan mqtt.Message, 4)
	subOpts := mqtt.NewClientOptions().
		AddBroker(address).
		SetClientID("compat-sub").
		SetCleanSession(true).
		SetOnConnectHandler(func(c mqtt.Client) {
			c.Subscribe("homeassistant/button/#", 0, func(_ mqtt.Client, m mqtt.Message) {
				received <- m
			})
		})
	sub := mqtt.NewClient(subOpts)
	if token := sub.Connect(); token.WaitTimeout(3*time.Second) && token.Error() != nil {
		t.Fatal(token.Error())
	}
	defer sub.Disconnect(250)
	time.Sleep(200 * time.Millisecond)

	pubOpts := mqtt.NewClientOptions().AddBroker(address).SetClientID("compat-pub").SetCleanSession(true)
	pub := mqtt.NewClient(pubOpts)
	if token := pub.Connect(); token.WaitTimeout(3*time.Second) && token.Error() != nil {
		t.Fatal(token.Error())
	}
	defer pub.Disconnect(250)

	assert.NoError(t, PublishDiscovery(pub, cfg))

	got := map[string]struct{}{}
	timeout := time.After(3 * time.Second)
	for len(got) < 2 {
		select {
		case m := <-received:
			got[m.Topic()] = struct{}{}
		case <-timeout:
			t.Fatalf("timeout waiting for button discovery messages; got %d/2", len(got))
		}
	}
	assert.Contains(t, got, "homeassistant/button/mqttrooper_compat/alpha/config")
	assert.Contains(t, got, "homeassistant/button/mqttrooper_compat/beta/config")
}

func TestPublishDiscoveryEmitsNumberConfig(test *testing.T) {
	t := setupMqttTest(test, nil)
	defer t.teardown()

	host, _ := t.testContainer.Host(t.ctx)
	port, _ := t.testContainer.MappedPort(t.ctx, "1883/tcp")
	address := fmt.Sprintf("%s:%s", host, port)

	cfg := &Config{
		Mqtt: MqttConfig{
			Enabled:  true,
			Topic:    "/mqttrooper/test",
			ClientID: "test-num",
			Discovery: DiscoveryConfig{
				Enabled:      true,
				Prefix:       "homeassistant",
				DevicePrefix: "mqttrooper_test",
				DeviceName:   "mqttrooper test",
			},
		},
		Entities: map[string]EntityConfig{
			"volume": {
				Type: EntityTypeNumber,
				Min:  0,
				Max:  100,
				Step: 1,
				Get:  "echo 50",
				Set:  "echo {value}",
			},
		},
		Services: ServicesMap{},
	}

	received := make(chan mqtt.Message, 4)
	subOpts := mqtt.NewClientOptions().
		AddBroker(address).
		SetClientID("num-disc-sub").
		SetCleanSession(true).
		SetOnConnectHandler(func(c mqtt.Client) {
			c.Subscribe("homeassistant/number/#", 0, func(_ mqtt.Client, m mqtt.Message) {
				received <- m
			})
		})
	sub := mqtt.NewClient(subOpts)
	if token := sub.Connect(); token.WaitTimeout(3*time.Second) && token.Error() != nil {
		t.Fatal(token.Error())
	}
	defer sub.Disconnect(250)
	time.Sleep(200 * time.Millisecond)

	pubOpts := mqtt.NewClientOptions().AddBroker(address).SetClientID("num-disc-pub").SetCleanSession(true)
	pub := mqtt.NewClient(pubOpts)
	if token := pub.Connect(); token.WaitTimeout(3*time.Second) && token.Error() != nil {
		t.Fatal(token.Error())
	}
	defer pub.Disconnect(250)

	assert.NoError(t, PublishDiscovery(pub, cfg))

	select {
	case m := <-received:
		assert.Equal(t, "homeassistant/number/mqttrooper_test/volume/config", m.Topic())
		var payload map[string]any
		assert.NoError(t, json.Unmarshal(m.Payload(), &payload))
		assert.Equal(t, "volume", payload["name"])
		assert.Equal(t, "mqttrooper_test_volume", payload["unique_id"])
		assert.Equal(t, "mqttrooper_test_volume", payload["object_id"])
		assert.Equal(t, float64(0), payload["min"])
		assert.Equal(t, float64(100), payload["max"])
		assert.Equal(t, float64(1), payload["step"])
		assert.Equal(t, "/mqttrooper/test/number/volume/set", payload["command_topic"])
		assert.Equal(t, "/mqttrooper/test/number/volume/state", payload["state_topic"])
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for number discovery message")
	}
}

func TestPublishDiscoveryEmitsSwitchConfig(test *testing.T) {
	t := setupMqttTest(test, nil)
	defer t.teardown()

	host, _ := t.testContainer.Host(t.ctx)
	port, _ := t.testContainer.MappedPort(t.ctx, "1883/tcp")
	address := fmt.Sprintf("%s:%s", host, port)

	cfg := &Config{
		Mqtt: MqttConfig{
			Enabled:  true,
			Topic:    "/mqttrooper/test",
			ClientID: "test-bool",
			Discovery: DiscoveryConfig{
				Enabled:      true,
				Prefix:       "homeassistant",
				DevicePrefix: "mqttrooper_test",
				DeviceName:   "mqttrooper test",
			},
		},
		Entities: map[string]EntityConfig{
			"mute": {
				Type: EntityTypeSwitch,
				Get:  "echo yes",
				On:   "echo on",
				Off:  "echo off",
			},
		},
		Services: ServicesMap{},
	}

	received := make(chan mqtt.Message, 4)
	subOpts := mqtt.NewClientOptions().
		AddBroker(address).
		SetClientID("bool-disc-sub").
		SetCleanSession(true).
		SetOnConnectHandler(func(c mqtt.Client) {
			c.Subscribe("homeassistant/switch/#", 0, func(_ mqtt.Client, m mqtt.Message) {
				received <- m
			})
		})
	sub := mqtt.NewClient(subOpts)
	if token := sub.Connect(); token.WaitTimeout(3*time.Second) && token.Error() != nil {
		t.Fatal(token.Error())
	}
	defer sub.Disconnect(250)
	time.Sleep(200 * time.Millisecond)

	pubOpts := mqtt.NewClientOptions().AddBroker(address).SetClientID("bool-disc-pub").SetCleanSession(true)
	pub := mqtt.NewClient(pubOpts)
	if token := pub.Connect(); token.WaitTimeout(3*time.Second) && token.Error() != nil {
		t.Fatal(token.Error())
	}
	defer pub.Disconnect(250)

	assert.NoError(t, PublishDiscovery(pub, cfg))

	select {
	case m := <-received:
		assert.Equal(t, "homeassistant/switch/mqttrooper_test/mute/config", m.Topic())
		var payload map[string]any
		assert.NoError(t, json.Unmarshal(m.Payload(), &payload))
		assert.Equal(t, "mute", payload["name"])
		assert.Equal(t, "mqttrooper_test_mute", payload["unique_id"])
		assert.Equal(t, "mqttrooper_test_mute", payload["object_id"])
		assert.Equal(t, "/mqttrooper/test/switch/mute/set", payload["command_topic"])
		assert.Equal(t, "/mqttrooper/test/switch/mute/state", payload["state_topic"])
		assert.Equal(t, "ON", payload["payload_on"])
		assert.Equal(t, "OFF", payload["payload_off"])
		assert.Equal(t, "ON", payload["state_on"])
		assert.Equal(t, "OFF", payload["state_off"])
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for switch discovery message")
	}
}

func TestPublishDiscoveryClearsStaleEntriesAllTypes(test *testing.T) {
	t := setupMqttTest(test, nil)
	defer t.teardown()

	host, _ := t.testContainer.Host(t.ctx)
	port, _ := t.testContainer.MappedPort(t.ctx, "1883/tcp")
	address := fmt.Sprintf("%s:%s", host, port)

	dp := "mqttrooper_multiclean"
	cfg := &Config{
		Mqtt: MqttConfig{
			Enabled: true,
			Topic:   "/mqttrooper/test",
			Discovery: DiscoveryConfig{
				Enabled:      true,
				Prefix:       "homeassistant",
				DevicePrefix: dp,
			},
		},
		Entities: map[string]EntityConfig{
			"kept-button": {Type: EntityTypeCommand, Run: "echo kept"},
			"kept-number": {Type: EntityTypeNumber, Min: 0, Max: 100, Step: 1, Get: "echo 50", Set: "echo {value}"},
			"kept-switch": {Type: EntityTypeSwitch, Get: "echo yes", On: "echo on", Off: "echo off"},
		},
		Services: ServicesMap{},
	}

	seed := func(entityType, name string, payload []byte) {
		opts := mqtt.NewClientOptions().AddBroker(address).SetClientID("seed-" + entityType + "-" + name).SetCleanSession(true)
		c := mqtt.NewClient(opts)
		if tok := c.Connect(); tok.WaitTimeout(3*time.Second) && tok.Error() != nil {
			t.Fatal(tok.Error())
		}
		defer c.Disconnect(250)
		topic := fmt.Sprintf("homeassistant/%s/%s/%s/config", entityType, dp, name)
		tok := c.Publish(topic, byte(qos), true, payload)
		tok.Wait()
		assert.NoError(t, tok.Error())
	}
	seed("button", "kept-button", []byte(`{"name":"kept-button"}`))
	seed("button", "gone-button", []byte(`{"name":"gone-button"}`))
	seed("number", "kept-number", []byte(`{"name":"kept-number"}`))
	seed("number", "gone-number", []byte(`{"name":"gone-number"}`))
	seed("switch", "kept-switch", []byte(`{"name":"kept-switch"}`))
	seed("switch", "gone-switch", []byte(`{"name":"gone-switch"}`))

	pubOpts := mqtt.NewClientOptions().AddBroker(address).SetClientID("multi-disc-run").SetCleanSession(true)
	pub := mqtt.NewClient(pubOpts)
	if tok := pub.Connect(); tok.WaitTimeout(3*time.Second) && tok.Error() != nil {
		t.Fatal(tok.Error())
	}
	defer pub.Disconnect(250)
	assert.NoError(t, PublishDiscovery(pub, cfg))

	// Fresh subscriber sees only retained messages that remain.
	got := map[string]struct{}{}
	var mu sync.Mutex
	subOpts := mqtt.NewClientOptions().
		AddBroker(address).
		SetClientID("multi-post-sub").
		SetCleanSession(true).
		SetOnConnectHandler(func(c mqtt.Client) {
			c.Subscribe(fmt.Sprintf("homeassistant/+/%s/+/config", dp), 0, func(_ mqtt.Client, m mqtt.Message) {
				mu.Lock()
				got[m.Topic()] = struct{}{}
				mu.Unlock()
			})
		})
	sub := mqtt.NewClient(subOpts)
	if tok := sub.Connect(); tok.WaitTimeout(3*time.Second) && tok.Error() != nil {
		t.Fatal(tok.Error())
	}
	defer sub.Disconnect(250)
	time.Sleep(700 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Contains(t, got, fmt.Sprintf("homeassistant/button/%s/kept-button/config", dp))
	assert.Contains(t, got, fmt.Sprintf("homeassistant/number/%s/kept-number/config", dp))
	assert.Contains(t, got, fmt.Sprintf("homeassistant/switch/%s/kept-switch/config", dp))
	assert.NotContains(t, got, fmt.Sprintf("homeassistant/button/%s/gone-button/config", dp))
	assert.NotContains(t, got, fmt.Sprintf("homeassistant/number/%s/gone-number/config", dp))
	assert.NotContains(t, got, fmt.Sprintf("homeassistant/switch/%s/gone-switch/config", dp))
}

func TestPublishDiscoveryClearsStaleEntries(test *testing.T) {
	t := setupMqttTest(test, nil)
	defer t.teardown()

	host, _ := t.testContainer.Host(t.ctx)
	port, _ := t.testContainer.MappedPort(t.ctx, "1883/tcp")
	address := fmt.Sprintf("%s:%s", host, port)

	cfg := &Config{
		Mqtt: MqttConfig{
			Enabled: true,
			Topic:   "/mqttrooper/test",
			Discovery: DiscoveryConfig{
				Enabled:      true,
				Prefix:       "homeassistant",
				DevicePrefix: "mqttrooper_cleanup",
				DeviceName:   "mqttrooper cleanup",
			},
		},
		Entities: map[string]EntityConfig{
			"kept": {Type: EntityTypeCommand, Run: "echo kept"},
		},
		Services: ServicesMap{},
	}

	// Seed a stale retained discovery message for a service that is no
	// longer in the config, plus one sibling the cleanup should preserve
	// (the broker will replay both, but only "gone" should be cleared).
	seed := func(t *testing.T, service string, payload []byte) {
		opts := mqtt.NewClientOptions().AddBroker(address).SetClientID("seed-" + service).SetCleanSession(true)
		c := mqtt.NewClient(opts)
		if tok := c.Connect(); tok.WaitTimeout(3*time.Second) && tok.Error() != nil {
			t.Fatal(tok.Error())
		}
		defer c.Disconnect(250)
		topic := fmt.Sprintf("homeassistant/button/mqttrooper_cleanup/%s/config", service)
		tok := c.Publish(topic, byte(qos), true, payload)
		tok.Wait()
		assert.NoError(t, tok.Error())
	}
	seed(test, "gone", []byte(`{"name":"gone","command_topic":"/x","payload_press":"gone"}`))
	seed(test, "kept", []byte(`{"name":"kept","command_topic":"/x","payload_press":"kept"}`))

	// Run PublishDiscovery (which should clear "gone" and re-publish "kept").
	pubOpts := mqtt.NewClientOptions().AddBroker(address).SetClientID("disc-run").SetCleanSession(true)
	pub := mqtt.NewClient(pubOpts)
	if tok := pub.Connect(); tok.WaitTimeout(3*time.Second) && tok.Error() != nil {
		t.Fatal(tok.Error())
	}
	defer pub.Disconnect(250)

	assert.NoError(t, PublishDiscovery(pub, cfg))

	// Fresh subscriber: broker replays only retained messages that still
	// exist. We expect exactly "kept" and NOT "gone".
	got := map[string][]byte{}
	var mu sync.Mutex
	subOpts := mqtt.NewClientOptions().
		AddBroker(address).
		SetClientID("post-cleanup-sub").
		SetCleanSession(true).
		SetOnConnectHandler(func(c mqtt.Client) {
			c.Subscribe("homeassistant/button/mqttrooper_cleanup/+/config", 0, func(_ mqtt.Client, m mqtt.Message) {
				mu.Lock()
				got[m.Topic()] = m.Payload()
				mu.Unlock()
			})
		})
	sub := mqtt.NewClient(subOpts)
	if tok := sub.Connect(); tok.WaitTimeout(3*time.Second) && tok.Error() != nil {
		t.Fatal(tok.Error())
	}
	defer sub.Disconnect(250)

	time.Sleep(700 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	_, keptOk := got["homeassistant/button/mqttrooper_cleanup/kept/config"]
	assert.True(t, keptOk, "kept entity should remain after cleanup")
	_, goneOk := got["homeassistant/button/mqttrooper_cleanup/gone/config"]
	assert.False(t, goneOk, "gone entity should be cleared (no retained message replayed)")
}
