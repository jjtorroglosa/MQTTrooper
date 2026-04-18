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
		ServicesList: ServicesList{
			{Name: "alpha", Command: "echo a"},
			{Name: "beta", Command: "echo b"},
		},
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
		ServicesList: ServicesList{
			{Name: "kept", Command: "echo kept"},
		},
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
