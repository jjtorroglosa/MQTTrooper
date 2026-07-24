package internal

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type executeCall struct {
	name  string
	op    EntityOperation
	value *string
}

type spyEntityExecutor struct {
	mu        sync.Mutex
	calls     []executeCall
	responses map[string]string
	fn        func(name string, op EntityOperation, value *string) (string, error)
}

func (s *spyEntityExecutor) Execute(name string, op EntityOperation, value *string) (string, error) {
	s.mu.Lock()
	s.calls = append(s.calls, executeCall{name: name, op: op, value: value})
	s.mu.Unlock()

	if s.fn != nil {
		return s.fn(name, op, value)
	}
	if s.responses != nil {
		if v, ok := s.responses[fmt.Sprintf("%s:%s", name, op)]; ok {
			return v, nil
		}
	}
	return "", nil
}

func (s *spyEntityExecutor) Calls() []executeCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]executeCall, len(s.calls))
	copy(out, s.calls)
	return out
}

type fakeToken struct {
	err  error
	done chan struct{}
}

func newFakeToken(err error) *fakeToken {
	return &fakeToken{err: err, done: closedChan()}
}

func closedChan() chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

func (t *fakeToken) Wait() bool                     { return true }
func (t *fakeToken) WaitTimeout(time.Duration) bool { return true }
func (t *fakeToken) Done() <-chan struct{}          { return t.done }
func (t *fakeToken) Error() error                   { return t.err }

type fakeMessage struct {
	topic    string
	payload  []byte
	retained bool
}

func (m fakeMessage) Duplicate() bool   { return false }
func (m fakeMessage) Qos() byte         { return 0 }
func (m fakeMessage) Retained() bool    { return m.retained }
func (m fakeMessage) Topic() string     { return m.topic }
func (m fakeMessage) MessageID() uint16 { return 0 }
func (m fakeMessage) Payload() []byte   { return m.payload }
func (m fakeMessage) Ack()              {}

type fakeBroker struct {
	mu       sync.Mutex
	retained map[string][]byte
	subs     map[string][]mqtt.MessageHandler
}

func newFakeBroker() *fakeBroker {
	return &fakeBroker{
		retained: map[string][]byte{},
		subs:     map[string][]mqtt.MessageHandler{},
	}
}

type fakeClient struct {
	broker *fakeBroker
}

func newFakeClient(b *fakeBroker) mqtt.Client {
	return &fakeClient{broker: b}
}

func (c *fakeClient) IsConnected() bool      { return true }
func (c *fakeClient) IsConnectionOpen() bool { return true }
func (c *fakeClient) Connect() mqtt.Token    { return newFakeToken(nil) }
func (c *fakeClient) Disconnect(uint)        {}

func payloadBytes(payload interface{}) []byte {
	switch v := payload.(type) {
	case []byte:
		return append([]byte(nil), v...)
	case string:
		return []byte(v)
	default:
		return []byte(fmt.Sprint(v))
	}
}

func (c *fakeClient) Publish(topic string, _ byte, retained bool, payload interface{}) mqtt.Token {
	body := payloadBytes(payload)
	c.broker.mu.Lock()
	if retained {
		if len(body) == 0 {
			delete(c.broker.retained, topic)
		} else {
			c.broker.retained[topic] = body
		}
	}
	handlers := append([]mqtt.MessageHandler(nil), c.broker.subs[topic]...)
	c.broker.mu.Unlock()

	msg := fakeMessage{topic: topic, payload: body, retained: retained}
	for _, h := range handlers {
		h(c, msg)
	}
	return newFakeToken(nil)
}

func (c *fakeClient) Subscribe(topic string, _ byte, callback mqtt.MessageHandler) mqtt.Token {
	c.broker.mu.Lock()
	c.broker.subs[topic] = append(c.broker.subs[topic], callback)
	retained, ok := c.broker.retained[topic]
	c.broker.mu.Unlock()

	if ok {
		callback(c, fakeMessage{topic: topic, payload: append([]byte(nil), retained...), retained: true})
	}
	return newFakeToken(nil)
}

func (c *fakeClient) SubscribeMultiple(filters map[string]byte, callback mqtt.MessageHandler) mqtt.Token {
	for topic := range filters {
		if tok := c.Subscribe(topic, filters[topic], callback); tok.Error() != nil {
			return tok
		}
	}
	return newFakeToken(nil)
}

func (c *fakeClient) Unsubscribe(topics ...string) mqtt.Token {
	c.broker.mu.Lock()
	defer c.broker.mu.Unlock()
	for _, topic := range topics {
		delete(c.broker.subs, topic)
	}
	return newFakeToken(nil)
}

func (c *fakeClient) AddRoute(string, mqtt.MessageHandler) {}

func (c *fakeClient) OptionsReader() mqtt.ClientOptionsReader { return mqtt.ClientOptionsReader{} }

func TestCreateEntityExecutorRunsGetCommand(t *testing.T) {
	exec := CreateEntityExecutor(false, "/bin/bash", map[string]EntityConfig{
		"volume": {Type: EntityTypeNumber, Get: "echo 42"},
	})

	out, err := exec.Execute("volume", EntityOpGet, nil)
	require.NoError(t, err)
	assert.Equal(t, "42", out)
}

func TestCreateEntityExecutorSubstitutesSetValue(t *testing.T) {
	exec := CreateEntityExecutor(false, "/bin/bash", map[string]EntityConfig{
		"volume": {Type: EntityTypeNumber, Set: "echo set-{value}"},
	})

	value := "80"
	out, err := exec.Execute("volume", EntityOpSet, &value)
	require.NoError(t, err)
	assert.Equal(t, "set-80", out)
}

func TestCreateEntityExecutorDryRunSkipsExecution(t *testing.T) {
	filename := fmt.Sprintf("/tmp/mqttrooper-entity-%d", time.Now().UnixNano())
	defer func() { _ = os.Remove(filename) }()

	exec := CreateEntityExecutor(true, "/bin/bash", map[string]EntityConfig{
		"volume": {Type: EntityTypeNumber, Set: fmt.Sprintf("echo should-not-run > %s", filename)},
	})

	value := "80"
	out, err := exec.Execute("volume", EntityOpSet, &value)
	require.NoError(t, err)
	assert.Empty(t, out)
	_, statErr := os.Stat(filename)
	assert.Error(t, statErr)
}

func TestPublishEntityStatesPublishesNumberState(t *testing.T) {
	broker := newFakeBroker()
	cfg := &Config{
		Mqtt: MqttConfig{Enabled: true, Topic: "/mqttrooper/test"},
		Entities: map[string]EntityConfig{
			"volume": {Type: EntityTypeNumber, Get: "echo 42", Set: "echo {value}"},
		},
	}

	stateExec := &spyEntityExecutor{responses: map[string]string{"volume:get": "42"}}

	received := make(chan mqtt.Message, 1)
	sub := newFakeClient(broker)
	sub.Subscribe("/mqttrooper/test/number/volume/state", 0, func(_ mqtt.Client, m mqtt.Message) {
		received <- m
	})

	pub := newFakeClient(broker)
	assert.NoError(t, PublishEntityStates(pub, cfg, stateExec))

	select {
	case m := <-received:
		assert.Equal(t, "42", string(m.Payload()))
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for entity state")
	}

	calls := stateExec.Calls()
	require.Len(t, calls, 1)
	assert.Equal(t, "volume", calls[0].name)
	assert.Equal(t, EntityOpGet, calls[0].op)
	assert.Nil(t, calls[0].value)

	retainedCh := make(chan mqtt.Message, 1)
	lateSub := newFakeClient(broker)
	lateSub.Subscribe("/mqttrooper/test/number/volume/state", 0, func(_ mqtt.Client, m mqtt.Message) {
		retainedCh <- m
	})
	select {
	case m := <-retainedCh:
		assert.Equal(t, "42", string(m.Payload()))
		assert.True(t, m.Retained())
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for retained state replay")
	}
}

func TestSubscribeEntitiesHandlesNumberSet(t *testing.T) {
	broker := newFakeBroker()
	var mu sync.Mutex
	var statePayloads []string
	executor := &spyEntityExecutor{
		fn: func(name string, op EntityOperation, value *string) (string, error) {
			switch op {
			case EntityOpSet:
				require.NotNil(t, value)
				assert.Equal(t, "80", *value)
				return "", nil
			case EntityOpGet:
				return "75", nil
			default:
				t.Fatalf("unexpected op: %s", op)
				return "", nil
			}
		},
	}

	cfg := &Config{
		Mqtt: MqttConfig{Enabled: true, Topic: "/mqttrooper/test"},
		Entities: map[string]EntityConfig{
			"volume": {Type: EntityTypeNumber, Get: "echo 75", Set: "echo set-{value}"},
		},
	}

	stateSub := newFakeClient(broker)
	stateSub.Subscribe("/mqttrooper/test/number/volume/state", 0, func(_ mqtt.Client, m mqtt.Message) {
		mu.Lock()
		statePayloads = append(statePayloads, string(m.Payload()))
		mu.Unlock()
	})

	daemon := newFakeClient(broker)
	assert.NoError(t, SubscribeEntities(daemon, cfg, executor))

	pub := newFakeClient(broker)
	tok := pub.Publish("/mqttrooper/test/number/volume/set", 0, false, "80")
	tok.Wait()
	assert.NoError(t, tok.Error())

	calls := executor.Calls()
	require.Len(t, calls, 2)
	assert.Equal(t, EntityOpSet, calls[0].op)
	assert.Equal(t, "80", *calls[0].value)
	assert.Equal(t, EntityOpGet, calls[1].op)
	assert.Nil(t, calls[1].value)

	mu.Lock()
	defer mu.Unlock()
	assert.NotEmpty(t, statePayloads, "expected state published after set")
	assert.Equal(t, "75", statePayloads[len(statePayloads)-1])
}

func TestPublishEntityStatesPublishesBooleanState(t *testing.T) {
	for _, tc := range []struct {
		getCmd   string
		expected string
	}{
		{"echo yes", "ON"},
		{"echo 0", "OFF"},
	} {
		broker := newFakeBroker()
		cfg := &Config{
			Mqtt: MqttConfig{Enabled: true, Topic: "/mqttrooper/test"},
			Entities: map[string]EntityConfig{
				"mute": {Type: EntityTypeSwitch, Get: tc.getCmd, On: "echo on", Off: "echo off"},
			},
		}

		stateExec := &spyEntityExecutor{responses: map[string]string{"mute:get": strings.TrimSpace(strings.TrimPrefix(tc.getCmd, "echo "))}}

		received := make(chan mqtt.Message, 1)
		sub := newFakeClient(broker)
		sub.Subscribe("/mqttrooper/test/switch/mute/state", 0, func(_ mqtt.Client, m mqtt.Message) {
			received <- m
		})

		pub := newFakeClient(broker)
		assert.NoError(t, PublishEntityStates(pub, cfg, stateExec))

		select {
		case m := <-received:
			assert.Equal(t, tc.expected, string(m.Payload()))
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for boolean entity state")
		}

		calls := stateExec.Calls()
		require.Len(t, calls, 1)
		assert.Equal(t, EntityOpGet, calls[0].op)
		assert.Nil(t, calls[0].value)
	}
}

func TestSubscribeEntitiesHandlesBooleanSet(t *testing.T) {
	broker := newFakeBroker()
	var mu sync.Mutex
	var executed []EntityOperation
	var statePayloads []string
	executor := &spyEntityExecutor{
		fn: func(name string, op EntityOperation, value *string) (string, error) {
			mu.Lock()
			executed = append(executed, op)
			mu.Unlock()
			switch op {
			case EntityOpOn:
				return "", nil
			case EntityOpOff:
				return "", nil
			case EntityOpGet:
				return "yes", nil
			default:
				return "", nil
			}
		},
	}

	cfg := &Config{
		Mqtt: MqttConfig{Enabled: true, Topic: "/mqttrooper/test"},
		Entities: map[string]EntityConfig{
			"mute": {Type: EntityTypeSwitch, Get: "echo yes", On: "echo turning-on", Off: "echo turning-off"},
		},
	}

	stateSub := newFakeClient(broker)
	stateSub.Subscribe("/mqttrooper/test/switch/mute/state", 0, func(_ mqtt.Client, m mqtt.Message) {
		mu.Lock()
		statePayloads = append(statePayloads, string(m.Payload()))
		mu.Unlock()
	})

	daemon := newFakeClient(broker)
	assert.NoError(t, SubscribeEntities(daemon, cfg, executor))

	pub := newFakeClient(broker)

	// Send ON
	tok := pub.Publish("/mqttrooper/test/switch/mute/set", 0, false, "ON")
	tok.Wait()
	assert.NoError(t, tok.Error())

	// Send OFF
	tok = pub.Publish("/mqttrooper/test/switch/mute/set", 0, false, "OFF")
	tok.Wait()
	assert.NoError(t, tok.Error())

	// Send invalid
	tok = pub.Publish("/mqttrooper/test/switch/mute/set", 0, false, "INVALID")
	tok.Wait()
	assert.NoError(t, tok.Error())

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []EntityOperation{EntityOpOn, EntityOpGet, EntityOpOff, EntityOpGet}, executed)
	assert.Len(t, statePayloads, 2, "expected state published for ON and OFF only")
	assert.Equal(t, "ON", statePayloads[0])
	assert.Equal(t, "ON", statePayloads[1]) // Get always returns "yes" → "ON"
}
