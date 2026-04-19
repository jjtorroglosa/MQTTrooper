package internal

import (
	"fmt"
	"sync"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
)

func brokerAddress(t *mqttTest) string {
	host, _ := t.testContainer.Host(t.ctx)
	port, _ := t.testContainer.MappedPort(t.ctx, "1883/tcp")
	return fmt.Sprintf("%s:%s", host, port)
}

func newClient(t *mqttTest, id string) mqtt.Client {
	opts := mqtt.NewClientOptions().
		AddBroker(brokerAddress(t)).
		SetClientID(id).
		SetCleanSession(true)
	c := mqtt.NewClient(opts)
	if tok := c.Connect(); tok.WaitTimeout(3*time.Second) && tok.Error() != nil {
		t.Fatal(tok.Error())
	}
	return c
}

func TestPublishEntityStatesPublishesNumberState(test *testing.T) {
	t := setupMqttTest(test, nil)
	defer t.teardown()

	cfg := &Config{
		Mqtt: MqttConfig{
			Enabled: true,
			Topic:   "/mqttrooper/test",
		},
		Entities: map[string]EntityConfig{
			"volume": {
				Type: EntityTypeNumber,
				Min:  0, Max: 100, Step: 1,
				Get: "echo 42",
				Set: "echo {value}",
			},
		},
		Executor: ExecutorConfig{Shell: "/bin/bash", DryRun: false},
	}

	received := make(chan mqtt.Message, 4)
	sub := newClient(t, "state-sub")
	defer sub.Disconnect(250)
	sub.Subscribe("/mqttrooper/test/number/volume/state", 0, func(_ mqtt.Client, m mqtt.Message) {
		received <- m
	})
	time.Sleep(100 * time.Millisecond)

	pub := newClient(t, "state-pub")
	defer pub.Disconnect(250)

	assert.NoError(t, PublishEntityStates(pub, cfg, cfg.Executor.Shell, cfg.Executor.DryRun))

	select {
	case m := <-received:
		assert.Equal(t, "42", string(m.Payload()))
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for entity state")
	}

	// Verify retained by reconnecting fresh subscriber.
	retainedCh := make(chan mqtt.Message, 2)
	lateSub := newClient(t, "state-late-sub")
	defer lateSub.Disconnect(250)
	lateSub.Subscribe("/mqttrooper/test/number/volume/state", 0, func(_ mqtt.Client, m mqtt.Message) {
		retainedCh <- m
	})
	select {
	case m := <-retainedCh:
		assert.Equal(t, "42", string(m.Payload()))
		assert.True(t, m.Retained())
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for retained state replay")
	}
}

func TestSubscribeEntitiesHandlesNumberSet(test *testing.T) {
	t := setupMqttTest(test, nil)
	defer t.teardown()

	var mu sync.Mutex
	var executedCmd string
	var statePayloads []string

	cfg := &Config{
		Mqtt: MqttConfig{
			Enabled: true,
			Topic:   "/mqttrooper/test",
		},
		Entities: map[string]EntityConfig{
			"volume": {
				Type: EntityTypeNumber,
				Min:  0, Max: 100, Step: 1,
				Get: "echo 75",
				Set: "echo set-{value}",
			},
		},
		Executor: ExecutorConfig{Shell: "/bin/bash", DryRun: false},
	}

	stateSub := newClient(t, "num-state-sub")
	defer stateSub.Disconnect(250)
	stateSub.Subscribe("/mqttrooper/test/number/volume/state", 0, func(_ mqtt.Client, m mqtt.Message) {
		mu.Lock()
		statePayloads = append(statePayloads, string(m.Payload()))
		mu.Unlock()
	})
	time.Sleep(100 * time.Millisecond)

	daemon := newClient(t, "num-daemon")
	defer daemon.Disconnect(250)

	assert.NoError(t, SubscribeEntities(daemon, cfg, cfg.Executor.Shell, cfg.Executor.DryRun))
	time.Sleep(100 * time.Millisecond)

	// HA publishes a new value
	pub := newClient(t, "num-pub")
	defer pub.Disconnect(250)
	tok := pub.Publish("/mqttrooper/test/number/volume/set", 0, false, "80")
	tok.Wait()
	assert.NoError(t, tok.Error())

	time.Sleep(500 * time.Millisecond)
	_ = executedCmd

	mu.Lock()
	defer mu.Unlock()
	assert.NotEmpty(t, statePayloads, "expected state published after set")
	// get command returns "75", that's the state published
	assert.Equal(t, "75", statePayloads[len(statePayloads)-1])
}
